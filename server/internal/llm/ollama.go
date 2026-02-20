package llm

import (
	"context"
	"encoding/base64"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/ollama/ollama/api"
)

type OllamaProvider struct {
	client *api.Client
}

func NewOllamaProvider(baseURL string) *OllamaProvider {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	parsedURL, parsingError := url.Parse(baseURL)
	if parsingError != nil {
		// Fallback to default if URL is invalid
		parsedURL, _ = url.Parse("http://localhost:11434")
	}

	return &OllamaProvider{
		client: api.NewClient(parsedURL, http.DefaultClient),
	}
}

func (provider *OllamaProvider) Name() string {
	return "ollama"
}

func (provider *OllamaProvider) Chat(jobContext context.Context, request *ChatRequest) (<-chan ChatResponseChunk, error) {
	// Safety check: ensure "ollama:" prefix is stripped
	modelName := strings.TrimPrefix(request.Model, "ollama:")
	request.Model = modelName

	responseChannel := make(chan ChatResponseChunk)

	var ollamaMessages []api.Message
	for _, message := range request.Messages {
		var contentBuilder strings.Builder
		var images []api.ImageData

		for _, contentPart := range message.Content {
			switch contentPart.Type {
			case "text":
				contentBuilder.WriteString(contentPart.Text)
			case "image":
				// Ollama expects base64 data without the data:image/... prefix
				imageData := contentPart.ImageURL
				if commaIndex := strings.Index(imageData, ","); commaIndex != -1 {
					imageData = imageData[commaIndex+1:]
				}

				decodedData, decodingError := base64.StdEncoding.DecodeString(imageData)
				if decodingError == nil {
					images = append(images, api.ImageData(decodedData))
				}
			}
		}

		ollamaMessages = append(ollamaMessages, api.Message{
			Role:    message.Role,
			Content: contentBuilder.String(),
			Images:  images,
		})
	}

	isStreaming := request.Stream
	ollamaRequest := &api.ChatRequest{
		Model:    request.Model,
		Messages: ollamaMessages,
		Stream:   &isStreaming,
		Options:  make(map[string]any),
	}

	// Set max_tokens (Ollama uses "num_predict" for this)
	if request.MaxTokens > 0 {
		ollamaRequest.Options["num_predict"] = request.MaxTokens
	}

	go func() {
		defer close(responseChannel)

		responseHandler := func(chatResponse api.ChatResponse) error {
			if chatResponse.Done {
				slog.Debug("Ollama request complete", "model", request.Model, "input_tokens", chatResponse.PromptEvalCount, "output_tokens", chatResponse.EvalCount)
			}

			responseChunk := ChatResponseChunk{
				Text: chatResponse.Message.Content,
			}
			if chatResponse.Done {
				responseChunk.InputTokens = chatResponse.PromptEvalCount
				responseChunk.OutputTokens = chatResponse.EvalCount
			}

			if responseChunk.Text != "" || chatResponse.Done {
				responseChannel <- responseChunk
			}
			return nil
		}

		chatError := provider.client.Chat(jobContext, ollamaRequest, responseHandler)
		if chatError != nil {
			responseChannel <- ChatResponseChunk{Error: chatError}
		}
	}()

	return responseChannel, nil
}
