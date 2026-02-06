package api

import (
	"encoding/json"
	"net/http"
)

// handleListJobs lists recent background jobs for the current user
func (server *Server) handleListJobs(responseWriter http.ResponseWriter, request *http.Request) {
	userID := server.getUserID(request)

	jobRows, databaseError := server.database.Query(`
		SELECT id, type, status, progress, progress_message_text, input_tokens, output_tokens, estimated_cost, created_at
		FROM jobs
		WHERE user_id = ?
		ORDER BY created_at DESC
		LIMIT 50
	`, userID)
	if databaseError != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to list jobs", nil)
		return
	}
	defer jobRows.Close()

	var jobsList []map[string]any
	for jobRows.Next() {
		var id, jobType, status, progressMsg string
		var progress, inputTokens, outputTokens int
		var estimatedCost float64
		var createdAt string

		if err := jobRows.Scan(&id, &jobType, &status, &progress, &progressMsg, &inputTokens, &outputTokens, &estimatedCost, &createdAt); err != nil {
			continue
		}

		jobsList = append(jobsList, map[string]any{
			"id":                    id,
			"type":                  jobType,
			"status":                status,
			"progress":              progress,
			"progress_message_text": progressMsg,
			"input_tokens":          inputTokens,
			"output_tokens":         outputTokens,
			"estimated_cost":        estimatedCost,
			"created_at":            createdAt,
		})
	}

	server.writeJSON(responseWriter, http.StatusOK, jobsList)
}

// handleGetJob retrieves detailed status of a specific job
func (server *Server) handleGetJob(responseWriter http.ResponseWriter, request *http.Request) {
	jobID := request.URL.Query().Get("job_id")
	if jobID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "job_id is required", nil)
		return
	}

	userID := server.getUserID(request)

	job, err := server.jobQueue.GetJob(jobID)
	if err != nil || job.UserID != userID {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Job not found", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusOK, job)
}

// handleCancelJob requests cancellation of a running job
func (server *Server) handleCancelJob(responseWriter http.ResponseWriter, request *http.Request) {
	var cancelRequest struct {
		JobID string `json:"job_id"`
	}
	if err := json.NewDecoder(request.Body).Decode(&cancelRequest); err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid body", nil)
		return
	}

	if cancelRequest.JobID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "job_id is required", nil)
		return
	}

	userID := server.getUserID(request)

	job, err := server.jobQueue.GetJob(cancelRequest.JobID)
	if err != nil || job.UserID != userID {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Job not found", nil)
		return
	}

	if err := server.jobQueue.CancelJob(cancelRequest.JobID); err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "JOB_ERROR", "Failed to cancel job", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusOK, map[string]string{"message": "Job cancellation requested"})
}
