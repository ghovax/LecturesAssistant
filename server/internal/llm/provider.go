package llm

import (
	"context"
)

// ContentPart represents a part of a message (text or image)
type ContentPart struct {
	Type     string `json:"type"`                // "text" or "image"
	Text     string `json:"text,omitempty"`      // For type "text"
	ImageURL string `json:"image_url,omitempty"` // For type "image" (base64 or URL)
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
	Chat(context context.Context, request ChatRequest) (<-chan ChatResponseChunk, error)

	// Name returns the identifier of the provider
	Name() string
}
