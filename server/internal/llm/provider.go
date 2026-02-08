package llm

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
)

// ContentPart represents a part of a message (text, image, or audio)
type ContentPart struct {
	Type        string `json:"type"`                   // "text", "image", or "input_audio"
	Text        string `json:"text,omitempty"`         // For type "text"
	ImageURL    string `json:"image_url,omitempty"`    // For type "image" (base64 or URL)
	AudioData   string `json:"audio_data,omitempty"`   // For type "input_audio" (base64-encoded)
	AudioFormat string `json:"audio_format,omitempty"` // For type "input_audio" (e.g., "wav", "mp3")
}

// Message represents a chat message
type Message struct {
	Role    string        `json:"role"`    // "system", "user", "assistant"
	Content []ContentPart `json:"content"` // Multimodal content
}

// ChatRequest represents a request to the LLM
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

// ChatResponseChunk represents a chunk of the streamed response
type ChatResponseChunk struct {
	Text         string  `json:"text"`
	InputTokens  int     `json:"input_tokens,omitempty"`
	OutputTokens int     `json:"output_tokens,omitempty"`
	Cost         float64 `json:"cost,omitempty"`
	Error        error   `json:"error,omitempty"`
}

// Provider defines the common interface for LLM services
type Provider interface {
	// Chat sends a request to the LLM and returns a stream of response chunks
	Chat(context context.Context, request *ChatRequest) (<-chan ChatResponseChunk, error)

	// Name returns the identifier of the provider
	Name() string
}

// RoutingProvider routes requests to different providers based on a prefix or default
type RoutingProvider struct {
	providers       map[string]Provider
	defaultProvider Provider
	providersMutex  sync.RWMutex
}

func NewRoutingProvider(defaultProvider Provider) *RoutingProvider {
	return &RoutingProvider{
		providers:       make(map[string]Provider),
		defaultProvider: defaultProvider,
	}
}

func (routingProvider *RoutingProvider) Register(name string, provider Provider) {
	routingProvider.providersMutex.Lock()
	defer routingProvider.providersMutex.Unlock()
	routingProvider.providers[name] = provider
}

func (routingProvider *RoutingProvider) GetProvider(name string) Provider {
	routingProvider.providersMutex.RLock()
	defer routingProvider.providersMutex.RUnlock()
	return routingProvider.providers[name]
}

func (routingProvider *RoutingProvider) Chat(jobContext context.Context, request *ChatRequest) (<-chan ChatResponseChunk, error) {
	originalModelName := request.Model
	providerName := ""
	modelName := originalModelName

	// Check for provider prefix (e.g., "openrouter:google/gemini-2.5-flash-lite")
	if strings.Contains(originalModelName, ":") {
		parts := strings.SplitN(originalModelName, ":", 2)
		potentialProvider := parts[0]

		// Check if it's a known registered provider
		routingProvider.providersMutex.RLock()
		_, exists := routingProvider.providers[potentialProvider]
		routingProvider.providersMutex.RUnlock()

		// Also allow "openrouter" and "ollama" as hardcoded known prefixes
		if exists || potentialProvider == "openrouter" || potentialProvider == "ollama" {
			providerName = potentialProvider
			modelName = parts[1]
			request.Model = modelName // Strip the prefix from the request
			slog.Info("Routing LLM request with prefix stripping", "model", modelName)
		}
	}

	// Route to specific provider if prefix was found
	if providerName != "" {
		routingProvider.providersMutex.RLock()
		provider, exists := routingProvider.providers[providerName]
		routingProvider.providersMutex.RUnlock()

		if exists {
			return provider.Chat(jobContext, request)
		}

		// If prefix matched "openrouter" or "ollama" but wasn't in the map,
		// fall back to default if it matches the name
		if routingProvider.defaultProvider != nil && routingProvider.defaultProvider.Name() == providerName {
			return routingProvider.defaultProvider.Chat(jobContext, request)
		}
	}

	// Fallback to default provider
	if routingProvider.defaultProvider != nil {
		slog.Debug("Routing LLM request to default provider", "provider", routingProvider.defaultProvider.Name(), "model", request.Model)
		return routingProvider.defaultProvider.Chat(jobContext, request)
	}

	return nil, fmt.Errorf("no LLM provider found for: %s", originalModelName)
}
func (routingProvider *RoutingProvider) Name() string {
	return "routing-provider"
}
