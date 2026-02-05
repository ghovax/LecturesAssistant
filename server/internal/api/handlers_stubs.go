package api

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"lectures/internal/models"

	"github.com/gorilla/mux"
)

// Health check
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]string{
		"status":  "healthy",
		"version": "1.0.0",
	})
}

// Transcripts
func (s *Server) handleTranscribe(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	lectureID := vars["lecture_id"]

	// Create transcription job
	jobID, err := s.jobQueue.Enqueue(models.JobTypeTranscribeMedia, map[string]string{
		"lecture_id": lectureID,
	})

	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "JOB_ERROR", "Failed to create transcription job", nil)
		return
	}

	s.writeJSON(w, http.StatusAccepted, map[string]string{
		"job_id":  jobID,
		"message": "Transcription job created",
	})
}

func (s *Server) handleGetTranscript(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	lectureID := vars["lecture_id"]

	// Get transcript
	var transcriptID, status string
	err := s.db.QueryRow(`
		SELECT id, status FROM transcripts WHERE lecture_id = ?
	`, lectureID).Scan(&transcriptID, &status)

	if err == sql.ErrNoRows {
		s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Transcript not found", nil)
		return
	}
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get transcript", nil)
		return
	}

	// Get segments
	rows, err := s.db.Query(`
		SELECT id, transcript_id, media_id, start_millisecond, end_millisecond, text, confidence, speaker
		FROM transcript_segments
		WHERE transcript_id = ?
		ORDER BY start_millisecond ASC
	`, transcriptID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get segments", nil)
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

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"transcript_id": transcriptID,
		"status":        status,
		"segments":      segments,
	})
}

// Documents
func (s *Server) handleUploadDocument(w http.ResponseWriter, r *http.Request) {
	s.writeError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Document upload not yet implemented", nil)
}

func (s *Server) handleListDocuments(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, []interface{}{})
}

func (s *Server) handleGetDocument(w http.ResponseWriter, r *http.Request) {
	s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Document not found", nil)
}

func (s *Server) handleDeleteDocument(w http.ResponseWriter, r *http.Request) {
	s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Document not found", nil)
}

// Tools
func (s *Server) handleCreateTool(w http.ResponseWriter, r *http.Request) {
	s.writeError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Tool creation not yet implemented", nil)
}

func (s *Server) handleListTools(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, []interface{}{})
}

func (s *Server) handleGetTool(w http.ResponseWriter, r *http.Request) {
	s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Tool not found", nil)
}

func (s *Server) handleDeleteTool(w http.ResponseWriter, r *http.Request) {
	s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Tool not found", nil)
}

// Chat
func (s *Server) handleCreateChatSession(w http.ResponseWriter, r *http.Request) {
	s.writeError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Chat not yet implemented", nil)
}

func (s *Server) handleListChatSessions(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, []interface{}{})
}

func (s *Server) handleGetChatSession(w http.ResponseWriter, r *http.Request) {
	s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Chat session not found", nil)
}

func (s *Server) handleDeleteChatSession(w http.ResponseWriter, r *http.Request) {
	s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Chat session not found", nil)
}

func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	s.writeError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Chat not yet implemented", nil)
}

// Jobs
func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(`
		SELECT id, type, status, progress, progress_message_text, created_at
		FROM jobs
		ORDER BY created_at DESC
		LIMIT 50
	`)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to list jobs", nil)
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

	s.writeJSON(w, http.StatusOK, jobs)
}

func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	job, err := s.jobQueue.GetJob(jobID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Job not found", nil)
		return
	}

	s.writeJSON(w, http.StatusOK, job)
}

func (s *Server) handleCancelJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["id"]

	if err := s.jobQueue.CancelJob(jobID); err != nil {
		s.writeError(w, http.StatusInternalServerError, "JOB_ERROR", "Failed to cancel job", nil)
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{"message": "Job cancelled"})
}

// Settings
func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"llm": map[string]string{
			"provider": s.config.LLM.Provider,
			"model":    s.config.LLM.OpenRouter.DefaultModel,
		},
		"transcription": map[string]string{
			"provider": s.config.Transcription.Provider,
		},
	})
}

func (s *Server) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	// TODO: Update settings in database and config
	s.writeJSON(w, http.StatusOK, req)
}

// WebSocket
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement WebSocket protocol
	s.writeError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "WebSocket not yet implemented", nil)
}
