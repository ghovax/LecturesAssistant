package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"lectures/internal/models"
)

// BCP-47 Regex (basic validation)
var bcp47Regex = regexp.MustCompile(`^[a-zA-Z]{2,3}(?:-[a-zA-Z]{4})?(?:-[a-zA-Z]{2}|-[0-9]{3})?$`)

// handleCreateTool triggers a tool generation job
func (server *Server) handleCreateTool(responseWriter http.ResponseWriter, request *http.Request) {
	var createToolRequest struct {
		ExamID                  string `json:"exam_id"`
		LectureID               string `json:"lecture_id"`
		Type                    string `json:"type"` // "guide", "flashcard", "quiz"
		Length                  string `json:"length"`
		LanguageCode            string `json:"language_code"`
		EnableDocumentsMatching *bool  `json:"enable_documents_matching"`
		AdherenceThreshold      int    `json:"adherence_threshold"`
		MaximumRetries          int    `json:"maximum_retries"`
		// Models
		ModelDocumentsMatching string `json:"model_documents_matching"`
		ModelStructure         string `json:"model_structure"`
		ModelGeneration        string `json:"model_generation"`
		ModelAdherence         string `json:"model_adherence"`
		ModelPolishing         string `json:"model_polishing"`
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

	enableMatching := server.configuration.LLM.EnableDocumentsMatching
	if createToolRequest.EnableDocumentsMatching != nil {
		enableMatching = *createToolRequest.EnableDocumentsMatching
	}

	// Validate BCP-47 language code
	if !bcp47Regex.MatchString(createToolRequest.LanguageCode) {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid language_code format (BCP-47 required)", nil)
		return
	}

	userID := server.getUserID(request)

	// Enqueue job
	jobIdentifier, err := server.jobQueue.Enqueue(userID, models.JobTypeBuildMaterial, map[string]string{
		"exam_id":                   createToolRequest.ExamID,
		"lecture_id":                createToolRequest.LectureID,
		"type":                      createToolRequest.Type,
		"length":                    createToolRequest.Length,
		"language_code":             createToolRequest.LanguageCode,
		"enable_documents_matching": fmt.Sprintf("%v", enableMatching),
		"adherence_threshold":       fmt.Sprintf("%d", createToolRequest.AdherenceThreshold),
		"maximum_retries":           fmt.Sprintf("%d", createToolRequest.MaximumRetries),
		"model_documents_matching":  createToolRequest.ModelDocumentsMatching,
		"model_structure":           createToolRequest.ModelStructure,
		"model_generation":          createToolRequest.ModelGeneration,
		"model_adherence":           createToolRequest.ModelAdherence,
		"model_polishing":           createToolRequest.ModelPolishing,
	}, createToolRequest.ExamID, createToolRequest.LectureID)

	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "BACKGROUND_JOB_ERROR", "Failed to create generation job", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusAccepted, map[string]string{
		"job_id":  jobIdentifier,
		"message": "Generation job created",
	})
}

