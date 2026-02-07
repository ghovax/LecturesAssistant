package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"lectures/internal/configuration"
	"lectures/internal/llm"
	"lectures/internal/markdown"
	"lectures/internal/models"
	"lectures/internal/prompts"
)

type ToolGenerator struct {
	configuration *configuration.Configuration
	llmProvider   llm.Provider
	promptManager *prompts.Manager
}

func NewToolGenerator(configuration *configuration.Configuration, llmProvider llm.Provider, promptManager *prompts.Manager) *ToolGenerator {
	return &ToolGenerator{
		configuration: configuration,
		llmProvider:   llmProvider,
		promptManager: promptManager,
	}
}

// GenerateStudyGuide implements the self-healing sequential generation pipeline
func (generator *ToolGenerator) GenerateStudyGuide(
	jobContext context.Context,
	lecture models.Lecture,
	transcript string,
	referenceFilesContent string,
	length string,
	languageCode string,
	options models.GenerationOptions,
	updateProgress func(int, string, any, models.JobMetrics),
) (string, string, error) {
	var totalMetrics models.JobMetrics

	// PHASE 2: Documents Matching
	updateProgress(5, "Matching relevant reference materials...", nil, totalMetrics)
	relevantMaterials := referenceFilesContent
	if options.EnableDocumentsMatching && referenceFilesContent != "" {
		materials, metrics, err := generator.matchRelevantDocuments(jobContext, transcript, referenceFilesContent, options)
		if err == nil {
			relevantMaterials = materials
			totalMetrics.InputTokens += metrics.InputTokens
			totalMetrics.OutputTokens += metrics.OutputTokens
			totalMetrics.EstimatedCost += metrics.EstimatedCost
		} else {
			slog.Warn("Documents matching failed, falling back to full content", "error", err)
		}
	}

	// PHASE 3: Sequential Generation
	updateProgress(10, "Analyzing lecture structure...", nil, totalMetrics)

	// 3.1 Analyze Structure with Retries
	structure, metrics, err := generator.analyzeStructureWithRetries(jobContext, transcript, relevantMaterials, length, languageCode, options)
	if err != nil {
		return "", "", fmt.Errorf("structure analysis failed: %w", err)
	}
	totalMetrics.InputTokens += metrics.InputTokens
	totalMetrics.OutputTokens += metrics.OutputTokens
	totalMetrics.EstimatedCost += metrics.EstimatedCost

	// 3.2 Sequential Building
	updateProgress(15, "Building study guide sections...", nil, totalMetrics)
	finalMarkdown, finalTitle, genMetrics, err := generator.generateSequentialStudyGuide(jobContext, lecture, transcript, relevantMaterials, structure, languageCode, options, updateProgress, totalMetrics)
	if err != nil {
		return "", "", fmt.Errorf("sequential generation failed: %w", err)
	}
	totalMetrics.InputTokens += genMetrics.InputTokens
	totalMetrics.OutputTokens += genMetrics.OutputTokens
	totalMetrics.EstimatedCost += genMetrics.EstimatedCost

	updateProgress(100, "Generation complete.", nil, totalMetrics)
	return finalMarkdown, finalTitle, nil
}

func (generator *ToolGenerator) matchRelevantDocuments(jobContext context.Context, transcript, fullMaterials string, options models.GenerationOptions) (string, models.JobMetrics, error) {
	if generator.llmProvider == nil {
		return fullMaterials, models.JobMetrics{}, nil
	}

	var waitGroup sync.WaitGroup
	var mutex sync.Mutex
	var allMetrics models.JobMetrics
	var allRanges [][]struct {
		Start int `json:"start"`
		End   int `json:"end"`
	}

	model := options.ModelDocumentsMatching
	if model == "" {
		model = generator.configuration.LLM.GetModelForTask("documents_matching")
	}

	maximumRetries := options.MaximumRetries
	if maximumRetries <= 0 {
		maximumRetries = generator.configuration.Safety.MaximumRetries
		if maximumRetries <= 0 {
			maximumRetries = 3
		}
	}

	for attemptIndex := 0; attemptIndex < maximumRetries; attemptIndex++ {
		waitGroup.Add(1)
		go func(attemptIndex int) {
			defer waitGroup.Done()

			var prompt string
			if generator.promptManager != nil {
				prompt, _ = generator.promptManager.GetPrompt(prompts.PromptGetRelevantPages, map[string]string{
					"transcript":      transcript,
					"reference_files": fullMaterials,
				})
			}

			response, stepMetrics, err := generator.callLLMWithModel(jobContext, prompt, model)
			mutex.Lock()
			defer mutex.Unlock()
			allMetrics.InputTokens += stepMetrics.InputTokens
			allMetrics.OutputTokens += stepMetrics.OutputTokens
			allMetrics.EstimatedCost += stepMetrics.EstimatedCost

			if err == nil {
				var result struct {
					PageRanges []struct {
						Start int `json:"start"`
						End   int `json:"end"`
					} `json:"page_ranges"`
				}
				if unmarshalErr := generator.unmarshalJSONWithFallback(response, &result); unmarshalErr == nil {
					allRanges = append(allRanges, result.PageRanges)
				}
			}
		}(attemptIndex)
	}
	waitGroup.Wait()

	if len(allRanges) == 0 {
		return fullMaterials, allMetrics, fmt.Errorf("all document matching runs failed")
	}

	finalRanges := generator.unionAndMergeRanges(allRanges)
	return generator.filterMaterialsByRanges(fullMaterials, finalRanges), allMetrics, nil
}

