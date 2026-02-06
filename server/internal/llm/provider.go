package llm

import (
	"context"
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
	Chat(context context.Context, request ChatRequest) (<-chan ChatResponseChunk, error)

	// Name returns the identifier of the provider
	Name() string
}
