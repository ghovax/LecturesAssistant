package api

import (
	"encoding/json"
	"net/http"

	"lectures/internal/models"
)

// handleImport handles requests to import files from external providers (Google Drive, Dropbox, etc.)
func (server *Server) handleImport(responseWriter http.ResponseWriter, request *http.Request) {
	var importRequest struct {
		Source   string          `json:"source"`   // e.g., "google_drive", "dropbox"
		Filename string          `json:"filename"` // Optional override
		Data     json.RawMessage `json:"data"`     // Provider-specific data
	}

	if decodingError := json.NewDecoder(request.Body).Decode(&importRequest); decodingError != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	if importRequest.Source == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "source is required", nil)
		return
	}

	userID := server.getUserID(request)
	var jobIdentifier string
	var enqueuingError error

	switch importRequest.Source {
	case "google_drive":
		// Parse Google Drive specific data
		var driveData struct {
			FileID     string `json:"file_id"`
			OAuthToken string `json:"oauth_token"`
		}
		if err := json.Unmarshal(importRequest.Data, &driveData); err != nil {
			server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid data for google_drive source", nil)
			return
		}

		if driveData.FileID == "" || driveData.OAuthToken == "" {
			server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "file_id and oauth_token are required for google_drive", nil)
			return
		}

		// Enqueue the specific Google Drive download job
		jobIdentifier, enqueuingError = server.jobQueue.Enqueue(userID, models.JobTypeDownloadGoogleDrive, map[string]string{
			"file_id":     driveData.FileID,
			"oauth_token": driveData.OAuthToken,
			"filename":    importRequest.Filename,
		}, "", "")

	// Future providers can be added here
	// case "dropbox":
	//     ...

	default:
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Unsupported import source: "+importRequest.Source, nil)
		return
	}

	if enqueuingError != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "BACKGROUND_JOB_ERROR", "Failed to enqueue download job", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusAccepted, map[string]string{
		"job_id":  jobIdentifier,
		"message": "Import job created",
	})
}
