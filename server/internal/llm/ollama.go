package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type OllamaProvider struct {
	baseURL string
}

func NewOllamaProvider(baseURL string) *OllamaProvider {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	return &OllamaProvider{
		baseURL: baseURL,
	}
}

func (provider *OllamaProvider) Name() string {
	return "ollama"
}

type ollamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

type ollamaMessage struct {
	Role    string   `json:"role"`
	Content string   `json:"content"`
	Images  []string `json:"images,omitempty"`
}

type ollamaChatResponse struct {
	Model     string        `json:"model"`
	CreatedAt string        `json:"created_at"`
	Message   ollamaMessage `json:"message"`
	Done      bool          `json:"done"`
	// Usage metrics
	PromptEvalCount int `json:"prompt_eval_count"`
	EvalCount       int `json:"eval_count"`
}

func (provider *OllamaProvider) Chat(jobContext context.Context, request ChatRequest) (<-chan ChatResponseChunk, error) {
	responseChannel := make(chan ChatResponseChunk)

	var ollamaMessages []ollamaMessage
	for _, message := range request.Messages {
		var contentBuilder bytes.Buffer
		var images []string

		for _, contentPart := range message.Content {
			switch contentPart.Type {
			case "text":
				contentBuilder.WriteString(contentPart.Text)
			case "image":
				// Ollama expects base64 data without the data:image/... prefix
				data := contentPart.ImageURL
				if commaIndex := bytes.IndexByte([]byte(data), ','); commaIndex != -1 {
					data = data[commaIndex+1:]
				}
				images = append(images, data)
			}
		}

		ollamaMessages = append(ollamaMessages, ollamaMessage{
			Role:    message.Role,
			Content: contentBuilder.String(),
			Images:  images,
		})
	}

	ollamaRequestPayload := ollamaChatRequest{
		Model:    request.Model,
		Messages: ollamaMessages,
		Stream:   request.Stream,
	}

	payload, jsonError := json.Marshal(ollamaRequestPayload)
	if jsonError != nil {
		return nil, fmt.Errorf("failed to marshal ollama request: %w", jsonError)
	}

	go func() {
		defer close(responseChannel)

		httpRequest, requestError := http.NewRequestWithContext(jobContext, "POST", provider.baseURL+"/api/chat", bytes.NewBuffer(payload))
		if requestError != nil {
			responseChannel <- ChatResponseChunk{Error: requestError}
			return
		}
		httpRequest.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		httpResponse, executionError := client.Do(httpRequest)
		if executionError != nil {
			responseChannel <- ChatResponseChunk{Error: executionError}
			return
		}
		defer httpResponse.Body.Close()

		if httpResponse.StatusCode != http.StatusOK {
			var errorBody bytes.Buffer
			io.Copy(&errorBody, httpResponse.Body)
			responseChannel <- ChatResponseChunk{Error: fmt.Errorf("ollama API returned status %d: %s", httpResponse.StatusCode, errorBody.String())}
			return
		}

		scanner := bufio.NewScanner(httpResponse.Body)
		for scanner.Scan() {
			responseLine := scanner.Bytes()
			if len(responseLine) == 0 {
				continue
			}

			var ollamaResponse ollamaChatResponse
			if scanningError := json.Unmarshal(responseLine, &ollamaResponse); scanningError != nil {
				responseChannel <- ChatResponseChunk{Error: fmt.Errorf("failed to decode ollama response line: %w, line: %s", scanningError, string(responseLine))}
				return
			}

			chunk := ChatResponseChunk{
				Text: ollamaResponse.Message.Content,
			}

			if ollamaResponse.Done {
				chunk.InputTokens = ollamaResponse.PromptEvalCount
				chunk.OutputTokens = ollamaResponse.EvalCount
			}

			if chunk.Text != "" || ollamaResponse.Done {
				responseChannel <- chunk
			}
		}

		if scanningError := scanner.Err(); scanningError != nil {
			responseChannel <- ChatResponseChunk{Error: scanningError}
		}
	}()

	return responseChannel, nil
}
