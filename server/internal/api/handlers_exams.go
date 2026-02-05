package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"lectures/internal/models"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
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

	exam := models.Exam{
		ID:          uuid.New().String(),
		Title:       createExamRequest.Title,
		Description: createExamRequest.Description,
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
	rows, err := server.database.Query(`
		SELECT id, title, description, created_at, updated_at
		FROM exams
		ORDER BY created_at DESC
	`)
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to list exams", nil)
		return
	}
	defer rows.Close()

	exams := []models.Exam{}
	for rows.Next() {
		var exam models.Exam
		if err := rows.Scan(&exam.ID, &exam.Title, &exam.Description, &exam.CreatedAt, &exam.UpdatedAt); err != nil {
			server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to scan exam", nil)
			return
		}
		exams = append(exams, exam)
	}

	server.writeJSON(responseWriter, http.StatusOK, exams)
}

// handleGetExam retrieves a specific exam
func (server *Server) handleGetExam(responseWriter http.ResponseWriter, request *http.Request) {
	pathVariables := mux.Vars(request)
	examIdentifier := pathVariables["id"]

	var exam models.Exam
	err := server.database.QueryRow(`
		SELECT id, title, description, created_at, updated_at
		FROM exams
		WHERE id = ?
	`, examIdentifier).Scan(&exam.ID, &exam.Title, &exam.Description, &exam.CreatedAt, &exam.UpdatedAt)

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
	pathVariables := mux.Vars(request)
	examIdentifier := pathVariables["id"]

	var updateExamRequest struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
	}

	if err := json.NewDecoder(request.Body).Decode(&updateExamRequest); err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	// Check if exam exists
	var exists bool
	err := server.database.QueryRow("SELECT EXISTS(SELECT 1 FROM exams WHERE id = ?)", examIdentifier).Scan(&exists)
	if err != nil || !exists {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Exam not found", nil)
		return
	}

	// Build update query dynamically
	updates := []interface{}{}
	query := "UPDATE exams SET updated_at = ?"
	updates = append(updates, time.Now())

	if updateExamRequest.Title != nil {
		query += ", title = ?"
		updates = append(updates, *updateExamRequest.Title)
	}
	if updateExamRequest.Description != nil {
		query += ", description = ?"
		updates = append(updates, *updateExamRequest.Description)
	}

	query += " WHERE id = ?"
	updates = append(updates, examIdentifier)

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
	`, examIdentifier).Scan(&exam.ID, &exam.Title, &exam.Description, &exam.CreatedAt, &exam.UpdatedAt)

	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to fetch updated exam", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusOK, exam)
}

// handleDeleteExam deletes an exam and all associated data
func (server *Server) handleDeleteExam(responseWriter http.ResponseWriter, request *http.Request) {
	pathVariables := mux.Vars(request)
	examIdentifier := pathVariables["id"]

	result, err := server.database.Exec("DELETE FROM exams WHERE id = ?", examIdentifier)
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to delete exam", nil)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Exam not found", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusOK, map[string]string{"message": "Exam deleted successfully"})
}
