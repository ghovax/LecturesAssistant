package llm

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"

	openrouter "github.com/revrost/go-openrouter"
)

type OpenRouterProvider struct {
	client      *openrouter.Client
	clientMutex sync.RWMutex
}

func NewOpenRouterProvider(apiKey string) *OpenRouterProvider {
	return &OpenRouterProvider{
		client: openrouter.NewClient(apiKey),
	}
}

func (provider *OpenRouterProvider) SetAPIKey(apiKey string) {
	provider.clientMutex.Lock()
	defer provider.clientMutex.Unlock()
	provider.client = openrouter.NewClient(apiKey)
}

func (provider *OpenRouterProvider) Name() string {
	return "openrouter"
}

func (provider *OpenRouterProvider) Chat(jobContext context.Context, request *ChatRequest) (<-chan ChatResponseChunk, error) {
	provider.clientMutex.RLock()
	client := provider.client
	provider.clientMutex.RUnlock()

	// Safety check: ensure "openrouter:" prefix is stripped
	modelName := strings.TrimPrefix(request.Model, "openrouter:")
	request.Model = modelName

	responseChannel := make(chan ChatResponseChunk)

	var chatMessages []openrouter.ChatCompletionMessage
	for _, message := range request.Messages {
		var contentParts []openrouter.ChatMessagePart
		for _, contentPart := range message.Content {
			switch contentPart.Type {
			case "text":
				contentParts = append(contentParts, openrouter.ChatMessagePart{
					Type: openrouter.ChatMessagePartTypeText,
					Text: contentPart.Text,
				})
			case "image":
				contentParts = append(contentParts, openrouter.ChatMessagePart{
					Type: "image_url", // Based on common patterns, though type alias exists
					ImageURL: &openrouter.ChatMessageImageURL{
						URL: contentPart.ImageURL,
					},
				})
			case "input_audio":
				contentParts = append(contentParts, openrouter.ChatMessagePart{
					Type: openrouter.ChatMessagePartTypeInputAudio,
					InputAudio: &openrouter.ChatMessageInputAudio{
						Data:   contentPart.AudioData,
						Format: openrouter.AudioFormat(contentPart.AudioFormat),
					},
				})
			}
		}
		chatMessages = append(chatMessages, openrouter.ChatCompletionMessage{
			Role: message.Role,
			Content: openrouter.Content{
				Multi: contentParts,
			},
		})
	}

	go func() {
		defer close(responseChannel)

		if request.Stream {
			completionStream, streamError := client.CreateChatCompletionStream(jobContext, openrouter.ChatCompletionRequest{
				Model:    request.Model,
				Messages: chatMessages,
				Stream:   true,
			})
			if streamError != nil {
				responseChannel <- ChatResponseChunk{Error: streamError}
				return
			}
			defer completionStream.Close()

			for {
				chatResponse, receiveError := completionStream.Recv()
				if receiveError != nil {
					if errors.Is(receiveError, io.EOF) {
						return
					}
					responseChannel <- ChatResponseChunk{Error: receiveError}
					return
				}
				if len(chatResponse.Choices) > 0 {
					responseContent := chatResponse.Choices[0].Delta.Content
					responseChunk := ChatResponseChunk{Text: responseContent}
					if chatResponse.Usage != nil {
						responseChunk.InputTokens = chatResponse.Usage.PromptTokens
						responseChunk.OutputTokens = chatResponse.Usage.CompletionTokens
						responseChunk.Cost = chatResponse.Usage.Cost
					}
					if responseContent != "" || chatResponse.Usage != nil {
						responseChannel <- responseChunk
					}
				}
			}
		} else {
			chatResponse, chatError := client.CreateChatCompletion(jobContext, openrouter.ChatCompletionRequest{
				Model:    request.Model,
				Messages: chatMessages,
			})
			if chatError != nil {
				responseChannel <- ChatResponseChunk{Error: chatError}
				return
			}
			if len(chatResponse.Choices) > 0 {
				responseChunk := ChatResponseChunk{
					Text:         chatResponse.Choices[0].Message.Content.Text,
					InputTokens:  chatResponse.Usage.PromptTokens,
					OutputTokens: chatResponse.Usage.CompletionTokens,
					Cost:         chatResponse.Usage.Cost,
				}
				responseChannel <- responseChunk
			}
		}
	}()

	return responseChannel, nil
}