// handleListTools lists all tools for an exam (must belong to the user)
func (server *Server) handleListTools(responseWriter http.ResponseWriter, request *http.Request) {
	examID := request.URL.Query().Get("exam_id")
	if examID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "exam_id is required", nil)
		return
	}

	userID := server.getUserID(request)
	toolType := request.URL.Query().Get("type")

	query := `
		SELECT tools.id, tools.exam_id, tools.type, tools.title, tools.language_code, tools.created_at, tools.updated_at
		FROM tools
		JOIN exams ON tools.exam_id = exams.id
		WHERE tools.exam_id = ? AND exams.user_id = ?
	`
	arguments := []any{examID, userID}

	if toolType != "" {
		query += " AND tools.type = ?"
		arguments = append(arguments, toolType)
	}

	query += " ORDER BY tools.created_at DESC"

	toolRows, databaseError := server.database.Query(query, arguments...)
	if databaseError != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to list tools", nil)
		return
	}
	defer toolRows.Close()

	var toolsList []models.Tool
	for toolRows.Next() {
		var tool models.Tool
		if err := toolRows.Scan(&tool.ID, &tool.ExamID, &tool.Type, &tool.Title, &tool.LanguageCode, &tool.CreatedAt, &tool.UpdatedAt); err != nil {
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

	userID := server.getUserID(request)

	var tool models.Tool
	err := server.database.QueryRow(`
		SELECT tools.id, tools.exam_id, tools.type, tools.title, tools.language_code, tools.content, tools.created_at, tools.updated_at
		FROM tools
		JOIN exams ON tools.exam_id = exams.id
		WHERE tools.id = ? AND tools.exam_id = ? AND exams.user_id = ?
	`, toolID, examID, userID).Scan(&tool.ID, &tool.ExamID, &tool.Type, &tool.Title, &tool.LanguageCode, &tool.Content, &tool.CreatedAt, &tool.UpdatedAt)

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

	userID := server.getUserID(request)

	result, err := server.database.Exec(`
		DELETE FROM tools
		WHERE id = ? AND exam_id = ? AND EXISTS (
			SELECT 1 FROM exams WHERE id = ? AND user_id = ?
		)
	`, deleteRequest.ToolID, deleteRequest.ExamID, deleteRequest.ExamID, userID)
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

// handleExportTool triggers an export job for a specific tool (PDF, Docx, MD)
func (server *Server) handleExportTool(responseWriter http.ResponseWriter, request *http.Request) {
	var exportRequest struct {
		ToolID        string `json:"tool_id"`
		ExamID        string `json:"exam_id"`
		Format        string `json:"format"` // "pdf", "docx", "md"
		IncludeImages *bool  `json:"include_images"`
	}

	if decodingError := json.NewDecoder(request.Body).Decode(&exportRequest); decodingError != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	if exportRequest.ToolID == "" || exportRequest.ExamID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "tool_id and exam_id are required", nil)
		return
	}

	if exportRequest.Format == "" {
		exportRequest.Format = "pdf"
	}

	includeImages := true
	if exportRequest.IncludeImages != nil {
		includeImages = *exportRequest.IncludeImages
	}

	userID := server.getUserID(request)

	// Verify tool exists and belongs to the user
	var toolID, languageCode string
	queryError := server.database.QueryRow(`
		SELECT tools.id, tools.language_code
		FROM tools
		JOIN exams ON tools.exam_id = exams.id
		WHERE tools.id = ? AND tools.exam_id = ? AND exams.user_id = ?
	`, exportRequest.ToolID, exportRequest.ExamID, userID).Scan(&toolID, &languageCode)

	if queryError == sql.ErrNoRows {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Tool not found in this exam", nil)
		return
	}
	if queryError != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to verify tool", nil)
		return
	}

	// Enqueue export job
	jobIdentifier, enqueuingError := server.jobQueue.Enqueue(userID, models.JobTypePublishMaterial, map[string]string{
		"tool_id":        exportRequest.ToolID,
		"language_code":  languageCode,
		"format":         exportRequest.Format,
		"include_images": fmt.Sprintf("%v", includeImages),
	}, exportRequest.ExamID, "")

	if enqueuingError != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "BACKGROUND_JOB_ERROR", "Failed to create export job", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusAccepted, map[string]string{
		"job_id":  jobIdentifier,
		"message": "Export job created",
	})
}

// handleDownloadExport serves the generated export file
func (server *Server) handleDownloadExport(responseWriter http.ResponseWriter, request *http.Request) {
	filePath := request.URL.Query().Get("path")
	if filePath == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "path is required", nil)
		return
	}

	// Basic security check: ensure path is within data directory
	absoluteDataDir, _ := filepath.Abs(server.configuration.Storage.DataDirectory)
	absoluteFilePath, _ := filepath.Abs(filePath)

	if !filepath.HasPrefix(absoluteFilePath, absoluteDataDir) {
		server.writeError(responseWriter, http.StatusForbidden, "ACCESS_DENIED", "Access to this file is forbidden", nil)
		return
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "File not found", nil)
		return
	}

	// Set content-disposition to force download with original filename
	fileName := filepath.Base(filePath)
	responseWriter.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))

	http.ServeFile(responseWriter, request, filePath)
}