func (generator *ToolGenerator) analyzeStructureWithRetries(jobContext context.Context, transcript, materials, length, language string, options models.GenerationOptions) (string, models.JobMetrics, error) {
	if generator.llmProvider == nil {
		return "", models.JobMetrics{}, fmt.Errorf("llm provider is nil")
	}

	var metrics models.JobMetrics
	var sectionCounts struct {
		minimum, maximum int
		preferred        string
	}

	switch length {
	case "short":
		sectionCounts.minimum, sectionCounts.maximum, sectionCounts.preferred = 1, 4, "2-3"
	case "long":
		sectionCounts.minimum, sectionCounts.maximum, sectionCounts.preferred = 4, 7, "5-6"
	default:
		sectionCounts.minimum, sectionCounts.maximum, sectionCounts.preferred = 2, 5, "3-4"
	}

	var prompt string
	if generator.promptManager != nil {
		latexInstructions, _ := generator.promptManager.GetPrompt(prompts.PromptLatexInstructions, nil)
		exampleTemplatePath := prompts.PromptStudyGuideWithoutCitationsExample
		if materials != "" {
			exampleTemplatePath = prompts.PromptStudyGuideWithCitationsExample
		}
		exampleTemplate, _ := generator.promptManager.GetPrompt(exampleTemplatePath, nil)

		prompt, _ = generator.promptManager.GetPrompt(prompts.PromptAnalyzeLectureStructure, map[string]string{
			"language_requirement":    fmt.Sprintf("Use language code %s", language),
			"minimum_section_count":   strconv.Itoa(sectionCounts.minimum),
			"maximum_section_count":   strconv.Itoa(sectionCounts.maximum),
			"preferred_section_range": sectionCounts.preferred,
			"latex_instructions":      latexInstructions,
			"example_template":        exampleTemplate,
			"transcript":              transcript,
			"reference_materials":     materials,
		})
	}

	model := options.ModelStructure
	if model == "" {
		model = generator.configuration.LLM.GetModelForTask("outline_creation")
	}

	maximumRetries := options.MaximumRetries
	if maximumRetries <= 0 {
		maximumRetries = generator.configuration.Safety.MaximumRetries
		if maximumRetries <= 0 {
			maximumRetries = 3
		}
	}

	slog.Info("Starting structure analysis", "model", model, "maximum_retries", maximumRetries)

	for attempt := 1; attempt <= maximumRetries; attempt++ {
		slog.Debug("Structure analysis attempt", "attempt", attempt, "of", maximumRetries)

		response, stepMetrics, err := generator.callLLMWithModel(jobContext, prompt, model)
		metrics.InputTokens += stepMetrics.InputTokens
		metrics.OutputTokens += stepMetrics.OutputTokens
		metrics.EstimatedCost += stepMetrics.EstimatedCost

		slog.Debug("LLM response received",
			"attempt", attempt,
			"input_tokens", stepMetrics.InputTokens,
			"output_tokens", stepMetrics.OutputTokens,
			"cost", stepMetrics.EstimatedCost)

		if err != nil {
			slog.Error("LLM call failed", "attempt", attempt, "error", err)
			if attempt == maximumRetries {
				return "", metrics, err
			}
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}

		sections := generator.parseStructure(response)
		slog.Info("Structure parsed",
			"attempt", attempt,
			"sections_found", len(sections),
			"required_minimum", sectionCounts.minimum,
			"required_maximum", sectionCounts.maximum)

		if len(sections) >= sectionCounts.minimum && len(sections) <= sectionCounts.maximum {
			slog.Info("Structure validation passed", "sections", len(sections))

			// Clean the title before returning
			title := generator.parseTitle(response)
			slog.Debug("Cleaning document title", "original_title", title)

			cleanedTitle, titleMetrics, err := generator.CleanDocumentTitle(jobContext, title, options)
			if err == nil {
				metrics.InputTokens += titleMetrics.InputTokens
				metrics.OutputTokens += titleMetrics.OutputTokens
				metrics.EstimatedCost += titleMetrics.EstimatedCost

				slog.Debug("Title cleaned",
					"original", title,
					"cleaned", cleanedTitle,
					"changed", cleanedTitle != title)

				// Replace the title in the response if it changed
				if cleanedTitle != title && title != "" {
					response = strings.Replace(response, "# "+title, "# "+cleanedTitle, 1)
				}
			} else {
				slog.Warn("Title cleaning failed", "error", err)
			}

			slog.Info("Structure analysis complete",
				"final_sections", len(sections),
				"total_cost", metrics.EstimatedCost)
			return response, metrics, nil
		}
		slog.Warn("Structure validation failed, retrying...",
			"count", len(sections),
			"attempt", attempt,
			"expected_range", fmt.Sprintf("%d-%d", sectionCounts.minimum, sectionCounts.maximum))
	}

	return "", metrics, fmt.Errorf("failed to generate valid structure after %d attempts", maximumRetries)
}

