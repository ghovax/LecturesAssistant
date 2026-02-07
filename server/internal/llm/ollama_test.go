package llm

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestOllamaProvider_Chat_Real is an integration test that requires a local Ollama instance
// running with the gemma3:1b model.
func TestOllamaProvider_Chat_Real(tester *testing.T) {
	// We use a short timeout to fail fast if Ollama is not running
	jobContext, cancelFunc := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelFunc()

	ollamaProvider := NewOllamaProvider("http://localhost:11434")

	chatRequest := ChatRequest{
		Model: "gemma3:1b",
		Messages: []Message{
			{
				Role: "user",
				Content: []ContentPart{
					{Type: "text", Text: "Please respond with exactly the words: 'Ollama integration test successful.'"},
				},
			},
		},
		Stream: false,
	}

	responseChannel, chatError := ollamaProvider.Chat(jobContext, &chatRequest)
	if chatError != nil {
		tester.Fatalf("Ollama test failed: could not start chat (is Ollama running?): %v", chatError)
		return
	}

	var responseBuilder strings.Builder

	// The ChatResponseChunk has an Error field
	hasError := false
	for responseChunk := range responseChannel {
		if responseChunk.Error != nil {
			tester.Logf("Error from Ollama: %v", responseChunk.Error)
			hasError = true
			break
		}
		responseBuilder.WriteString(responseChunk.Text)
	}

	if hasError {
		tester.Fatal("Ollama test failed due to runtime error (maybe model 'gemma3:1b' is not pulled?)")
		return
	}

	responseText := responseBuilder.String()
	if responseText == "" {
		tester.Error("Received empty response from Ollama")
	}

	tester.Logf("Ollama response: %s", responseText)
}

func TestRoutingProvider_Ollama(tester *testing.T) {
	// This test verifies that the RoutingProvider correctly handles the ollama: prefix
	mockOllama := &mockProvider{name: "ollama-mock"}
	routingProvider := NewRoutingProvider(nil)
	routingProvider.Register("ollama", mockOllama)

	jobContext := context.Background()
	chatRequest := ChatRequest{
		Model: "ollama:gemma3:1b",
		Messages: []Message{
			{Role: "user", Content: []ContentPart{{Type: "text", Text: "hi"}}},
		},
	}

	_, _ = routingProvider.Chat(jobContext, &chatRequest)

	if !mockOllama.called {
		tester.Error("RoutingProvider did not route to Ollama provider")
	}

	if chatRequest.Model != "gemma3:1b" {
		tester.Errorf("Expected model to be 'gemma3:1b', got '%s'", chatRequest.Model)
	}
}

type mockProvider struct {
	name   string
	called bool
}

func (mock *mockProvider) Chat(jobContext context.Context, chatRequest *ChatRequest) (<-chan ChatResponseChunk, error) {
	mock.called = true
	responseChannel := make(chan ChatResponseChunk, 1)
	responseChannel <- ChatResponseChunk{Text: "mock response"}
	close(responseChannel)
	return responseChannel, nil
}

func (mock *mockProvider) Name() string {
	return mock.name
}
