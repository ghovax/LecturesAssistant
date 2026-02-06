package api

import (
	"encoding/json"
	"io"
	"net/http"
	"time"
)

// writeJSONResponse writes a JSON response to the ResponseWriter
func writeJSONResponse(responseWriter http.ResponseWriter, value interface{}) error {
	encoder := json.NewEncoder(responseWriter)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

// ProgressReader wraps an io.ReadCloser to track reading progress
type ProgressReader struct {
	Reader     io.ReadCloser
	Total      int64
	BytesRead  int64
	UploadID   string
	Hub        *Hub
	LastUpdate time.Time
	LastRead   int64
}

func (progressReader *ProgressReader) Read(p []byte) (n int, err error) {
	n, err = progressReader.Reader.Read(p)
	progressReader.BytesRead += int64(n)

	// Broadcast progress if it's been long enough or enough data read
	// We update every 100ms or every 1% of the total size, whichever is more frequent but not too frequent
	threshold := progressReader.Total / 100
	if threshold < 1024*1024 { // Minimum 1MB between updates if total is large
		threshold = 1024 * 1024
	}

	if time.Since(progressReader.LastUpdate) > 100*time.Millisecond || (progressReader.BytesRead-progressReader.LastRead) > threshold || err == io.EOF {
		progressReader.broadcast()
		progressReader.LastUpdate = time.Now()
		progressReader.LastRead = progressReader.BytesRead
	}

	return n, err
}

func (progressReader *ProgressReader) Close() error {
	return progressReader.Reader.Close()
}

func (progressReader *ProgressReader) broadcast() {
	if progressReader.Hub == nil || progressReader.UploadID == "" {
		return
	}

	progress := 0
	if progressReader.Total > 0 {
		progress = int(float64(progressReader.BytesRead) / float64(progressReader.Total) * 100)
	}

	if progress > 100 {
		progress = 100
	}

	progressReader.Hub.Broadcast(WSMessage{
		Type:    "upload:progress",
		Channel: "upload:" + progressReader.UploadID,
		Payload: map[string]any{
			"upload_id":   progressReader.UploadID,
			"bytes_read":  progressReader.BytesRead,
			"total_bytes": progressReader.Total,
			"progress":    progress,
		},
		Timestamp: time.Now().Format(time.RFC3339),
	})
}