func (generator *ToolGenerator) generateSequentialStudyGuide(
	jobContext context.Context,
	lecture models.Lecture,
	transcript, materials, structure, language string,
	options models.GenerationOptions,
	updateProgress func(int, string, any, models.JobMetrics),
	currentMetrics models.JobMetrics,
) (string, string, models.JobMetrics, error) {
	var metrics models.JobMetrics
	title := generator.parseTitle(structure)
	if title == "" {
		title = lecture.Title
	}
	sections := generator.parseStructure(structure)

	initialContextTemplate, _ := generator.promptManager.GetPrompt(prompts.PromptStudyGuideInitialContext, nil)
	languageRequirement, _ := generator.promptManager.GetPrompt(prompts.PromptLanguageRequirement, map[string]string{"language": language, "bcp_47_lang_code": language})
	latexInstructions, _ := generator.promptManager.GetPrompt(prompts.PromptLatexInstructions, nil)

	citationInstructions := ""
	exampleTemplatePrompt := prompts.PromptSectionWithoutCitationsExample
	if materials != "" {
		citationInstructions, _ = generator.promptManager.GetPrompt(prompts.PromptCitationInstructions, nil)
		exampleTemplatePrompt = prompts.PromptSectionWithCitationsExample
	}
	exampleTemplate, _ := generator.promptManager.GetPrompt(exampleTemplatePrompt, nil)

	initialContext := generator.replacePromptVariables(initialContextTemplate, map[string]string{
		"language_requirement": languageRequirement,
		"transcript":           transcript,
		"reference_materials":  materials,
		"structure_outline":    structure,
	})

	var successfulSections []string
	reconstructor := markdown.NewReconstructor()
	rootNode := &markdown.Node{Type: markdown.NodeDocument}
	rootNode.Children = append(rootNode.Children, &markdown.Node{Type: markdown.NodeHeading, Level: 1, Content: title})

	threshold := options.AdherenceThreshold
	if threshold <= 0 {
		threshold = 70
	}

	maximumRetries := options.MaximumRetries
	if maximumRetries <= 0 {
		maximumRetries = generator.configuration.Safety.MaximumRetries
		if maximumRetries <= 0 {
			maximumRetries = 3
		}
	}

	generationModel := options.ModelGeneration
	if generationModel == "" {
		generationModel = generator.configuration.LLM.GetModelForTask("content_generation")
	}

	adherenceModel := options.ModelAdherence
	if adherenceModel == "" {
		adherenceModel = generator.configuration.LLM.GetModelForTask("content_verification")
	}

	slog.Info("Starting sequential section generation",
		"total_sections", len(sections),
		"model", generationModel,
		"adherence_model", adherenceModel,
		"threshold", threshold)

	for sectionIndex, section := range sections {
		sectionNumber := sectionIndex + 1
		progress := 20 + int(float64(sectionIndex)/float64(len(sections))*75)

		slog.Info("Generating section",
			"section_number", sectionNumber,
			"total_sections", len(sections),
			"title", section.Title,
			"progress", progress)

		sectionPromptTemplate, _ := generator.promptManager.GetPrompt(prompts.PromptStudyGuideSectionGeneration, nil)
		sectionPrompt := generator.replacePromptVariables(sectionPromptTemplate, map[string]string{
			"language_requirement":  languageRequirement,
			"section_title":         section.Title,
			"section_coverage":      section.Coverage,
			"structure_outline":     structure,
			"citation_instructions": citationInstructions,
			"latex_instructions":    latexInstructions,
			"example_template":      exampleTemplate,
		})

		var acceptedContent string
		for attempt := 1; attempt <= maximumRetries; attempt++ {
			slog.Debug("Section generation attempt",
				"section", sectionNumber,
				"attempt", attempt,
				"maximum_retries", maximumRetries)
			history := []llm.Message{
				{Role: "user", Content: []llm.ContentPart{{Type: "text", Text: initialContext}}},
				{Role: "assistant", Content: []llm.ContentPart{{Type: "text", Text: "Ready."}}},
			}
			for i, content := range successfulSections {
				history = append(history, llm.Message{Role: "user", Content: []llm.ContentPart{{Type: "text", Text: "Generate " + sections[i].Title}}})
				history = append(history, llm.Message{Role: "assistant", Content: []llm.ContentPart{{Type: "text", Text: content}}})
			}

			updateProgress(progress, fmt.Sprintf("Generating section %d/%d...", sectionNumber, len(sections)), nil, currentMetrics)

			slog.Debug("Calling LLM for section generation",
				"section", sectionNumber,
				"attempt", attempt,
				"history_messages", len(history))

			response, generationMetrics, err := generator.callLLMWithHistoryAndModel(jobContext, sectionPrompt, history, generationModel)
			metrics.InputTokens += generationMetrics.InputTokens
			metrics.OutputTokens += generationMetrics.OutputTokens
			metrics.EstimatedCost += generationMetrics.EstimatedCost

			slog.Debug("Section generation response received",
				"section", sectionNumber,
				"attempt", attempt,
				"input_tokens", generationMetrics.InputTokens,
				"output_tokens", generationMetrics.OutputTokens,
				"cost", generationMetrics.EstimatedCost)

			if err != nil {
				slog.Error("Section generation failed",
					"section", sectionNumber,
					"attempt", attempt,
					"error", err)
				continue
			}

			sectionParser := markdown.NewParser()
			sectionAST := sectionParser.Parse(response)
			similarity := 0.0

			// Find the actual generated title from the AST
			generatedTitle := ""
			for _, child := range sectionAST.Children {
				if child.Type == markdown.NodeSection && child.Level == 2 {
					generatedTitle = child.Title
					break
				}
				if child.Type == markdown.NodeHeading && child.Level == 2 {
					generatedTitle = child.Content
					break
				}
			}

			if generatedTitle != "" {
				similarity = generator.calculateSimilarity(section.Title, generatedTitle)
				slog.Debug("Title similarity check",
					"section", sectionNumber,
					"expected", section.Title,
					"generated", generatedTitle,
					"similarity", similarity)
			}

			if similarity < 65 && attempt < maximumRetries {
				slog.Warn("Title similarity too low, retrying",
					"section", sectionNumber,
					"similarity", similarity,
					"threshold", 65,
					"attempt", attempt)
				continue
			}

			updateProgress(progress, "Verifying adherence...", nil, currentMetrics)
			slog.Debug("Starting adherence verification",
				"section", sectionNumber,
				"attempt", attempt)

			verificationTemplate, _ := generator.promptManager.GetPrompt(prompts.PromptVerifySectionAdherence, nil)
			verificationPrompt := generator.replacePromptVariables(verificationTemplate, map[string]string{
				"section_title": section.Title, "expected_coverage": section.Coverage, "generated_section": response,
			})
			verificationResponse, verificationMetrics, _ := generator.callLLMWithModel(jobContext, verificationPrompt, adherenceModel)
			metrics.InputTokens += verificationMetrics.InputTokens
			metrics.OutputTokens += verificationMetrics.OutputTokens
			metrics.EstimatedCost += verificationMetrics.EstimatedCost

			adherenceScore := generator.parseScore(verificationResponse)
			slog.Info("Adherence verification complete",
				"section", sectionNumber,
				"attempt", attempt,
				"score", adherenceScore,
				"threshold", threshold,
				"passed", adherenceScore >= threshold)

			if adherenceScore >= threshold || attempt == maximumRetries {
				slog.Info("Section accepted",
					"section", sectionNumber,
					"attempt", attempt,
					"adherence_score", adherenceScore,
					"forced_accept", attempt == maximumRetries && adherenceScore < threshold)

				acceptedContent = response
				rootNode.Children = append(rootNode.Children, sectionAST.Children...)
				successfulSections = append(successfulSections, response)
				break
			}

			slog.Warn("Section rejected, retrying",
				"section", sectionNumber,
				"attempt", attempt,
				"score", adherenceScore,
				"threshold", threshold)

			updateProgress(progress, "Rebuilding history for retry...", nil, currentMetrics)
		}

		if acceptedContent == "" {
			slog.Error("Section generation failed after all retries",
				"section", sectionNumber,
				"title", section.Title,
				"attempts", maximumRetries)
			return "", "", metrics, fmt.Errorf("failed to generate section %d", sectionNumber)
		}
	}

	slog.Info("Sequential generation complete",
		"total_sections", len(sections),
		"successful_sections", len(successfulSections),
		"total_input_tokens", metrics.InputTokens,
		"total_output_tokens", metrics.OutputTokens,
		"total_cost", metrics.EstimatedCost)

	return reconstructor.Reconstruct(rootNode), title, metrics, nil
}

