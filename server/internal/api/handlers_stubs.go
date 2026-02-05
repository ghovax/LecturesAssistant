package api

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"lectures/internal/models"

	"github.com/gorilla/mux"
)

// Health check
func (server *Server) handleHealth(responseWriter http.ResponseWriter, request *http.Request) {
	server.writeJSON(responseWriter, http.StatusOK, map[string]string{
		"status":  "healthy",
		"version": "1.0.0",
	})
}

// Transcripts
func (server *Server) handleTranscribe(responseWriter http.ResponseWriter, request *http.Request) {
	pathVariables := mux.Vars(request)
	lectureIdentifier := pathVariables["lectureId"]

	// Create transcription job
	jobIdentifier, err := server.jobQueue.Enqueue(models.JobTypeTranscribeMedia, map[string]string{
		"lecture_id": lectureIdentifier,
	})

	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "JOB_ERROR", "Failed to create transcription job", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusAccepted, map[string]string{
		"job_id":  jobIdentifier,
		"message": "Transcription job created",
	})
}

func (server *Server) handleGetTranscript(responseWriter http.ResponseWriter, request *http.Request) {
	pathVariables := mux.Vars(request)
	lectureIdentifier := pathVariables["lectureId"]

	// Get transcript
	var transcriptID, status string
	err := server.database.QueryRow(`
		SELECT id, status FROM transcripts WHERE lecture_id = ?
	`, lectureIdentifier).Scan(&transcriptID, &status)

	if err == sql.ErrNoRows {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Transcript not found", nil)
		return
	}
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get transcript", nil)
		return
	}

	// Get segments
	rows, err := server.database.Query(`
		SELECT id, transcript_id, media_id, start_millisecond, end_millisecond, text, confidence, speaker
		FROM transcript_segments
		WHERE transcript_id = ?
		ORDER BY start_millisecond ASC
	`, transcriptID)
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get segments", nil)
		return
	}
	defer rows.Close()

	segments := []map[string]interface{}{}
	for rows.Next() {
		var id int
		var transcriptID, mediaID, text, speaker sql.NullString
		var startMs, endMs int64
		var confidence sql.NullFloat64

		if err := rows.Scan(&id, &transcriptID, &mediaID, &startMs, &endMs, &text, &confidence, &speaker); err != nil {
			continue
		}

		segment := map[string]interface{}{
			"id":                id,
			"start_millisecond": startMs,
			"end_millisecond":   endMs,
			"text":              text.String,
		}
		if confidence.Valid {
			segment["confidence"] = confidence.Float64
		}
		if speaker.Valid {
			segment["speaker"] = speaker.String
		}
		segments = append(segments, segment)
	}

	server.writeJSON(responseWriter, http.StatusOK, map[string]interface{}{
		"transcript_id": transcriptID,
		"status":        status,
		"segments":      segments,
	})
}

// Documents
func (server *Server) handleUploadDocument(responseWriter http.ResponseWriter, request *http.Request) {
	server.writeError(responseWriter, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Document upload not yet implemented", nil)
}

func (server *Server) handleListDocuments(responseWriter http.ResponseWriter, request *http.Request) {
	server.writeJSON(responseWriter, http.StatusOK, []interface{}{})
}

func (server *Server) handleGetDocument(responseWriter http.ResponseWriter, request *http.Request) {
	server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Document not found", nil)
}

func (server *Server) handleDeleteDocument(responseWriter http.ResponseWriter, request *http.Request) {
	server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Document not found", nil)
}

// Tools
func (server *Server) handleCreateTool(responseWriter http.ResponseWriter, request *http.Request) {
	server.writeError(responseWriter, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Tool creation not yet implemented", nil)
}

func (server *Server) handleListTools(responseWriter http.ResponseWriter, request *http.Request) {
	server.writeJSON(responseWriter, http.StatusOK, []interface{}{})
}

func (server *Server) handleGetTool(responseWriter http.ResponseWriter, request *http.Request) {
	server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Tool not found", nil)
}

func (server *Server) handleDeleteTool(responseWriter http.ResponseWriter, request *http.Request) {
	server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Tool not found", nil)
}

// Chat
func (server *Server) handleCreateChatSession(responseWriter http.ResponseWriter, request *http.Request) {
	server.writeError(responseWriter, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Chat not yet implemented", nil)
}

func (server *Server) handleListChatSessions(responseWriter http.ResponseWriter, request *http.Request) {
	server.writeJSON(responseWriter, http.StatusOK, []interface{}{})
}

func (server *Server) handleGetChatSession(responseWriter http.ResponseWriter, request *http.Request) {
	server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Chat session not found", nil)
}

func (server *Server) handleDeleteChatSession(responseWriter http.ResponseWriter, request *http.Request) {
	server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Chat session not found", nil)
}

func (server *Server) handleSendMessage(responseWriter http.ResponseWriter, request *http.Request) {
	server.writeError(responseWriter, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Chat not yet implemented", nil)
}

// Jobs
func (server *Server) handleListJobs(responseWriter http.ResponseWriter, request *http.Request) {
	rows, err := server.database.Query(`
		SELECT id, type, status, progress, progress_message_text, created_at
		FROM jobs
		ORDER BY created_at DESC
		LIMIT 50
	`)
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to list jobs", nil)
		return
	}
	defer rows.Close()

	jobs := []map[string]interface{}{}
	for rows.Next() {
		var id, jobType, status, progressMsg string
		var progress int
		var createdAt string

		if err := rows.Scan(&id, &jobType, &status, &progress, &progressMsg, &createdAt); err != nil {
			continue
		}

		jobs = append(jobs, map[string]interface{}{
			"id":                    id,
			"type":                  jobType,
			"status":                status,
			"progress":              progress,
			"progress_message_text": progressMsg,
			"created_at":            createdAt,
		})
	}

	server.writeJSON(responseWriter, http.StatusOK, jobs)
}

func (server *Server) handleGetJob(responseWriter http.ResponseWriter, request *http.Request) {
	pathVariables := mux.Vars(request)
	jobIdentifier := pathVariables["id"]

	job, err := server.jobQueue.GetJob(jobIdentifier)
	if err != nil {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Job not found", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusOK, job)
}

func (server *Server) handleCancelJob(responseWriter http.ResponseWriter, request *http.Request) {
	pathVariables := mux.Vars(request)
	jobIdentifier := pathVariables["id"]

	if err := server.jobQueue.CancelJob(jobIdentifier); err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "JOB_ERROR", "Failed to cancel job", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusOK, map[string]string{"message": "Job cancelled"})
}

// Settings
func (server *Server) handleGetSettings(responseWriter http.ResponseWriter, request *http.Request) {
	server.writeJSON(responseWriter, http.StatusOK, map[string]interface{}{
		"llm": map[string]string{
			"provider": server.configuration.LLM.Provider,
			"model":    server.configuration.LLM.OpenRouter.DefaultModel,
		},
		"transcription": map[string]string{
			"provider": server.configuration.Transcription.Provider,
		},
	})
}

func (server *Server) handleUpdateSettings(responseWriter http.ResponseWriter, request *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(request.Body).Decode(&req); err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	// TODO: Update settings in database and config
	server.writeJSON(responseWriter, http.StatusOK, req)
}

// WebSocket
func (server *Server) handleWebSocket(responseWriter http.ResponseWriter, request *http.Request) {
	// TODO: Implement WebSocket protocol
	server.writeError(responseWriter, http.StatusNotImplemented, "NOT_IMPLEMENTED", "WebSocket not yet implemented", nil)
}
