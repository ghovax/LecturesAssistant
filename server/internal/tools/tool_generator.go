package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

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

// GenerateStudyGuide creates a comprehensive study guide section by section using sequential chat
func (generator *ToolGenerator) GenerateStudyGuide(jobContext context.Context, lecture models.Lecture, transcript string, referenceFilesContent string, length string, languageCode string, updateProgress func(int, string, any, models.JobMetrics)) (string, string, error) {
	var metrics models.JobMetrics

	// 1. Identify Relevant Pages
	updateProgress(5, "Identifying relevant reference materials...", nil, metrics)
	relevantMaterials, relevantMetrics, err := generator.getRelevantMaterials(jobContext, transcript, referenceFilesContent)
	if err == nil {
		metrics.InputTokens += relevantMetrics.InputTokens
		metrics.OutputTokens += relevantMetrics.OutputTokens
		metrics.EstimatedCost += relevantMetrics.EstimatedCost
	} else {
		slog.Warn("Failed to narrow down relevant materials, using full content", "error", err)
		relevantMaterials = referenceFilesContent
	}

	// 2. Analyze Lecture Structure
	updateProgress(10, "Analyzing lecture structure...", nil, metrics)

	type sectionRange struct {
		minimum   int
		maximum   int
		preferred string
	}

	var sectionCounts sectionRange
	switch length {
	case "short":
		sectionCounts = sectionRange{minimum: 1, maximum: 4, preferred: "2-3"}
	case "long":
		sectionCounts = sectionRange{minimum: 4, maximum: 7, preferred: "5-6"}
	default: // medium
		sectionCounts = sectionRange{minimum: 2, maximum: 5, preferred: "3-4"}
	}

	latexInstructions, _ := generator.promptManager.GetPrompt(prompts.PromptLatexInstructions, nil)

	// Load full document example for structure
	examplePrompt := prompts.PromptStudyGuideWithoutCitationsExample
	if relevantMaterials != "" {
		examplePrompt = prompts.PromptStudyGuideWithCitationsExample
	}
	exampleTemplateForStructure, _ := generator.promptManager.GetPrompt(examplePrompt, nil)

	structurePrompt, err := generator.promptManager.GetPrompt(prompts.PromptAnalyzeLectureStructure, map[string]string{
		"language_requirement":    fmt.Sprintf("Use language code %s", languageCode),
		"minimum_section_count":   strconv.Itoa(sectionCounts.minimum),
		"max_section_count":       strconv.Itoa(sectionCounts.maximum),
		"preferred_section_range": sectionCounts.preferred,
		"latex_instructions":      latexInstructions,
		"example_template":        exampleTemplateForStructure,
		"transcript":              transcript,
		"reference_materials":     relevantMaterials,
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to prepare structure prompt: %w", err)
	}

	structureResponse, structureMetrics, err := generator.callLLM(jobContext, structurePrompt)
	if err != nil {
		return "", "", fmt.Errorf("failed to analyze structure: %w", err)
	}
	metrics.InputTokens += structureMetrics.InputTokens
	metrics.OutputTokens += structureMetrics.OutputTokens
	metrics.EstimatedCost += structureMetrics.EstimatedCost

	title := generator.parseTitle(structureResponse)
	if title == "" {
		title = lecture.Title
	}

	sections := generator.parseStructure(structureResponse)
	if len(sections) == 0 {
		return "", "", fmt.Errorf("failed to parse sections from LLM response")
	}

	// 3. Sequential Generation Setup
	updateProgress(15, "Initializing sequential generation...", nil, metrics)

	initialContextTemplate, _ := generator.promptManager.GetPrompt(prompts.PromptStudyGuideInitialContext, nil)
	languageRequirement, _ := generator.promptManager.GetPrompt(prompts.PromptLanguageRequirement, map[string]string{
		"language":         languageCode,
		"bcp_47_lang_code": languageCode,
	})
	citationInstructions := ""
	exampleTemplatePrompt := prompts.PromptSectionWithoutCitationsExample
	if relevantMaterials != "" {
		citationInstructions, _ = generator.promptManager.GetPrompt(prompts.PromptCitationInstructions, nil)
		exampleTemplatePrompt = prompts.PromptSectionWithCitationsExample
	}
	exampleTemplate, _ := generator.promptManager.GetPrompt(exampleTemplatePrompt, nil)

	initialContext := generator.replacePromptVariables(initialContextTemplate, map[string]string{
		"language_requirement": languageRequirement,
		"transcript":           transcript,
		"reference_materials":  relevantMaterials,
		"structure_outline":    structureResponse,
	})

	history := []llm.Message{
		{Role: "user", Content: []llm.ContentPart{{Type: "text", Text: initialContext}}},
		{Role: "assistant", Content: []llm.ContentPart{{Type: "text", Text: "I understand the lecture content, reference materials, and structural outline. I'm ready to generate each section sequentially with complete coverage and smooth transitions."}}},
	}

	// 4. Generate Sections
	reconstructor := markdown.NewReconstructor()
	rootNode := &markdown.Node{Type: markdown.NodeDocument}
	rootNode.Children = append(rootNode.Children, &markdown.Node{
		Type:    markdown.NodeHeading,
		Level:   1,
		Content: title,
	})

	totalSections := len(sections)
	for sectionIndex, section := range sections {
		sectionTitle := section.Title
		sectionCoverage := section.Coverage

		sectionMetadata := map[string]any{
			"section_index":  sectionIndex + 1,
			"total_sections": totalSections,
			"section_title":  sectionTitle,
		}

		progress := 20 + int(float64(sectionIndex)/float64(totalSections)*75)
		updateProgress(progress, "Generating study guide section...", sectionMetadata, metrics)

		sectionPromptTemplate, _ := generator.promptManager.GetPrompt(prompts.PromptStudyGuideSectionGeneration, nil)
		sectionPrompt := generator.replacePromptVariables(sectionPromptTemplate, map[string]string{
			"language_requirement":  languageRequirement,
			"section_title":         sectionTitle,
			"section_coverage":      sectionCoverage,
			"structure_outline":     structureResponse,
			"citation_instructions": citationInstructions,
			"latex_instructions":    latexInstructions,
			"example_template":      exampleTemplate,
		})

		for attempt := 1; attempt <= 2; attempt++ {
			response, genMetrics, err := generator.callLLMWithHistory(jobContext, sectionPrompt, history)
			if err != nil {
				return "", "", err
			}
			metrics.InputTokens += genMetrics.InputTokens
			metrics.OutputTokens += genMetrics.OutputTokens
			metrics.EstimatedCost += genMetrics.EstimatedCost

			updateProgress(progress, "Verifying section quality...", sectionMetadata, metrics)
			verifyPromptTemplate, _ := generator.promptManager.GetPrompt(prompts.PromptVerifySectionAdherence, nil)
			verifyPrompt := generator.replacePromptVariables(verifyPromptTemplate, map[string]string{
				"section_title":     sectionTitle,
				"expected_coverage": sectionCoverage,
				"generated_section": response,
			})

			verifyResponse, verifyMetrics, _ := generator.callLLM(jobContext, verifyPrompt)
			metrics.InputTokens += verifyMetrics.InputTokens
			metrics.OutputTokens += verifyMetrics.OutputTokens
			metrics.EstimatedCost += verifyMetrics.EstimatedCost

			score := generator.parseScore(verifyResponse)
			if score >= 70 || attempt == 2 {
				sectionParser := markdown.NewParser()
				sectionAST := sectionParser.Parse(response)
				rootNode.Children = append(rootNode.Children, sectionAST.Children...)

				history = append(history, llm.Message{Role: "user", Content: []llm.ContentPart{{Type: "text", Text: sectionPrompt}}})
				history = append(history, llm.Message{Role: "assistant", Content: []llm.ContentPart{{Type: "text", Text: response}}})
				break
			}

			updateProgress(progress, "Regenerating section due to low quality...", map[string]any{
				"section_index": sectionIndex + 1,
				"section_title": sectionTitle,
				"score":         score,
				"attempt":       attempt,
			}, metrics)
		}
	}

	finalMarkdown := reconstructor.Reconstruct(rootNode)
	updateProgress(100, "Study guide generation completed", nil, metrics)
	return finalMarkdown, title, nil
}

// GenerateFlashcards creates a set of flashcards from the lecture content
func (generator *ToolGenerator) GenerateFlashcards(jobContext context.Context, lecture models.Lecture, transcript string, referenceFilesContent string, languageCode string, updateProgress func(int, string, any, models.JobMetrics)) (string, string, error) {
	var metrics models.JobMetrics
	updateProgress(10, "Preparing flashcard generation...", nil, metrics)

	latexInstructions, _ := generator.promptManager.GetPrompt(prompts.PromptLatexInstructions, nil)

	flashcardPrompt, err := generator.promptManager.GetPrompt(prompts.PromptGenerateFlashcards, map[string]string{
		"language_requirement": fmt.Sprintf("Generate the flashcards in language code %s", languageCode),
		"transcript":           transcript,
		"reference_materials":  referenceFilesContent,
		"latex_instructions":   latexInstructions,
	})
	if err != nil {
		return "", "", err
	}

	updateProgress(30, "Generating flashcards via LLM...", nil, metrics)
	flashcardResponse, llmMetrics, err := generator.callLLM(jobContext, flashcardPrompt)
	if err != nil {
		return "", "", err
	}
	metrics.InputTokens += llmMetrics.InputTokens
	metrics.OutputTokens += llmMetrics.OutputTokens
	metrics.EstimatedCost += llmMetrics.EstimatedCost

	updateProgress(100, "Flashcard generation completed", nil, metrics)
	return flashcardResponse, "Flashcards: " + lecture.Title, nil
}

// GenerateQuiz creates a multiple-choice quiz from the lecture content
func (generator *ToolGenerator) GenerateQuiz(jobContext context.Context, lecture models.Lecture, transcript string, referenceFilesContent string, languageCode string, updateProgress func(int, string, any, models.JobMetrics)) (string, string, error) {
	var metrics models.JobMetrics
	updateProgress(10, "Preparing quiz generation...", nil, metrics)

	latexInstructions, _ := generator.promptManager.GetPrompt(prompts.PromptLatexInstructions, nil)

	quizPrompt, err := generator.promptManager.GetPrompt(prompts.PromptGenerateQuiz, map[string]string{
		"language_requirement": fmt.Sprintf("Generate the quiz in language code %s", languageCode),
		"transcript":           transcript,
		"reference_materials":  referenceFilesContent,
		"latex_instructions":   latexInstructions,
	})
	if err != nil {
		return "", "", err
	}

	updateProgress(30, "Generating quiz via LLM...", nil, metrics)
	quizResponse, llmMetrics, err := generator.callLLM(jobContext, quizPrompt)
	if err != nil {
		return "", "", err
	}
	metrics.InputTokens += llmMetrics.InputTokens
	metrics.OutputTokens += llmMetrics.OutputTokens
	metrics.EstimatedCost += llmMetrics.EstimatedCost

	updateProgress(100, "Quiz generation completed", nil, metrics)
	return quizResponse, "Quiz: " + lecture.Title, nil
}

// ProcessFootnotesAI improves the quality of footnotes using AI
func (generator *ToolGenerator) ProcessFootnotesAI(jobContext context.Context, citations []markdown.ParsedCitation) ([]markdown.ParsedCitation, models.JobMetrics, error) {
	if len(citations) == 0 {
		return nil, models.JobMetrics{}, nil
	}

	var totalMetrics models.JobMetrics
	latexInstructions, _ := generator.promptManager.GetPrompt(prompts.PromptLatexInstructions, nil)

	// Step 1: Format current citations as markdown for parsing
	var footnotesMarkdown strings.Builder
	for _, citation := range citations {
		footnotesMarkdown.WriteString(fmt.Sprintf("[^%d]: %s\n\n", citation.Number, citation.Description))
	}

	// Step 2: Parse metadata using AI to verify and enrich
	parsePrompt, err := generator.promptManager.GetPrompt(prompts.PromptParseFootnotes, map[string]string{
		"footnotes":          footnotesMarkdown.String(),
		"latex_instructions": latexInstructions,
	})
	if err != nil {
		return citations, totalMetrics, err
	}

	parseResponse, parseMetrics, err := generator.callLLM(jobContext, parsePrompt)
	if err != nil {
		slog.Warn("Failed to parse footnotes via AI, using original", "error", err)
		return citations, totalMetrics, nil
	}
	totalMetrics.InputTokens += parseMetrics.InputTokens
	totalMetrics.OutputTokens += parseMetrics.OutputTokens
	totalMetrics.EstimatedCost += parseMetrics.EstimatedCost

	// Extract JSON from response
	jsonStr := parseResponse
	if idx := strings.Index(jsonStr, "{"); idx != -1 {
		if lastIdx := strings.LastIndex(jsonStr, "}"); lastIdx != -1 {
			jsonStr = jsonStr[idx : lastIdx+1]
		}
	}

	type aiFootnote struct {
		Number      int    `json:"number"`
		TextContent string `json:"text_content"`
		File        string `json:"file"`
		Pages       []int  `json:"pages"`
	}
	type aiResponse struct {
		Footnotes []aiFootnote `json:"footnotes"`
	}

	var result aiResponse
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		slog.Warn("Failed to unmarshal AI footnote response", "error", err)
		return citations, totalMetrics, nil
	}

	// Step 3: Fix content using format-footnotes prompt
	formatPrompt, err := generator.promptManager.GetPrompt(prompts.PromptFormatFootnotes, map[string]string{
		"footnotes":          footnotesMarkdown.String(),
		"latex_instructions": latexInstructions,
	})
	if err != nil {
		return citations, totalMetrics, err
	}

	formatResponse, formatMetrics, err := generator.callLLM(jobContext, formatPrompt)
	if err != nil {
		slog.Warn("Failed to format footnotes via AI, using original", "error", err)
		return citations, totalMetrics, nil
	}
	totalMetrics.InputTokens += formatMetrics.InputTokens
	totalMetrics.OutputTokens += formatMetrics.OutputTokens
	totalMetrics.EstimatedCost += formatMetrics.EstimatedCost

	// Step 4: Combine AI results back into citations
	// We parse the formatted response which should be markdown definitions
	parser := markdown.NewParser()
	ast := parser.Parse(formatResponse)

	// Create map of formatted content by footnote number
	formattedContent := make(map[int]string)
	for _, node := range ast.Children {
		if node.Type == markdown.NodeFootnote {
			formattedContent[node.FootnoteNumber] = node.Content
		}
	}

	updatedCitations := make([]markdown.ParsedCitation, len(citations))
	for i, original := range citations {
		updated := original

		// Find matching metadata from AI parse
		for _, aiFn := range result.Footnotes {
			if aiFn.Number == original.Number {
				if aiFn.File != "" {
					updated.File = aiFn.File
				}
				if len(aiFn.Pages) > 0 {
					updated.Pages = aiFn.Pages
				}
				break
			}
		}

		// Update description with formatted content if available
		if content, ok := formattedContent[original.Number]; ok {
			updated.Description = content
		}

		updatedCitations[i] = updated
	}

	return updatedCitations, totalMetrics, nil
}

