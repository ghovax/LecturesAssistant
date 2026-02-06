package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"lectures/internal/models"

	"github.com/google/uuid"
)

// handleCreateExam creates a new exam
func (server *Server) handleCreateExam(responseWriter http.ResponseWriter, request *http.Request) {
	var createExamRequest struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(request.Body).Decode(&createExamRequest); err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	if createExamRequest.Title == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Title is required", nil)
		return
	}

	// Clean title and description
	title, description, _, _ := server.toolGenerator.CorrectProjectTitleDescription(request.Context(), createExamRequest.Title, createExamRequest.Description, "")

	exam := models.Exam{
		ID:          uuid.New().String(),
		Title:       title,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err := server.database.Exec(`
		INSERT INTO exams (id, title, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, exam.ID, exam.Title, exam.Description, exam.CreatedAt, exam.UpdatedAt)

	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to create exam", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusCreated, exam)
}

// handleListExams lists all exams
func (server *Server) handleListExams(responseWriter http.ResponseWriter, request *http.Request) {
	examRows, databaseError := server.database.Query(`
		SELECT id, title, description, created_at, updated_at
		FROM exams
		ORDER BY created_at DESC
	`)
	if databaseError != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to list exams", nil)
		return
	}
	defer examRows.Close()

	exams := []models.Exam{}
	for examRows.Next() {
		var exam models.Exam
		if err := examRows.Scan(&exam.ID, &exam.Title, &exam.Description, &exam.CreatedAt, &exam.UpdatedAt); err != nil {
			server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to scan exam", nil)
			return
		}
		exams = append(exams, exam)
	}

	server.writeJSON(responseWriter, http.StatusOK, exams)
}

// handleGetExam retrieves a specific exam
func (server *Server) handleGetExam(responseWriter http.ResponseWriter, request *http.Request) {
	examID := request.URL.Query().Get("exam_id")
	if examID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "exam_id is required", nil)
		return
	}

	var exam models.Exam
	err := server.database.QueryRow(`
		SELECT id, title, description, created_at, updated_at
		FROM exams
		WHERE id = ?
	`, examID).Scan(&exam.ID, &exam.Title, &exam.Description, &exam.CreatedAt, &exam.UpdatedAt)

	if err == sql.ErrNoRows {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Exam not found", nil)
		return
	}
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get exam", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusOK, exam)
}

// handleUpdateExam updates an exam
func (server *Server) handleUpdateExam(responseWriter http.ResponseWriter, request *http.Request) {
	var updateExamRequest struct {
		ExamID      string  `json:"exam_id"`
		Title       *string `json:"title"`
		Description *string `json:"description"`
	}

	if err := json.NewDecoder(request.Body).Decode(&updateExamRequest); err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	if updateExamRequest.ExamID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "exam_id is required", nil)
		return
	}

	// Check if exam exists
	var exists bool
	err := server.database.QueryRow("SELECT EXISTS(SELECT 1 FROM exams WHERE id = ?)", updateExamRequest.ExamID).Scan(&exists)
	if err != nil || !exists {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Exam not found", nil)
		return
	}

	// Build update query dynamically
	updates := []any{}
	query := "UPDATE exams SET updated_at = ?"
	updates = append(updates, time.Now())

	if updateExamRequest.Title != nil || updateExamRequest.Description != nil {
		currentTitle := ""
		currentDescription := ""
		server.database.QueryRow("SELECT title, description FROM exams WHERE id = ?", updateExamRequest.ExamID).Scan(&currentTitle, &currentDescription)

		newTitle := currentTitle
		if updateExamRequest.Title != nil {
			newTitle = *updateExamRequest.Title
		}
		newDescription := currentDescription
		if updateExamRequest.Description != nil {
			newDescription = *updateExamRequest.Description
		}

		cleanedTitle, cleanedDescription, _, _ := server.toolGenerator.CorrectProjectTitleDescription(request.Context(), newTitle, newDescription, "")

		if updateExamRequest.Title != nil {
			query += ", title = ?"
			updates = append(updates, cleanedTitle)
		}
		if updateExamRequest.Description != nil {
			query += ", description = ?"
			updates = append(updates, cleanedDescription)
		}
	}

	query += " WHERE id = ?"
	updates = append(updates, updateExamRequest.ExamID)

	_, err = server.database.Exec(query, updates...)
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to update exam", nil)
		return
	}

	// Fetch updated exam
	var exam models.Exam
	err = server.database.QueryRow(`
		SELECT id, title, description, created_at, updated_at
		FROM exams
		WHERE id = ?
	`, updateExamRequest.ExamID).Scan(&exam.ID, &exam.Title, &exam.Description, &exam.CreatedAt, &exam.UpdatedAt)

	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to fetch updated exam", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusOK, exam)
}

// handleDeleteExam deletes an exam and all associated data
func (server *Server) handleDeleteExam(responseWriter http.ResponseWriter, request *http.Request) {
	var deleteRequest struct {
		ExamID string `json:"exam_id"`
	}
	if err := json.NewDecoder(request.Body).Decode(&deleteRequest); err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid body", nil)
		return
	}

	if deleteRequest.ExamID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "exam_id is required", nil)
		return
	}

	// 1. Get all lecture IDs for this exam to clean up files later
	lectureRows, queryError := server.database.Query("SELECT id FROM lectures WHERE exam_id = ?", deleteRequest.ExamID)
	var lectureIdentifiers []string
	if queryError == nil {
		for lectureRows.Next() {
			var lectureIdentifier string
			if err := lectureRows.Scan(&lectureIdentifier); err == nil {
				lectureIdentifiers = append(lectureIdentifiers, lectureIdentifier)
			}
		}
		lectureRows.Close()
	}

	// 2. Delete from database
	result, err := server.database.Exec("DELETE FROM exams WHERE id = ?", deleteRequest.ExamID)
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to delete exam", nil)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Exam not found", nil)
		return
	}

	// 3. Delete lecture files from filesystem
	for _, lectureIdentifier := range lectureIdentifiers {
		lectureDirectory := filepath.Join(server.configuration.Storage.DataDirectory, "files", "lectures", lectureIdentifier)
		_ = os.RemoveAll(lectureDirectory)
	}

	server.writeJSON(responseWriter, http.StatusOK, map[string]string{"message": "Exam deleted successfully"})
}
