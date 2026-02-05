package tools

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	config "lectures/internal/configuration"
	"lectures/internal/jobs"
	"lectures/internal/llm"
	"lectures/internal/models"
	"lectures/internal/prompts"
)

type GuideGenerator struct {
	configuration *config.Configuration
	llmProvider   llm.Provider
	promptManager *prompts.Manager
}

func NewGuideGenerator(configuration *config.Configuration, llmProvider llm.Provider, promptManager *prompts.Manager) *GuideGenerator {
	return &GuideGenerator{
		configuration: configuration,
		llmProvider:   llmProvider,
		promptManager: promptManager,
	}
}

// GenerateStudyGuide creates a comprehensive study guide section by section
func (generator *GuideGenerator) GenerateStudyGuide(jobContext context.Context, lecture models.Lecture, transcript string, referenceFilesContent string, length string, languageCode string, updateProgress func(int, string, any, jobs.JobMetrics)) (string, string, error) {
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

	// 2. Parse Structure
	title := generator.parseTitle(structureResponse)
	if title == "" {
		title = lecture.Title // Fallback
	}

	sections := generator.parseStructure(structureResponse)
	if len(sections) == 0 {
		return "", "", fmt.Errorf("failed to parse sections from LLM response")
	}

	var fullGuideBuilder strings.Builder
	fullGuideBuilder.WriteString("# " + title + "\n\n")

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
				fullGuideBuilder.WriteString(sectionContent + "\n\n")
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

	updateProgress(100, "Study guide generation completed", nil, metrics)
	return fullGuideBuilder.String(), title, nil
}

func (generator *GuideGenerator) callLLM(context context.Context, prompt string) (string, jobs.JobMetrics, error) {
	responseChannel, err := generator.llmProvider.Chat(context, llm.ChatRequest{
		Model:    generator.configuration.LLM.OpenRouter.DefaultModel,
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

func (generator *GuideGenerator) parseTitle(structure string) string {
	lines := strings.Split(structure, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return ""
}

func (generator *GuideGenerator) parseStructure(structure string) []sectionInfo {
	var sections []sectionInfo
	lines := strings.Split(structure, "\n")
	var currentSection *sectionInfo

	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			if currentSection != nil {
				sections = append(sections, *currentSection)
			}
			currentSection = &sectionInfo{
				Title: strings.TrimSpace(strings.TrimPrefix(line, "## ")),
			}
		} else if currentSection != nil {
			currentSection.Coverage += line + "\n"
		}
	}
	if currentSection != nil {
		sections = append(sections, *currentSection)
	}
	return sections
}

func (generator *GuideGenerator) parseScore(response string) int {
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
