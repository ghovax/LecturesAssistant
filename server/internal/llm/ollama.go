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

		for _, part := range message.Content {
			switch part.Type {
			case "text":
				contentBuilder.WriteString(part.Text)
			case "image":
				// Ollama expects base64 data without the data:image/... prefix
				data := part.ImageURL
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

	ollamaReq := ollamaChatRequest{
		Model:    request.Model,
		Messages: ollamaMessages,
		Stream:   request.Stream,
	}

	payload, jsonError := json.Marshal(ollamaReq)
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
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			var ollamaResp ollamaChatResponse
			if scanError := json.Unmarshal(line, &ollamaResp); scanError != nil {
				responseChannel <- ChatResponseChunk{Error: fmt.Errorf("failed to decode ollama response line: %w, line: %s", scanError, string(line))}
				return
			}

			chunk := ChatResponseChunk{
				Text: ollamaResp.Message.Content,
			}

			if ollamaResp.Done {
				chunk.InputTokens = ollamaResp.PromptEvalCount
				chunk.OutputTokens = ollamaResp.EvalCount
			}

			if chunk.Text != "" || ollamaResp.Done {
				responseChannel <- chunk
			}
		}

		if scanError := scanner.Err(); scanError != nil {
			responseChannel <- ChatResponseChunk{Error: scanError}
		}
	}()

	return responseChannel, nil
}
