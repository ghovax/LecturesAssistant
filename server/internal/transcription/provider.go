package transcription

import "context"

// Segment represents a portion of the transcript
type Segment struct {
	Start      float64 `json:"start"` // Start time in seconds relative to the beginning of the audio file
	End        float64 `json:"end"`   // End time in seconds
	Text       string  `json:"text"`
	Confidence float64 `json:"confidence,omitempty"`
	Speaker    string  `json:"speaker,omitempty"`
}

// Provider defines the interface for different transcription services
type Provider interface {
	// Transcribe processes an audio file and returns a list of segments
	Transcribe(context context.Context, audioPath string) ([]Segment, error)

	// CheckDependencies verifies that necessary external tools are installed
	CheckDependencies() error

	// Name returns the identifier of the provider
	Name() string
}