func (generator *ToolGenerator) callLLM(jobContext context.Context, prompt string) (string, models.JobMetrics, error) {
	return generator.callLLMWithHistory(jobContext, prompt, nil)
}

func (generator *ToolGenerator) callLLMWithHistory(jobContext context.Context, prompt string, history []llm.Message) (string, models.JobMetrics, error) {
	model := generator.configuration.LLM.OpenRouter.DefaultModel
	if generator.configuration.LLM.Provider == "ollama" {
		model = generator.configuration.LLM.Ollama.DefaultModel
	}

	messages := append(history, llm.Message{
		Role:    "user",
		Content: []llm.ContentPart{{Type: "text", Text: prompt}},
	})

	responseChannel, err := generator.llmProvider.Chat(jobContext, llm.ChatRequest{
		Model:    model,
		Messages: messages,
		Stream:   false,
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

	// Safety check: Cost Threshold
	// Since callLLMWithHistory might be called in a loop, the caller should ideally track the aggregate.
	// But as a catch-all safety measure, we'll check individual calls too.
	if generator.configuration.Safety.MaxCostPerJob > 0 && metrics.EstimatedCost > generator.configuration.Safety.MaxCostPerJob {
		return "", metrics, fmt.Errorf("safety threshold exceeded: call cost $%.4f > limit $%.4f", metrics.EstimatedCost, generator.configuration.Safety.MaxCostPerJob)
	}

	return resultBuilder.String(), metrics, nil
}

func (generator *ToolGenerator) getRelevantMaterials(jobContext context.Context, transcript string, fullMaterials string) (string, models.JobMetrics, error) {
	if fullMaterials == "" {
		return "", models.JobMetrics{}, nil
	}

	prompt, err := generator.promptManager.GetPrompt(prompts.PromptGetRelevantPages, map[string]string{
		"transcript":      transcript,
		"reference_files": fullMaterials,
	})
	if err != nil {
		return fullMaterials, models.JobMetrics{}, err
	}

	response, metrics, err := generator.callLLM(jobContext, prompt)
	if err != nil {
		return fullMaterials, metrics, err
	}

	type pageRange struct {
		Start int `json:"start"`
		End   int `json:"end"`
	}
	type resultType struct {
		PageRanges []pageRange `json:"page_ranges"`
	}

	var result resultType
	jsonStr := response
	if idx := strings.Index(jsonStr, "{"); idx != -1 {
		if lastIdx := strings.LastIndex(jsonStr, "}"); lastIdx != -1 {
			jsonStr = jsonStr[idx : lastIdx+1]
		}
	}

	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return fullMaterials, metrics, fmt.Errorf("failed to parse relevant pages JSON: %w", err)
	}

	if len(result.PageRanges) == 0 {
		return "", metrics, nil
	}

	// Refined Filtering using AST
	parser := markdown.NewParser()
	ast := parser.Parse(fullMaterials)
	reconstructor := markdown.NewReconstructor()

	filteredRoot := &markdown.Node{Type: markdown.NodeDocument}

	// We iterate through children. Headings level 1 indicate files, level 2 indicate pages
	var currentFileNode *markdown.Node
	includeFile := false

	for _, node := range ast.Children {
		if node.Type == markdown.NodeHeading && node.Level == 1 {
			// New file
			currentFileNode = node
			includeFile = false
		} else if node.Type == markdown.NodeHeading && node.Level == 2 {
			// Check if this page is in any range
			pageTitle := strings.ToLower(node.Content)
			if strings.Contains(pageTitle, "page") {
				numStr := strings.TrimSpace(strings.ReplaceAll(pageTitle, "page", ""))
				pageNum, _ := strconv.Atoi(numStr)

				isRelevant := false
				for _, r := range result.PageRanges {
					if pageNum >= r.Start && pageNum <= r.End {
						isRelevant = true
						break
					}
				}

				if isRelevant {
					if !includeFile && currentFileNode != nil {
						filteredRoot.Children = append(filteredRoot.Children, currentFileNode)
						includeFile = true
					}
					filteredRoot.Children = append(filteredRoot.Children, node)
				} else {
					includeFile = false
				}
			}
		} else if includeFile {
			// Append content of relevant page
			filteredRoot.Children = append(filteredRoot.Children, node)
		}
	}

	return reconstructor.Reconstruct(filteredRoot), metrics, nil
}

func (generator *ToolGenerator) replacePromptVariables(prompt string, variables map[string]string) string {
	result := prompt
	for key, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

type sectionInfo struct {
	Title    string
	Coverage string
}

func (generator *ToolGenerator) parseTitle(structure string) string {
	markdownParser := markdown.NewParser()
	documentAST := markdownParser.Parse(structure)

	for _, child := range documentAST.Children {
		if child.Type == markdown.NodeSection && child.Level == 1 {
			return child.Title
		}
		if child.Type == markdown.NodeHeading && child.Level == 1 {
			return child.Content
		}
	}
	return ""
}

func (generator *ToolGenerator) parseStructure(structure string) []sectionInfo {
	parser := markdown.NewParser()
	documentAST := parser.Parse(structure)
	reconstructor := markdown.NewReconstructor()

	var sections []sectionInfo

	var findSections func(*markdown.Node)
	findSections = func(node *markdown.Node) {
		if node.Type == markdown.NodeSection && node.Level == 2 {
			coverage := reconstructor.Reconstruct(&markdown.Node{
				Type:     markdown.NodeDocument,
				Children: node.Children,
			})
			sections = append(sections, sectionInfo{
				Title:    node.Title,
				Coverage: coverage,
			})
		} else {
			for _, child := range node.Children {
				findSections(child)
			}
		}
	}

	findSections(documentAST)
	return sections
}

func (generator *ToolGenerator) parseScore(response string) int {
	type scoreResponse struct {
		CoverageScore int `json:"coverage_score"`
	}

	// Try to find JSON block if fenced
	jsonStr := response
	if idx := strings.Index(jsonStr, "{"); idx != -1 {
		if lastIdx := strings.LastIndex(jsonStr, "}"); lastIdx != -1 {
			jsonStr = jsonStr[idx : lastIdx+1]
		}
	}

	var result scoreResponse
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		slog.Error("Failed to parse coverage score JSON", "error", err, "response", response)
		return 0
	}

	return result.CoverageScore
}
