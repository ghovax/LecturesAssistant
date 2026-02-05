package models

import "time"

// Exam represents a course or exam grouping
type Exam struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Lecture represents a single lesson or session
type Lecture struct {
	ID          string    `json:"id"`
	ExamID      string    `json:"exam_id"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Status      string    `json:"status"` // "processing", "ready", "failed"
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// LectureMedia represents audio or video files
type LectureMedia struct {
	ID                   string    `json:"id"`
	LectureID            string    `json:"lecture_id"`
	MediaType            string    `json:"media_type"` // "audio" or "video"
	SequenceOrder        int       `json:"sequence_order"`
	DurationMilliseconds int64     `json:"duration_milliseconds,omitempty"`
	FilePath             string    `json:"file_path"`
	CreatedAt            time.Time `json:"created_at"`
}

// Transcript represents a unified transcript from all media files
type Transcript struct {
	ID         string    `json:"id"`
	LectureID  string    `json:"lecture_id"`
	Language   string    `json:"language,omitempty"`
	Status     string    `json:"status"`
	Confidence float64   `json:"confidence"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// TranscriptSegment represents a segment of the transcript
type TranscriptSegment struct {
	ID                        int     `json:"id"`
	TranscriptID              string  `json:"transcript_id"`
	MediaID                   string  `json:"media_id,omitempty"`
	StartMillisecond          int64   `json:"start_millisecond"`
	EndMillisecond            int64   `json:"end_millisecond"`
	OriginalStartMilliseconds int64   `json:"original_start_milliseconds,omitempty"`
	OriginalEndMilliseconds   int64   `json:"original_end_milliseconds,omitempty"`
	Text                      string  `json:"text"`
	Confidence                float64 `json:"confidence,omitempty"`
	Speaker                   string  `json:"speaker,omitempty"`
}

// ReferenceDocument represents a PDF, PowerPoint, or other document
type ReferenceDocument struct {
	ID               string    `json:"id"`
	LectureID        string    `json:"lecture_id"`
	DocumentType     string    `json:"document_type"`
	Title            string    `json:"title"`
	FilePath         string    `json:"file_path"`
	PageCount        int       `json:"page_count"`
	ExtractionStatus string    `json:"extraction_status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// ReferencePage represents a page extracted from a document
type ReferencePage struct {
	ID            int    `json:"id"`
	DocumentID    string `json:"document_id"`
	PageNumber    int    `json:"page_number"`
	ImagePath     string `json:"image_path"`
	ExtractedText string `json:"extracted_text,omitempty"`
}

// Tool represents AI-generated study materials
type Tool struct {
	ID        string    `json:"id"`
	ExamID    string    `json:"exam_id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Content   string    `json:"content"` // JSON string
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ChatSession represents a conversation scoped to an exam
type ChatSession struct {
	ID        string    `json:"id"`
	ExamID    string    `json:"exam_id"`
	Title     string    `json:"title,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ChatMessage represents a single message in a chat session
type ChatMessage struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Role      string    `json:"role"` // "user", "assistant", "system"
	Content   string    `json:"content"`
	ModelUsed string    `json:"model_used,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Job represents a background task
type Job struct {
	ID                  string     `json:"id"`
	Type                string     `json:"type"`
	Status              string     `json:"status"`
	Progress            int        `json:"progress"`
	ProgressMessageText string     `json:"progress_message_text,omitempty"`
	Payload             string     `json:"payload"`          // JSON string
	Result              string     `json:"result,omitempty"` // JSON string
	Error               string     `json:"error,omitempty"`
	Metadata            any        `json:"metadata,omitempty"` // Additional context for progress
	InputTokens         int        `json:"input_tokens,omitempty"`
	OutputTokens        int        `json:"output_tokens,omitempty"`
	EstimatedCost       float64    `json:"estimated_cost,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	StartedAt           *time.Time `json:"started_at,omitempty"`
	CompletedAt         *time.Time `json:"completed_at,omitempty"`
}

// JobType constants
const (
	JobTypeTranscribeMedia = "TRANSCRIBE_MEDIA"
	JobTypeIngestDocuments = "INGEST_DOCUMENTS"
	JobTypeBuildMaterial   = "BUILD_MATERIAL"
	JobTypePublishMaterial = "PUBLISH_MATERIAL"
)

// JobStatus constants
const (
	JobStatusPending   = "PENDING"
	JobStatusRunning   = "RUNNING"
	JobStatusCompleted = "COMPLETED"
	JobStatusFailed    = "FAILED"
	JobStatusCancelled = "CANCELLED"
)

// APIResponse represents a standard API response
type APIResponse struct {
	Data interface{} `json:"data,omitempty"`
	Meta Meta        `json:"meta"`
}

// APIError represents a standard API error response
type APIError struct {
	Error ErrorDetails `json:"error"`
	Meta  Meta         `json:"meta"`
}

// Meta contains metadata for API responses
type Meta struct {
	Timestamp string `json:"timestamp"`
	RequestID string `json:"request_id"`
}

// ErrorDetails contains error information
type ErrorDetails struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}
