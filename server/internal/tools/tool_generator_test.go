package tools

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"lectures/internal/configuration"
	"lectures/internal/llm"
	"lectures/internal/markdown"
	"lectures/internal/models"
	"lectures/internal/prompts"
)

// UnbreakableSequentialMock returns responses in a fixed order
type UnbreakableSequentialMock struct {
	Responses []string
	CallIndex int
	Histories [][]llm.Message
	mutex     sync.Mutex
}

func (mock *UnbreakableSequentialMock) Chat(ctx context.Context, req llm.ChatRequest) (<-chan llm.ChatResponseChunk, error) {
	mock.mutex.Lock()
	defer mock.mutex.Unlock()

	channel := make(chan llm.ChatResponseChunk, 1)
	defer close(channel)

	mock.Histories = append(mock.Histories, req.Messages)

	if mock.CallIndex < len(mock.Responses) {
		channel <- llm.ChatResponseChunk{Text: mock.Responses[mock.CallIndex]}
		mock.CallIndex++
	} else {
		channel <- llm.ChatResponseChunk{Text: "Out of responses"}
	}

	return channel, nil
}

func (mock *UnbreakableSequentialMock) Name() string { return "unbreakable-sequential-mock" }

func TestToolGenerator_Triangulation(tester *testing.T) {
	config := &configuration.Configuration{}
	promptManager := prompts.NewManager("../../prompts")

	mockLLM := &UnbreakableSequentialMock{
		Responses: []string{
			`{"page_ranges": [{"start": 1, "end": 2}]}`,
			`{"page_ranges": [{"start": 4, "end": 6}]}`,
			`{"page_ranges": [{"start": 12, "end": 15}]}`,
		},
	}

	generator := NewToolGenerator(config, mockLLM, promptManager)

	fullMaterials := `# File A

## Page 1
Content 1

## Page 4
Content 4

## Page 12
Content 12`
	materials, _, err := generator.triangulateRelevantMaterials(context.Background(), "Transcript", fullMaterials, models.GenerationOptions{EnableTriangulation: true})
	if err != nil {
		tester.Fatalf("Triangulation failed: %v", err)
	}

	if !strings.Contains(materials, "Page 1") || !strings.Contains(materials, "Page 4") || !strings.Contains(materials, "Page 12") {
		tester.Errorf("Triangulation missed pages. Result:\n%s", materials)
	}
}

func TestToolGenerator_SequentialBuildingWithCleanHistory(tester *testing.T) {
	config := &configuration.Configuration{}
	promptManager := prompts.NewManager("../../prompts")

	mockLLM := &UnbreakableSequentialMock{
		Responses: []string{
			// Triangulation (3 calls)
			`{"page_ranges": []}`,
			`{"page_ranges": []}`,
			`{"page_ranges": []}`,
			// Structure Analysis
			`# Outline
## Intro
Coverage: Part 1
## Deep Dive
Coverage: Part 2`,
			// CleanDocumentTitle call (added in recent changes)
			`{"title": "Outline"}`,
			// Section 1: Intro
			`## Intro
Success 1`,
			`{"coverage_score": 95}`,
			// Section 2: Deep Dive (Attempt 1: Wrong Title)
			`## Mistake
Hallucination`,
			// Section 2: Deep Dive (Attempt 2: Success)
			// (Note: Attempt 1 had low similarity, so verification was skipped)
			`## Deep Dive
Success 2`,
			`{"coverage_score": 90}`,
		},
	}

	generator := NewToolGenerator(config, mockLLM, promptManager)
	lecture := models.Lecture{Title: "Lecture Title"}

	options := models.GenerationOptions{
		EnableTriangulation: true,
		AdherenceThreshold:  70,
		MaximumAttempts:     3,
	}

	result, _, err := generator.GenerateStudyGuide(context.Background(), lecture, "Transcript", "References", "medium", "en", options, func(p int, m string, meta any, met models.JobMetrics) {})
	if err != nil {
		tester.Fatalf("Generation failed: %v", err)
	}

	if strings.Count(result, "## ") != 2 {
		tester.Errorf("Expected 2 sections, found %d. Result:\n%s", strings.Count(result, "## "), result)
	}

	// Verify "Clean History" - Call index 8 should be the retry for Deep Dive
	if len(mockLLM.Histories) > 8 {
		historyStr := strings.ToLower(fmt.Sprintf("%v", mockLLM.Histories[8]))
		if strings.Contains(historyStr, "hallucination") {
			tester.Errorf("Clean History FAILED: history contains failed attempt data.")
		}
	}
}

func TestToolGenerator_FootnoteHealing(tester *testing.T) {
	config := &configuration.Configuration{}
	promptManager := prompts.NewManager("../../prompts")

	mockLLM := &UnbreakableSequentialMock{
		Responses: []string{
			`{"footnotes": [{"number": 1, "file": "f1.pdf", "pages": [1]}, {"number": 99, "file": "f2.pdf", "pages": [5]}]}`,
			`Body text.[^1] [^99]

[^1]: Improved 1

[^99]: Improved 2
`,
		},
	}

	generator := NewToolGenerator(config, mockLLM, promptManager)

	citations := []markdown.ParsedCitation{
		{Number: 1, Description: "Raw 1"},
		{Number: 2, Description: "Raw 2"},
	}

	updated, _, _ := generator.ProcessFootnotesAI(context.Background(), citations, models.GenerationOptions{})

	if updated[1].File != "f2.pdf" {
		tester.Errorf("Healing failed: Got: %s", updated[1].File)
	}
	if !strings.Contains(updated[1].Description, "Improved 2") {
		tester.Errorf("Polishing failed: Got: %s", updated[1].Description)
	}
}

func TestToolGenerator_FencedJSONExtraction(tester *testing.T) {
	generator := &ToolGenerator{}

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"Plain JSON", `{"valid": true}`, `{"valid": true}`},
		{"Markdown Fenced", "Sure!\n```json\n{\"a\": 1}\n```\nHope that helps!", `{"a": 1}`},
		{"Implicit Fence", "Here is the data: [1, 2, 3] check it out.", `[1, 2, 3]`},
	}

	for _, tc := range testCases {
		tester.Run(tc.name, func(subTester *testing.T) {
			result := generator.extractFencedJSON(tc.input)
			if result != tc.expected {
				subTester.Errorf("Extraction failed. Expected %s, got %s", tc.expected, result)
			}
		})
	}
}