func (generator *ToolGenerator) ProcessFootnotesAI(jobContext context.Context, citations []markdown.ParsedCitation, options models.GenerationOptions) ([]markdown.ParsedCitation, models.JobMetrics, error) {
	if len(citations) == 0 {
		return nil, models.JobMetrics{}, nil
	}

	var totalMetrics models.JobMetrics
	updatedCitations := make([]markdown.ParsedCitation, len(citations))
	copy(updatedCitations, citations)

	model := options.ModelPolishing
	if model == "" {
		model = generator.configuration.LLM.GetModelForTask("content_polishing")
	}

	for citationIndex := 0; citationIndex < len(citations); citationIndex += 10 {
		end := citationIndex + 10
		if end > len(citations) {
			end = len(citations)
		}
		batch := citations[citationIndex:end]

		batchMetrics, err := generator.processFootnoteBatch(jobContext, batch, updatedCitations, citationIndex, model, model)
		totalMetrics.InputTokens += batchMetrics.InputTokens
		totalMetrics.OutputTokens += batchMetrics.OutputTokens
		totalMetrics.EstimatedCost += batchMetrics.EstimatedCost
		if err != nil {
			slog.Error("Footnote batch failed", "error", err)
		}
	}

	return updatedCitations, totalMetrics, nil
}

