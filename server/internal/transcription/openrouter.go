package transcription

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"lectures/internal/llm"
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

func (provider *OpenRouterTranscriptionProvider) Transcribe(jobContext context.Context, audioPath string) ([]Segment, error) {
	// Read audio file
	audioData, err := os.ReadFile(audioPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read audio file: %w", err)
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
		return nil, fmt.Errorf("LLM chat failed: %w", err)
	}

	// Collect response
	var transcriptionBuilder strings.Builder
	for chunk := range responseChannel {
		if chunk.Error != nil {
			return nil, fmt.Errorf("LLM streaming error: %w", chunk.Error)
		}
		transcriptionBuilder.WriteString(chunk.Text)
	}

	transcribedText := strings.TrimSpace(transcriptionBuilder.String())
	if transcribedText == "" {
		return nil, fmt.Errorf("no transcription received from LLM")
	}

	// Return as a single segment (OpenRouter doesn't provide timestamps via chat API)
	// The TranscribeLecture service will handle chunking if needed
	return []Segment{
		{
			Start: 0,
			End:   0, // Unknown duration when using chat API
			Text:  transcribedText,
		},
	}, nil
}
