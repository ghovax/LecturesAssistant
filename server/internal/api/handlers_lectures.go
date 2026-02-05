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

// handleCreateLecture creates a new lecture
func (s *Server) handleCreateLecture(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	examID := vars["exam_id"]

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

	// Verify exam exists
	var examExists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM exams WHERE id = ?)", examID).Scan(&examExists)
	if err != nil || !examExists {
		s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Exam not found", nil)
		return
	}

	lecture := models.Lecture{
		ID:          uuid.New().String(),
		ExamID:      examID,
		Title:       req.Title,
		Description: req.Description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err = s.db.Exec(`
		INSERT INTO lectures (id, exam_id, title, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, lecture.ID, lecture.ExamID, lecture.Title, lecture.Description, lecture.CreatedAt, lecture.UpdatedAt)

	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to create lecture", nil)
		return
	}

	s.writeJSON(w, http.StatusCreated, lecture)
}

// handleListLectures lists all lectures for an exam
func (s *Server) handleListLectures(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	examID := vars["exam_id"]

	rows, err := s.db.Query(`
		SELECT id, exam_id, title, description, created_at, updated_at
		FROM lectures
		WHERE exam_id = ?
		ORDER BY created_at DESC
	`, examID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to list lectures", nil)
		return
	}
	defer rows.Close()

	lectures := []models.Lecture{}
	for rows.Next() {
		var lecture models.Lecture
		if err := rows.Scan(&lecture.ID, &lecture.ExamID, &lecture.Title, &lecture.Description, &lecture.CreatedAt, &lecture.UpdatedAt); err != nil {
			s.writeError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to scan lecture", nil)
			return
		}
		lectures = append(lectures, lecture)
	}

	s.writeJSON(w, http.StatusOK, lectures)
}

// handleGetLecture retrieves a specific lecture
func (s *Server) handleGetLecture(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	lectureID := vars["lecture_id"]

	var lecture models.Lecture
	err := s.db.QueryRow(`
		SELECT id, exam_id, title, description, created_at, updated_at
		FROM lectures
		WHERE id = ?
	`, lectureID).Scan(&lecture.ID, &lecture.ExamID, &lecture.Title, &lecture.Description, &lecture.CreatedAt, &lecture.UpdatedAt)

	if err == sql.ErrNoRows {
		s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Lecture not found", nil)
		return
	}
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get lecture", nil)
		return
	}

	s.writeJSON(w, http.StatusOK, lecture)
}

// handleUpdateLecture updates a lecture
func (s *Server) handleUpdateLecture(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	lectureID := vars["lecture_id"]

	var req struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	// Check if lecture exists
	var exists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM lectures WHERE id = ?)", lectureID).Scan(&exists)
	if err != nil || !exists {
		s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Lecture not found", nil)
		return
	}

	// Build update query dynamically
	updates := []interface{}{}
	query := "UPDATE lectures SET updated_at = ?"
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
	updates = append(updates, lectureID)

	_, err = s.db.Exec(query, updates...)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to update lecture", nil)
		return
	}

	// Fetch updated lecture
	var lecture models.Lecture
	err = s.db.QueryRow(`
		SELECT id, exam_id, title, description, created_at, updated_at
		FROM lectures
		WHERE id = ?
	`, lectureID).Scan(&lecture.ID, &lecture.ExamID, &lecture.Title, &lecture.Description, &lecture.CreatedAt, &lecture.UpdatedAt)

	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to fetch updated lecture", nil)
		return
	}

	s.writeJSON(w, http.StatusOK, lecture)
}

// handleDeleteLecture deletes a lecture and all associated data
func (s *Server) handleDeleteLecture(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	lectureID := vars["lecture_id"]

	result, err := s.db.Exec("DELETE FROM lectures WHERE id = ?", lectureID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to delete lecture", nil)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Lecture not found", nil)
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{"message": "Lecture deleted successfully"})
}