func (generator *ToolGenerator) processFootnoteBatch(jobContext context.Context, batch []markdown.ParsedCitation, allCitations []markdown.ParsedCitation, offset int, parsingModel, formattingModel string) (models.JobMetrics, error) {
	var metrics models.JobMetrics

	if generator.promptManager == nil {
		// Return batch as-is when promptManager is nil (e.g., in tests)
		for i, citation := range batch {
			allCitations[offset+i] = citation
		}
		return metrics, nil
	}

	latexInstructions, _ := generator.promptManager.GetPrompt(prompts.PromptLatexInstructions, nil)

	var markdownBuilder strings.Builder
	for _, citation := range batch {
		// Include file and page metadata in the footnote for LLM processing
		footnoteText := citation.Description
		if citation.File != "" {
			if len(citation.Pages) > 0 {
				pagePrefix := "p."
				if len(citation.Pages) > 1 {
					pagePrefix = "pp."
				}
				footnoteText = fmt.Sprintf("%s (`%s`, %s %s)", citation.Description, citation.File, pagePrefix, markdown.FormatPageNumbers(citation.Pages))
			} else {
				footnoteText = fmt.Sprintf("%s (`%s`)", citation.Description, citation.File)
			}
		}
		markdownBuilder.WriteString(fmt.Sprintf("[^%d]: %s\n\n", citation.Number, footnoteText))
	}

	parsingPrompt, _ := generator.promptManager.GetPrompt(prompts.PromptParseFootnotes, map[string]string{
		"footnotes": markdownBuilder.String(), "latex_instructions": latexInstructions,
	})
	parsingResponse, parsingMetrics, err := generator.callLLMWithModel(jobContext, parsingPrompt, parsingModel)
	metrics.InputTokens += parsingMetrics.InputTokens
	metrics.OutputTokens += parsingMetrics.OutputTokens
	metrics.EstimatedCost += parsingMetrics.EstimatedCost
	if err != nil {
		return metrics, err
	}

	var result struct {
		Footnotes []struct {
			Number      int    `json:"number"`
			TextContent string `json:"text_content"`
			File        string `json:"file"`
			Pages       []int  `json:"pages"`
		} `json:"footnotes"`
	}
	if unmarshalErr := generator.unmarshalJSONWithFallback(parsingResponse, &result); unmarshalErr == nil {
		for batchCitationIndex, citation := range batch {
			found := false
			for _, aiFootnote := range result.Footnotes {
				if aiFootnote.Number == citation.Number {
					allCitations[offset+batchCitationIndex].File = aiFootnote.File
					allCitations[offset+batchCitationIndex].Pages = aiFootnote.Pages
					found = true
					break
				}
			}
			if !found && batchCitationIndex < len(result.Footnotes) {
				allCitations[offset+batchCitationIndex].File = result.Footnotes[batchCitationIndex].File
				allCitations[offset+batchCitationIndex].Pages = result.Footnotes[batchCitationIndex].Pages
			}
		}
	}

	formattingPrompt, _ := generator.promptManager.GetPrompt(prompts.PromptFormatFootnotes, map[string]string{
		"footnotes": markdownBuilder.String(), "latex_instructions": latexInstructions,
	})
	formattingResponse, formattingMetrics, _ := generator.callLLMWithModel(jobContext, formattingPrompt, formattingModel)
	metrics.InputTokens += formattingMetrics.InputTokens
	metrics.OutputTokens += formattingMetrics.OutputTokens
	metrics.EstimatedCost += formattingMetrics.EstimatedCost

	parser := markdown.NewParser()
	ast := parser.Parse(formattingResponse)

	// Extract all footnotes from the AST
	var aiFootnotes []*markdown.Node
	for _, node := range ast.Children {
		if node.Type == markdown.NodeFootnote {
			aiFootnotes = append(aiFootnotes, node)
		}
	}

	for batchCitationIndex, citation := range batch {
		found := false
		// Try matching by number first
		for _, node := range aiFootnotes {
			if node.FootnoteNumber == citation.Number {
				allCitations[offset+batchCitationIndex].Description = node.Content
				found = true
				break
			}
		}
		// Positional fallback if number matching fails
		if !found && batchCitationIndex < len(aiFootnotes) {
			allCitations[offset+batchCitationIndex].Description = aiFootnotes[batchCitationIndex].Content
		}
	}

	return metrics, nil
}

// unmarshalJSONWithFallback attempts idiomatic parsing then robust extraction
func (generator *ToolGenerator) unmarshalJSONWithFallback(text string, target interface{}) error {
	if err := json.Unmarshal([]byte(strings.TrimSpace(text)), target); err == nil {
		return nil
	}
	fencedJSON := generator.extractFencedJSON(text)
	return json.Unmarshal([]byte(fencedJSON), target)
}

func (generator *ToolGenerator) extractFencedJSON(text string) string {
	start := strings.Index(text, "{")
	if start == -1 {
		start = strings.Index(text, "[")
	}
	end := strings.LastIndex(text, "}")
	if end == -1 {
		end = strings.LastIndex(text, "]")
	}
	if start != -1 && end != -1 && end > start {
		return text[start : end+1]
	}
	return text
}

