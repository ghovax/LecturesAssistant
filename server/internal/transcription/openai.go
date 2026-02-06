package transcription

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

type OpenAIProvider struct {
	apiKey  string
	baseURL string
	model   string
	prompt  string
}

func NewOpenAIProvider(apiKey, baseURL, model string) *OpenAIProvider {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if model == "" {
		model = "whisper-1"
	}
	return &OpenAIProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
	}
}

func (provider *OpenAIProvider) SetPrompt(prompt string) {
	provider.prompt = prompt
}

func (provider *OpenAIProvider) Name() string {
	return "openai"
}

func (provider *OpenAIProvider) CheckDependencies() error {
	if provider.apiKey == "" {
		return fmt.Errorf("OpenAI API key is missing")
	}
	return nil
}

type openAITranscriptionResponse struct {
	Text     string `json:"text"`
	Segments []struct {
		Start float64 `json:"start"`
		End   float64 `json:"end"`
		Text  string  `json:"text"`
	} `json:"segments"`
}

func (provider *OpenAIProvider) Transcribe(jobContext context.Context, audioPath string) ([]Segment, error) {
	file, err := os.Open(audioPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filepath.Base(audioPath))
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, err
	}

	writer.WriteField("model", provider.model)
	if provider.prompt != "" {
		writer.WriteField("prompt", provider.prompt)
	}
	// Request verbose_json to get segments
	writer.WriteField("response_format", "verbose_json")

	if err := writer.Close(); err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(jobContext, "POST", provider.baseURL+"/audio/transcriptions", body)
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Authorization", "Bearer "+provider.apiKey)

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		errorBody, _ := io.ReadAll(response.Body)
		return nil, fmt.Errorf("OpenAI API returned status %d: %s", response.StatusCode, string(errorBody))
	}

	var openAIResp openAITranscriptionResponse
	if err := json.NewDecoder(response.Body).Decode(&openAIResp); err != nil {
		return nil, err
	}

	var segments []Segment
	if len(openAIResp.Segments) > 0 {
		for _, segment := range openAIResp.Segments {
			segments = append(segments, Segment{
				Start: segment.Start,
				End:   segment.End,
				Text:  segment.Text,
			})
		}
	} else if openAIResp.Text != "" {
		// Fallback if no segments provided
		segments = append(segments, Segment{
			Start: 0,
			End:   0, // Unknown
			Text:  openAIResp.Text,
		})
	}

	return segments, nil
}
