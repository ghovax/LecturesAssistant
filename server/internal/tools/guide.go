package tools

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"lectures/internal/configuration"
	"lectures/internal/jobs"
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

// GenerateStudyGuide creates a comprehensive study guide section by section
func (generator *ToolGenerator) GenerateStudyGuide(jobContext context.Context, lecture models.Lecture, transcript string, referenceFilesContent string, length string, languageCode string, updateProgress func(int, string, any, jobs.JobMetrics)) (string, string, error) {
	var metrics jobs.JobMetrics

	// 1. Analyze Lecture Structure
	updateProgress(5, "Analyzing lecture structure...", nil, metrics)

	structurePrompt, err := generator.promptManager.GetPrompt(prompts.PromptAnalyzeLectureStructure, map[string]string{
		"language_requirement":    "Generate the outline in the requested language.",
		"min_section_count":       "5",
		"max_section_count":       "15",
		"preferred_section_range": "8-12",
		"latex_instructions":      "",
		"transcript":              transcript,
		"reference_materials":     referenceFilesContent,
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

	reconstructor := markdown.NewReconstructor()

	var sections []sectionInfo

	// 2. Parse Structure
	title := generator.parseTitle(structureResponse)
	if title == "" {
		title = lecture.Title // Fallback
	}

	sections = generator.parseStructure(structureResponse)
	if len(sections) == 0 {
		return "", "", fmt.Errorf("failed to parse sections from LLM response")
	}

	// Create root document node
	rootNode := &markdown.Node{
		Type: markdown.NodeDocument,
	}

	// Add title
	rootNode.Children = append(rootNode.Children, &markdown.Node{
		Type:    markdown.NodeHeading,
		Level:   1,
		Content: title,
	})

	// 3. Generate Section by Section
	totalSections := len(sections)
	for sectionIndex, section := range sections {
		sectionTitle := section.Title
		sectionCoverage := section.Coverage

		sectionMetadata := map[string]any{
			"section_index":  sectionIndex + 1,
			"total_sections": totalSections,
			"section_title":  sectionTitle,
		}

		progress := 10 + int(float64(sectionIndex)/float64(totalSections)*80)
		updateProgress(progress, "Generating study guide section...", sectionMetadata, metrics)

		for attempt := 1; attempt <= 2; attempt++ {
			// Prepare generation prompt
			genPrompt, err := generator.promptManager.GetPrompt(prompts.PromptSequentialDocumentSectionGeneration, map[string]string{
				"language_requirement":  fmt.Sprintf("Use language code %s", languageCode),
				"section_title":         sectionTitle,
				"section_coverage":      sectionCoverage,
				"transcript":            transcript,
				"reference_materials":   referenceFilesContent,
				"structure_outline":     structureResponse,
				"citation_instructions": "",
				"latex_instructions":    "",
				"example_template":      "",
			})
			if err != nil {
				return "", "", err
			}

			sectionContent, genMetrics, err := generator.callLLM(jobContext, genPrompt)
			if err != nil {
				return "", "", err
			}
			metrics.InputTokens += genMetrics.InputTokens
			metrics.OutputTokens += genMetrics.OutputTokens
			metrics.EstimatedCost += genMetrics.EstimatedCost

			// 4. Refine/Verify Section
			updateProgress(progress, "Verifying section quality...", sectionMetadata, metrics)

			verifyPrompt, err := generator.promptManager.GetPrompt(prompts.PromptVerifySectionAdherence, map[string]string{
				"section_title":     sectionTitle,
				"expected_coverage": sectionCoverage,
				"generated_section": sectionContent,
			})
			if err != nil {
				return "", "", err
			}

			verifyResponse, verifyMetrics, err := generator.callLLM(jobContext, verifyPrompt)
			if err != nil {
				return "", "", err
			}
			metrics.InputTokens += verifyMetrics.InputTokens
			metrics.OutputTokens += verifyMetrics.OutputTokens
			metrics.EstimatedCost += verifyMetrics.EstimatedCost

			score := generator.parseScore(verifyResponse)
			if score >= 70 || attempt == 2 {
				// Parse the generated section and append its children to the document
				sectionParser := markdown.NewParser()
				sectionAST := sectionParser.Parse(sectionContent)
				rootNode.Children = append(rootNode.Children, sectionAST.Children...)
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
func (generator *ToolGenerator) GenerateFlashcards(jobContext context.Context, lecture models.Lecture, transcript string, referenceFilesContent string, languageCode string, updateProgress func(int, string, any, jobs.JobMetrics)) (string, string, error) {
	var metrics jobs.JobMetrics
	updateProgress(10, "Preparing flashcard generation...", nil, metrics)

	flashcardPrompt, err := generator.promptManager.GetPrompt(prompts.PromptGenerateFlashcards, map[string]string{
		"language_requirement": fmt.Sprintf("Generate the flashcards in language code %s", languageCode),
		"transcript":           transcript,
		"reference_materials":  referenceFilesContent,
		"latex_instructions":   "",
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
func (generator *ToolGenerator) GenerateQuiz(jobContext context.Context, lecture models.Lecture, transcript string, referenceFilesContent string, languageCode string, updateProgress func(int, string, any, jobs.JobMetrics)) (string, string, error) {
	var metrics jobs.JobMetrics
	updateProgress(10, "Preparing quiz generation...", nil, metrics)

	quizPrompt, err := generator.promptManager.GetPrompt(prompts.PromptGenerateQuiz, map[string]string{
		"language_requirement": fmt.Sprintf("Generate the quiz in language code %s", languageCode),
		"transcript":           transcript,
		"reference_materials":  referenceFilesContent,
		"latex_instructions":   "",
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

func (generator *ToolGenerator) callLLM(context context.Context, prompt string) (string, jobs.JobMetrics, error) {
	model := generator.configuration.LLM.OpenRouter.DefaultModel
	if generator.configuration.LLM.Provider == "ollama" {
		model = generator.configuration.LLM.Ollama.DefaultModel
	}

	responseChannel, err := generator.llmProvider.Chat(context, llm.ChatRequest{
		Model:    model,
		Messages: []llm.Message{{Role: "user", Content: []llm.ContentPart{{Type: "text", Text: prompt}}}},
		Stream:   false,
	})
	if err != nil {
		return "", jobs.JobMetrics{}, err
	}

	var resultBuilder strings.Builder
	var metrics jobs.JobMetrics
	for chunk := range responseChannel {
		if chunk.Error != nil {
			return "", jobs.JobMetrics{}, chunk.Error
		}
		resultBuilder.WriteString(chunk.Text)
		metrics.InputTokens += chunk.InputTokens
		metrics.OutputTokens += chunk.OutputTokens
		metrics.EstimatedCost += chunk.Cost
	}

	return resultBuilder.String(), metrics, nil
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

	// Helper function to recursively find sections at level 2
	var findSections func(*markdown.Node)
	findSections = func(node *markdown.Node) {
		if node.Type == markdown.NodeSection && node.Level == 2 {
			// Reconstruct coverage from children
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
	startTag := "<coverage_score>"
	endTag := "</coverage_score>"
	start := strings.Index(response, startTag)
	end := strings.Index(response, endTag)
	if start == -1 || end == -1 {
		return 0
	}
	scoreStr := response[start+len(startTag) : end]
	score, _ := strconv.Atoi(strings.TrimSpace(scoreStr))
	return score
}
