package transcription

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"lectures/internal/llm"
	"lectures/internal/models"
)

type OpenRouterTranscriptionProvider struct {
	llmProvider llm.Provider
	model       string
	prompt      string
}

func NewOpenRouterTranscriptionProvider(llmProvider llm.Provider, model string) *OpenRouterTranscriptionProvider {
	if model == "" {
		// Default to a model that supports audio input
		model = "google/gemini-2.5-flash-lite"
	}
	return &OpenRouterTranscriptionProvider{
		llmProvider: llmProvider,
		model:       model,
	}
}

func (provider *OpenRouterTranscriptionProvider) SetPrompt(prompt string) {
	provider.prompt = prompt
}

func (provider *OpenRouterTranscriptionProvider) Name() string {
	return "openrouter-transcription"
}

func (provider *OpenRouterTranscriptionProvider) CheckDependencies() error {
	if provider.llmProvider == nil {
		return fmt.Errorf("LLM provider is missing")
	}
	return nil
}

func (provider *OpenRouterTranscriptionProvider) Transcribe(jobContext context.Context, audioPath string) ([]Segment, models.JobMetrics, error) {
	var metrics models.JobMetrics

	// Read audio file
	audioData, err := os.ReadFile(audioPath)
	if err != nil {
		return nil, metrics, fmt.Errorf("failed to read audio file: %w", err)
	}

	// Encode to base64
	base64Audio := base64.StdEncoding.EncodeToString(audioData)

	// Determine audio format from file extension
	ext := strings.ToLower(filepath.Ext(audioPath))
	format := strings.TrimPrefix(ext, ".")
	if format == "" {
		format = "mp3" // default
	}

	// Build the transcription prompt
	transcriptionPrompt := "Please transcribe this audio file. Return only the transcribed text without any additional commentary."
	if provider.prompt != "" {
		transcriptionPrompt = provider.prompt
	}

	// Create chat request with audio input
	request := llm.ChatRequest{
		Model: provider.model,
		Messages: []llm.Message{
			{
				Role: "user",
				Content: []llm.ContentPart{
					{
						Type: "text",
						Text: transcriptionPrompt,
					},
					{
						Type:        "input_audio",
						AudioData:   base64Audio,
						AudioFormat: format,
					},
				},
			},
		},
		Stream: false,
	}

	// Call LLM
	responseChannel, err := provider.llmProvider.Chat(jobContext, request)
	if err != nil {
		return nil, metrics, fmt.Errorf("LLM chat failed: %w", err)
	}

	// Collect response
	var transcriptionBuilder strings.Builder
	for chunk := range responseChannel {
		if chunk.Error != nil {
			return nil, metrics, fmt.Errorf("LLM streaming error: %w", chunk.Error)
		}
		transcriptionBuilder.WriteString(chunk.Text)
		metrics.InputTokens += chunk.InputTokens
		metrics.OutputTokens += chunk.OutputTokens
		metrics.EstimatedCost += chunk.Cost
	}

	transcribedText := strings.TrimSpace(transcriptionBuilder.String())
	if transcribedText == "" {
		return nil, metrics, fmt.Errorf("no transcription received from LLM")
	}

	// Return as a single segment (OpenRouter doesn't provide timestamps via chat API)
	// The TranscribeLecture service will handle chunking if needed
	segments := []Segment{
		{
			Start: 0,
			End:   0, // Unknown duration when using chat API
			Text:  transcribedText,
		},
	}

	return segments, metrics, nil
}
