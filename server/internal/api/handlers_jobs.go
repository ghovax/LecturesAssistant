package api

import (
	"net/http"

	"github.com/gorilla/mux"
)

// handleListJobs lists recent background jobs
func (server *Server) handleListJobs(responseWriter http.ResponseWriter, request *http.Request) {
	jobRows, databaseError := server.database.Query(`
		SELECT id, type, status, progress, progress_message_text, input_tokens, output_tokens, estimated_cost, created_at
		FROM jobs
		ORDER BY created_at DESC
		LIMIT 50
	`)
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
	pathVariables := mux.Vars(request)
	jobIdentifier := pathVariables["id"]

	job, err := server.jobQueue.GetJob(jobIdentifier)
	if err != nil {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Job not found", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusOK, job)
}

// handleCancelJob requests cancellation of a running job
func (server *Server) handleCancelJob(responseWriter http.ResponseWriter, request *http.Request) {
	pathVariables := mux.Vars(request)
	jobIdentifier := pathVariables["id"]

	if err := server.jobQueue.CancelJob(jobIdentifier); err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "JOB_ERROR", "Failed to cancel job", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusOK, map[string]string{"message": "Job cancellation requested"})
}
