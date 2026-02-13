package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

// handleListJobs lists recent background jobs for the current user
func (server *Server) handleListJobs(responseWriter http.ResponseWriter, request *http.Request) {
	userID := server.getUserID(request)
	courseIDParam := request.URL.Query().Get("course_id")
	lectureIDParam := request.URL.Query().Get("lecture_id")

	query := `
		SELECT id, type, status, progress, progress_message_text, payload, result, course_id, lecture_id, input_tokens, output_tokens, estimated_cost, created_at
		FROM jobs
		WHERE user_id = ?
	`
	args := []any{userID}

	if courseIDParam != "" {
		query += " AND course_id = ?"
		args = append(args, courseIDParam)
	}
	if lectureIDParam != "" {
		query += " AND lecture_id = ?"
		args = append(args, lectureIDParam)
	}

	query += " ORDER BY created_at DESC LIMIT 100"

	jobRows, databaseError := server.database.Query(query, args...)
	if databaseError != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to list jobs", nil)
		return
	}
	defer jobRows.Close()

	var jobsList = []map[string]any{}
	for jobRows.Next() {
		var id, jobType, status, progressMsg, payload, result string
		var courseID, lectureID sql.NullString
		var progress, inputTokens, outputTokens int
		var estimatedCost float64
		var createdAt string

		if err := jobRows.Scan(&id, &jobType, &status, &progress, &progressMsg, &payload, &result, &courseID, &lectureID, &inputTokens, &outputTokens, &estimatedCost, &createdAt); err != nil {
			continue
		}

		jobData := map[string]any{
			"id":                    id,
			"type":                  jobType,
			"status":                status,
			"progress":              progress,
			"progress_message_text": progressMsg,
			"payload":               payload,
			"result":                result,
			"input_tokens":          inputTokens,
			"output_tokens":         outputTokens,
			"estimated_cost":        estimatedCost,
			"created_at":            createdAt,
		}

		if courseID.Valid {
			jobData["course_id"] = courseID.String
		}
		if lectureID.Valid {
			jobData["lecture_id"] = lectureID.String
		}

		jobsList = append(jobsList, jobData)
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
		JobID  string `json:"job_id"`
		Delete bool   `json:"delete"`
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

	if cancelRequest.Delete {
		_, err := server.database.Exec("DELETE FROM jobs WHERE id = ?", cancelRequest.JobID)
		if err != nil {
			server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to delete job record", nil)
			return
		}
		server.writeJSON(responseWriter, http.StatusOK, map[string]string{"message": "Job record deleted"})
		return
	}

	// Try to cancel via queue (will stop active workers)
	err = server.jobQueue.CancelJob(cancelRequest.JobID)

	// Even if queue cancellation fails (job not active in a worker),
	// ensure it is marked as CANCELLED in database if it was PENDING
	if err != nil {
		_, dbErr := server.database.Exec(`
			UPDATE jobs SET status = ?, completed_at = ? 
			WHERE id = ? AND status IN (?, ?)
		`, "CANCELLED", time.Now(), cancelRequest.JobID, "PENDING", "RUNNING")

		if dbErr != nil {
			server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to cancel job in database", nil)
			return
		}
	}

	server.writeJSON(responseWriter, http.StatusOK, map[string]string{"message": "Job cancellation requested"})
}
