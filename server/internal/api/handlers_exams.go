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
func (s *Server) handleCreateExam(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	if req.Title == "" {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Title is required", nil)
		return
	}

	exam := models.Exam{
		ID:          uuid.New().String(),
		Title:       req.Title,
		Description: req.Description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err := s.db.Exec(`
		INSERT INTO exams (id, title, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, exam.ID, exam.Title, exam.Description, exam.CreatedAt, exam.UpdatedAt)

	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to create exam", nil)
		return
	}

	s.writeJSON(w, http.StatusCreated, exam)
}

// handleListExams lists all exams
func (s *Server) handleListExams(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(`
		SELECT id, title, description, created_at, updated_at
		FROM exams
		ORDER BY created_at DESC
	`)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to list exams", nil)
		return
	}
	defer rows.Close()

	exams := []models.Exam{}
	for rows.Next() {
		var exam models.Exam
		if err := rows.Scan(&exam.ID, &exam.Title, &exam.Description, &exam.CreatedAt, &exam.UpdatedAt); err != nil {
			s.writeError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to scan exam", nil)
			return
		}
		exams = append(exams, exam)
	}

	s.writeJSON(w, http.StatusOK, exams)
}

// handleGetExam retrieves a specific exam
func (s *Server) handleGetExam(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	examID := vars["id"]

	var exam models.Exam
	err := s.db.QueryRow(`
		SELECT id, title, description, created_at, updated_at
		FROM exams
		WHERE id = ?
	`, examID).Scan(&exam.ID, &exam.Title, &exam.Description, &exam.CreatedAt, &exam.UpdatedAt)

	if err == sql.ErrNoRows {
		s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Exam not found", nil)
		return
	}
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get exam", nil)
		return
	}

	s.writeJSON(w, http.StatusOK, exam)
}

// handleUpdateExam updates an exam
func (s *Server) handleUpdateExam(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	examID := vars["id"]

	var req struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	// Check if exam exists
	var exists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM exams WHERE id = ?)", examID).Scan(&exists)
	if err != nil || !exists {
		s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Exam not found", nil)
		return
	}

	// Build update query dynamically
	updates := []interface{}{}
	query := "UPDATE exams SET updated_at = ?"
	updates = append(updates, time.Now())

	if req.Title != nil {
		query += ", title = ?"
		updates = append(updates, *req.Title)
	}
	if req.Description != nil {
		query += ", description = ?"
		updates = append(updates, *req.Description)
	}

	query += " WHERE id = ?"
	updates = append(updates, examID)

	_, err = s.db.Exec(query, updates...)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to update exam", nil)
		return
	}

	// Fetch updated exam
	var exam models.Exam
	err = s.db.QueryRow(`
		SELECT id, title, description, created_at, updated_at
		FROM exams
		WHERE id = ?
	`, examID).Scan(&exam.ID, &exam.Title, &exam.Description, &exam.CreatedAt, &exam.UpdatedAt)

	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to fetch updated exam", nil)
		return
	}

	s.writeJSON(w, http.StatusOK, exam)
}

// handleDeleteExam deletes an exam and all associated data
func (s *Server) handleDeleteExam(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	examID := vars["id"]

	result, err := s.db.Exec("DELETE FROM exams WHERE id = ?", examID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to delete exam", nil)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Exam not found", nil)
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{"message": "Exam deleted successfully"})
}
