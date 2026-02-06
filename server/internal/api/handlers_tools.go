package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"lectures/internal/models"
)

// handleCreateTool triggers a tool generation job
func (server *Server) handleCreateTool(responseWriter http.ResponseWriter, request *http.Request) {
	var createToolRequest struct {
		ExamID              string `json:"exam_id"`
		LectureID           string `json:"lecture_id"`
		Type                string `json:"type"` // "guide", "flashcard", "quiz"
		Length              string `json:"length"`
		LanguageCode        string `json:"language_code"`
		EnableTriangulation bool   `json:"enable_triangulation"`
		AdherenceThreshold  int    `json:"adherence_threshold"`
		MaximumRetries      int    `json:"maximum_retries"`
		// Models
		ModelTriangulation string `json:"model_triangulation"`
		ModelStructure     string `json:"model_structure"`
		ModelGeneration    string `json:"model_generation"`
		ModelAdherence     string `json:"model_adherence"`
		ModelPolishing     string `json:"model_polishing"`
	}

	if err := json.NewDecoder(request.Body).Decode(&createToolRequest); err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	if createToolRequest.ExamID == "" || createToolRequest.LectureID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "exam_id and lecture_id are required", nil)
		return
	}

	// Verify exam and lecture exist
	var lecture models.Lecture
	err := server.database.QueryRow("SELECT id, status FROM lectures WHERE id = ? AND exam_id = ?", createToolRequest.LectureID, createToolRequest.ExamID).Scan(&lecture.ID, &lecture.Status)
	if err != nil {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Lecture not found in this exam", nil)
		return
	}

	if lecture.Status != "ready" {
		server.writeError(responseWriter, http.StatusConflict, "LECTURE_NOT_READY", fmt.Sprintf("Lecture is currently in status: %s. Please wait for processing to complete.", lecture.Status), nil)
		return
	}

	// Default values
	if createToolRequest.Type == "" {
		createToolRequest.Type = "guide"
	}
	if createToolRequest.Length == "" {
		createToolRequest.Length = "medium"
	}
	if createToolRequest.LanguageCode == "" {
		createToolRequest.LanguageCode = server.configuration.LLM.Language
	}

	// Enqueue job
	jobIdentifier, err := server.jobQueue.Enqueue(models.JobTypeBuildMaterial, map[string]string{
		"exam_id":              createToolRequest.ExamID,
		"lecture_id":           createToolRequest.LectureID,
		"type":                 createToolRequest.Type,
		"length":               createToolRequest.Length,
		"language_code":        createToolRequest.LanguageCode,
		"enable_triangulation": fmt.Sprintf("%v", createToolRequest.EnableTriangulation),
		"adherence_threshold":  fmt.Sprintf("%d", createToolRequest.AdherenceThreshold),
		"maximum_retries":      fmt.Sprintf("%d", createToolRequest.MaximumRetries),
		"model_triangulation":  createToolRequest.ModelTriangulation,
		"model_structure":      createToolRequest.ModelStructure,
		"model_generation":     createToolRequest.ModelGeneration,
		"model_adherence":      createToolRequest.ModelAdherence,
		"model_polishing":      createToolRequest.ModelPolishing,
	})

	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "JOB_ERROR", "Failed to create generation job", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusAccepted, map[string]string{
		"job_id":  jobIdentifier,
		"message": "Generation job created",
	})
}

// handleListTools lists all tools for an exam
func (server *Server) handleListTools(responseWriter http.ResponseWriter, request *http.Request) {
	examID := request.URL.Query().Get("exam_id")
	if examID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "exam_id is required", nil)
		return
	}

	toolType := request.URL.Query().Get("type")

	query := `
		SELECT id, exam_id, type, title, created_at, updated_at
		FROM tools
		WHERE exam_id = ?
	`
	arguments := []any{examID}

	if toolType != "" {
		query += " AND type = ?"
		arguments = append(arguments, toolType)
	}

	query += " ORDER BY created_at DESC"

	toolRows, databaseError := server.database.Query(query, arguments...)
	if databaseError != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to list tools", nil)
		return
	}
	defer toolRows.Close()

	var toolsList []models.Tool
	for toolRows.Next() {
		var tool models.Tool
		if err := toolRows.Scan(&tool.ID, &tool.ExamID, &tool.Type, &tool.Title, &tool.CreatedAt, &tool.UpdatedAt); err != nil {
			continue
		}
		toolsList = append(toolsList, tool)
	}

	server.writeJSON(responseWriter, http.StatusOK, toolsList)
}

// handleGetTool retrieves a specific tool
func (server *Server) handleGetTool(responseWriter http.ResponseWriter, request *http.Request) {
	toolID := request.URL.Query().Get("tool_id")
	examID := request.URL.Query().Get("exam_id")

	if toolID == "" || examID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "tool_id and exam_id are required", nil)
		return
	}

	var tool models.Tool
	err := server.database.QueryRow(`
		SELECT id, exam_id, type, title, content, created_at, updated_at
		FROM tools
		WHERE id = ? AND exam_id = ?
	`, toolID, examID).Scan(&tool.ID, &tool.ExamID, &tool.Type, &tool.Title, &tool.Content, &tool.CreatedAt, &tool.UpdatedAt)

	if err == sql.ErrNoRows {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Tool not found in this exam", nil)
		return
	}
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get tool", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusOK, tool)
}

// handleDeleteTool deletes a specific tool
func (server *Server) handleDeleteTool(responseWriter http.ResponseWriter, request *http.Request) {
	var deleteRequest struct {
		ToolID string `json:"tool_id"`
		ExamID string `json:"exam_id"`
	}
	if err := json.NewDecoder(request.Body).Decode(&deleteRequest); err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid body", nil)
		return
	}

	if deleteRequest.ToolID == "" || deleteRequest.ExamID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "tool_id and exam_id are required", nil)
		return
	}

	result, err := server.database.Exec("DELETE FROM tools WHERE id = ? AND exam_id = ?", deleteRequest.ToolID, deleteRequest.ExamID)
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to delete tool", nil)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Tool not found in this exam", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusOK, map[string]string{"message": "Tool deleted successfully"})
}
