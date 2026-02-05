package llm

import (
	"context"
	"errors"
	"io"

	openrouter "github.com/revrost/go-openrouter"
)

type OpenRouterProvider struct {
	client *openrouter.Client
}

func NewOpenRouterProvider(apiKey string) *OpenRouterProvider {
	return &OpenRouterProvider{
		client: openrouter.NewClient(apiKey),
	}
}

func (provider *OpenRouterProvider) Name() string {
	return "openrouter"
}

func (provider *OpenRouterProvider) Chat(jobContext context.Context, request ChatRequest) (<-chan ChatResponseChunk, error) {
	responseChannel := make(chan ChatResponseChunk)

	var chatMessages []openrouter.ChatCompletionMessage
	for _, message := range request.Messages {
		var contentParts []openrouter.ChatMessagePart
		for _, part := range message.Content {
			switch part.Type {
			case "text":
				contentParts = append(contentParts, openrouter.ChatMessagePart{
					Type: openrouter.ChatMessagePartTypeText,
					Text: part.Text,
				})
			case "image":
				contentParts = append(contentParts, openrouter.ChatMessagePart{
					Type: "image_url", // Based on common patterns, though type alias exists
					ImageURL: &openrouter.ChatMessageImageURL{
						URL: part.ImageURL,
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
			stream, err := provider.client.CreateChatCompletionStream(jobContext, openrouter.ChatCompletionRequest{
				Model:    request.Model,
				Messages: chatMessages,
				Stream:   true,
			})
			if err != nil {
				responseChannel <- ChatResponseChunk{Error: err}
				return
			}
			defer stream.Close()

			for {
				response, err := stream.Recv()
				if err != nil {
					if errors.Is(err, io.EOF) {
						return
					}
					responseChannel <- ChatResponseChunk{Error: err}
					return
				}
				if len(response.Choices) > 0 {
					content := response.Choices[0].Delta.Content
					chunk := ChatResponseChunk{Text: content}
					if response.Usage != nil {
						chunk.InputTokens = response.Usage.PromptTokens
						chunk.OutputTokens = response.Usage.CompletionTokens
						chunk.Cost = response.Usage.Cost
					}
					if content != "" || response.Usage != nil {
						responseChannel <- chunk
					}
				}
			}
		} else {
			response, err := provider.client.CreateChatCompletion(jobContext, openrouter.ChatCompletionRequest{
				Model:    request.Model,
				Messages: chatMessages,
			})
			if err != nil {
				responseChannel <- ChatResponseChunk{Error: err}
				return
			}
			if len(response.Choices) > 0 {
				chunk := ChatResponseChunk{
					Text:         response.Choices[0].Message.Content.Text,
					InputTokens:  response.Usage.PromptTokens,
					OutputTokens: response.Usage.CompletionTokens,
					Cost:         response.Usage.Cost,
				}
				responseChannel <- chunk
			}
		}
	}()

	return responseChannel, nil
}