func (generator *ToolGenerator) filterMaterialsByRanges(materials string, ranges []struct {
	Start int `json:"start"`
	End   int `json:"end"`
}) string {
	parser := markdown.NewParser()
	ast := parser.Parse(materials)
	reconstructor := markdown.NewReconstructor()
	filteredRoot := &markdown.Node{Type: markdown.NodeDocument}

	var currentFile *markdown.Node
	pageRegex := regexp.MustCompile(`(?i)page\s*(\d+)`)

	var processNode func(*markdown.Node)
	processNode = func(node *markdown.Node) {
		if node.Type == markdown.NodeSection || node.Type == markdown.NodeHeading {
			title := node.Title
			if title == "" {
				title = node.Content
			}

			if node.Level == 1 {
				currentFile = &markdown.Node{Type: markdown.NodeHeading, Level: 1, Content: title}
			} else {
				match := pageRegex.FindStringSubmatch(title)
				if match != nil {
					pageNum, _ := strconv.Atoi(match[1])
					isRelevant := false
					for _, pageRange := range ranges {
						if pageNum >= pageRange.Start && pageNum <= pageRange.End {
							isRelevant = true
							break
						}
					}

					if isRelevant {
						if currentFile != nil {
							alreadyAdded := false
							for _, added := range filteredRoot.Children {
								if added.Type == markdown.NodeHeading && added.Level == 1 && added.Content == currentFile.Content {
									alreadyAdded = true
									break
								}
							}
							if !alreadyAdded {
								filteredRoot.Children = append(filteredRoot.Children, currentFile)
							}
						}
						filteredRoot.Children = append(filteredRoot.Children, &markdown.Node{
							Type: markdown.NodeHeading, Level: 2, Content: title,
						})
						filteredRoot.Children = append(filteredRoot.Children, node.Children...)
					}
				}
			}
		}
		for _, child := range node.Children {
			processNode(child)
		}
	}

	processNode(ast)
	return reconstructor.Reconstruct(filteredRoot)
}

func (generator *ToolGenerator) callLLMWithModel(jobContext context.Context, prompt string, model string) (string, models.JobMetrics, error) {
	return generator.callLLMWithHistoryAndModel(jobContext, prompt, nil, model)
}

func (generator *ToolGenerator) callLLMWithHistoryAndModel(jobContext context.Context, prompt string, history []llm.Message, model string) (string, models.JobMetrics, error) {
	if model == "" {
		model = generator.configuration.LLM.Model
	}

	messages := append(history, llm.Message{
		Role: "user", Content: []llm.ContentPart{{Type: "text", Text: prompt}},
	})

	responseChannel, err := generator.llmProvider.Chat(jobContext, llm.ChatRequest{
		Model: model, Messages: messages, Stream: false,
	})
	if err != nil {
		return "", models.JobMetrics{}, err
	}

	var resultBuilder strings.Builder
	var metrics models.JobMetrics
	for chunk := range responseChannel {
		if chunk.Error != nil {
			return "", models.JobMetrics{}, chunk.Error
		}
		resultBuilder.WriteString(chunk.Text)
		metrics.InputTokens += chunk.InputTokens
		metrics.OutputTokens += chunk.OutputTokens
		metrics.EstimatedCost += chunk.Cost
	}

	if generator.configuration.Safety.MaximumCostPerJob > 0 && metrics.EstimatedCost > generator.configuration.Safety.MaximumCostPerJob {
		return "", metrics, fmt.Errorf("safety threshold exceeded: call cost $%.4f > limit $%.4f", metrics.EstimatedCost, generator.configuration.Safety.MaximumCostPerJob)
	}

	return resultBuilder.String(), metrics, nil
}

func (generator *ToolGenerator) replacePromptVariables(prompt string, variables map[string]string) string {
	result := prompt
	for key, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

func (generator *ToolGenerator) parseTitle(structure string) string {
	parser := markdown.NewParser()
	ast := parser.Parse(structure)
	for _, child := range ast.Children {
		if child.Type == markdown.NodeSection && child.Level == 1 {
			slog.Debug("Found title from NodeSection", "title", child.Title)
			return child.Title
		}
		if child.Type == markdown.NodeHeading && child.Level == 1 {
			slog.Debug("Found title from NodeHeading", "content", child.Content)
			return child.Content
		}
	}
	slog.Warn("No title found in structure", "children_count", len(ast.Children))
	return ""
}

func (generator *ToolGenerator) parseStructure(structure string) []sectionInfo {
	parser := markdown.NewParser()
	ast := parser.Parse(structure)
	reconstructor := markdown.NewReconstructor()
	var sections []sectionInfo
	var find func(*markdown.Node)
	find = func(node *markdown.Node) {
		if node.Type == markdown.NodeSection && node.Level == 2 {
			coverage := reconstructor.Reconstruct(&markdown.Node{Type: markdown.NodeDocument, Children: node.Children})
			sections = append(sections, sectionInfo{Title: node.Title, Coverage: coverage})
		} else {
			for _, child := range node.Children {
				find(child)
			}
		}
	}
	find(ast)
	return sections
}

func (generator *ToolGenerator) parseScore(response string) int {
	var result struct {
		CoverageScore int `json:"coverage_score"`
	}
	if err := generator.unmarshalJSONWithFallback(response, &result); err != nil {
		return 0
	}
	return result.CoverageScore
}

func (generator *ToolGenerator) calculateSimilarity(firstString, secondString string) float64 {
	firstString = strings.ToLower(strings.TrimSpace(firstString))
	secondString = strings.ToLower(strings.TrimSpace(secondString))
	if firstString == secondString {
		return 100
	}
	if len(firstString) == 0 || len(secondString) == 0 {
		return 0
	}
	distanceMatrix := make([][]int, len(firstString)+1)
	for rowIndex := range distanceMatrix {
		distanceMatrix[rowIndex] = make([]int, len(secondString)+1)
		distanceMatrix[rowIndex][0] = rowIndex
	}
	for columnIndex := range distanceMatrix[0] {
		distanceMatrix[0][columnIndex] = columnIndex
	}
	for firstStringIndex := 1; firstStringIndex <= len(firstString); firstStringIndex++ {
		for secondStringIndex := 1; secondStringIndex <= len(secondString); secondStringIndex++ {
			cost := 1
			if firstString[firstStringIndex-1] == secondString[secondStringIndex-1] {
				cost = 0
			}
			distanceMatrix[firstStringIndex][secondStringIndex] = generator.minimumInt(distanceMatrix[firstStringIndex-1][secondStringIndex]+1, generator.minimumInt(distanceMatrix[firstStringIndex][secondStringIndex-1]+1, distanceMatrix[firstStringIndex-1][secondStringIndex-1]+cost))
		}
	}
	maximumLength := len(firstString)
	if len(secondString) > maximumLength {
		maximumLength = len(secondString)
	}
	return (1.0 - float64(distanceMatrix[len(firstString)][len(secondString)])/float64(maximumLength)) * 100
}

func (generator *ToolGenerator) minimumInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (generator *ToolGenerator) CleanDocumentTitle(jobContext context.Context, title string, options models.GenerationOptions) (string, models.JobMetrics, error) {
	if title == "" || title == "Untitled Document" || generator.llmProvider == nil {
		return title, models.JobMetrics{}, nil
	}

	var prompt string
	var err error
	if generator.promptManager != nil {
		prompt, err = generator.promptManager.GetPrompt(prompts.PromptCleanDocumentTitle, map[string]string{
			"title": title,
		})
		if err != nil {
			return title, models.JobMetrics{}, err
		}
	}

	model := options.ModelPolishing
	if model == "" {
		model = generator.configuration.LLM.GetModelForTask("content_polishing")
	}

	response, metrics, err := generator.callLLMWithModel(jobContext, prompt, model)
	if err != nil {
		return title, metrics, err
	}

	// Extract title from JSON
	var result struct {
		Title string `json:"title"`
	}
	if err := generator.unmarshalJSONWithFallback(response, &result); err == nil {
		if result.Title != "" {
			return result.Title, metrics, nil
		}
	}

	return title, metrics, nil
}

func (generator *ToolGenerator) CorrectProjectTitleDescription(jobContext context.Context, title, description string, model string) (string, string, models.JobMetrics, error) {
	if title == "" || generator.llmProvider == nil {
		return title, description, models.JobMetrics{}, nil
	}

	if model == "" {
		model = generator.configuration.LLM.GetModelForTask("content_polishing")
		if model == "" {
			model = generator.configuration.LLM.Model
		}
	}

	var prompt string
	var err error
	if generator.promptManager != nil {
		prompt, err = generator.promptManager.GetPrompt(prompts.PromptCorrectProjectTitleDescription, map[string]string{
			"title":       title,
			"description": description,
		})
		if err != nil {
			return title, description, models.JobMetrics{}, err
		}
	}

	response, metrics, err := generator.callLLMWithModel(jobContext, prompt, model)
	if err != nil {
		return title, description, metrics, err
	}

	var result struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := generator.unmarshalJSONWithFallback(response, &result); err == nil {
		return result.Title, result.Description, metrics, nil
	}

	return title, description, metrics, nil
}

func (generator *ToolGenerator) GenerateSuggestedQuestions(jobContext context.Context, documentMarkdown string, model string) ([]string, models.JobMetrics, error) {
	if generator.llmProvider == nil {
		return nil, models.JobMetrics{}, nil
	}

	var prompt string
	if generator.promptManager != nil {
		latexInstructions, _ := generator.promptManager.GetPrompt(prompts.PromptLatexInstructions, nil)
		prompt, _ = generator.promptManager.GetPrompt(prompts.PromptGenerateChatQuestions, map[string]string{
			"document_content":   documentMarkdown,
			"latex_instructions": latexInstructions,
		})
	}

	if model == "" {
		model = generator.configuration.LLM.GetModelForTask("content_polishing")
		if model == "" {
			model = generator.configuration.LLM.Model
		}
	}

	response, metrics, err := generator.callLLMWithModel(jobContext, prompt, model)
	if err != nil {
		return nil, metrics, err
	}

	var result struct {
		Questions []string `json:"questions"`
	}
	if err := generator.unmarshalJSONWithFallback(response, &result); err != nil {
		return nil, metrics, err
	}

	return result.Questions, metrics, nil
}

func (generator *ToolGenerator) GenerateAbstract(jobContext context.Context, documentMarkdown string, languageCode string, model string) (string, models.JobMetrics, error) {
	if generator.llmProvider == nil {
		return "", models.JobMetrics{}, nil
	}

	var prompt string
	if generator.promptManager != nil {
		latexInstructions, _ := generator.promptManager.GetPrompt(prompts.PromptLatexInstructions, nil)
		languageRequirement, _ := generator.promptManager.GetPrompt(prompts.PromptLanguageRequirement, map[string]string{
			"language":         languageCode,
			"bcp_47_lang_code": languageCode,
		})
		prompt, _ = generator.promptManager.GetPrompt(prompts.PromptGenerateDocumentDescription, map[string]string{
			"document_content":     documentMarkdown,
			"latex_instructions":   latexInstructions,
			"language_requirement": languageRequirement,
		})
	}

	if model == "" {
		model = generator.configuration.LLM.GetModelForTask("content_polishing")
		if model == "" {
			model = generator.configuration.LLM.Model
		}
	}

	response, metrics, err := generator.callLLMWithModel(jobContext, prompt, model)
	if err != nil {
		return "", metrics, err
	}

	var result struct {
		Description string `json:"description"`
	}
	if err := generator.unmarshalJSONWithFallback(response, &result); err != nil {
		return "", metrics, err
	}

	return result.Description, metrics, nil
}

type sectionInfo struct {
	Title    string
	Coverage string
}

func (generator *ToolGenerator) GenerateFlashcards(jobContext context.Context, lecture models.Lecture, transcript string, referenceFilesContent string, languageCode string, options models.GenerationOptions, updateProgress func(int, string, any, models.JobMetrics)) (string, string, models.JobMetrics, error) {
	if generator.llmProvider == nil {
		return "", lecture.Title, models.JobMetrics{}, nil
	}

	var prompt string
	if generator.promptManager != nil {
		latexInstructions, _ := generator.promptManager.GetPrompt(prompts.PromptLatexInstructions, nil)
		prompt, _ = generator.promptManager.GetPrompt(prompts.PromptGenerateFlashcards, map[string]string{
			"language_requirement": fmt.Sprintf("Generate in code %s", languageCode),
			"transcript":           transcript, "reference_materials": referenceFilesContent, "latex_instructions": latexInstructions,
		})
	}

	model := options.ModelGeneration
	if model == "" {
		model = generator.configuration.LLM.GetModelForTask("content_generation")
	}

	response, metrics, err := generator.callLLMWithModel(jobContext, prompt, model)
	if err != nil {
		return "", "", metrics, err
	}
	return response, lecture.Title, metrics, nil
}

func (generator *ToolGenerator) GenerateQuiz(jobContext context.Context, lecture models.Lecture, transcript string, referenceFilesContent string, languageCode string, options models.GenerationOptions, updateProgress func(int, string, any, models.JobMetrics)) (string, string, models.JobMetrics, error) {
	if generator.llmProvider == nil {
		return "", lecture.Title, models.JobMetrics{}, nil
	}

	var prompt string
	if generator.promptManager != nil {
		latexInstructions, _ := generator.promptManager.GetPrompt(prompts.PromptLatexInstructions, nil)
		prompt, _ = generator.promptManager.GetPrompt(prompts.PromptGenerateQuiz, map[string]string{
			"language_requirement": fmt.Sprintf("Generate in code %s", languageCode),
			"transcript":           transcript, "reference_materials": referenceFilesContent, "latex_instructions": latexInstructions,
		})
	}

	model := options.ModelGeneration
	if model == "" {
		model = generator.configuration.LLM.GetModelForTask("content_generation")
	}

	response, metrics, err := generator.callLLMWithModel(jobContext, prompt, model)
	if err != nil {
		return "", "", metrics, err
	}
	return response, lecture.Title, metrics, nil
}

func (generator *ToolGenerator) unionAndMergeRanges(allRuns [][]struct {
	Start int `json:"start"`
	End   int `json:"end"`
}) []struct {
	Start int `json:"start"`
	End   int `json:"end"`
} {
	var flattenedRanges []struct {
		Start int `json:"start"`
		End   int `json:"end"`
	}
	for _, run := range allRuns {
		flattenedRanges = append(flattenedRanges, run...)
	}
	if len(flattenedRanges) == 0 {
		return nil
	}

	sort.Slice(flattenedRanges, func(firstIndex, secondIndex int) bool {
		return flattenedRanges[firstIndex].Start < flattenedRanges[secondIndex].Start
	})

	var mergedRanges []struct {
		Start int `json:"start"`
		End   int `json:"end"`
	}
	currentRange := flattenedRanges[0]
	for rangeIndex := 1; rangeIndex < len(flattenedRanges); rangeIndex++ {
		// Gap filling (<= 5 pages)
		if flattenedRanges[rangeIndex].Start-currentRange.End <= 6 {
			if flattenedRanges[rangeIndex].End > currentRange.End {
				currentRange.End = flattenedRanges[rangeIndex].End
			}
		} else {
			mergedRanges = append(mergedRanges, currentRange)
			currentRange = flattenedRanges[rangeIndex]
		}
	}
	mergedRanges = append(mergedRanges, currentRange)
	return mergedRanges
}
