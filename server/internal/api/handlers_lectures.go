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
func (server *Server) handleCreateLecture(responseWriter http.ResponseWriter, request *http.Request) {
	pathVariables := mux.Vars(request)
	examIdentifier := pathVariables["examId"]

	var createLectureRequest struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(request.Body).Decode(&createLectureRequest); err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	if createLectureRequest.Title == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Title is required", nil)
		return
	}

	// Verify exam exists
	var examExists bool
	err := server.database.QueryRow("SELECT EXISTS(SELECT 1 FROM exams WHERE id = ?)", examIdentifier).Scan(&examExists)
	if err != nil || !examExists {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Exam not found", nil)
		return
	}

	lecture := models.Lecture{
		ID:          uuid.New().String(),
		ExamID:      examIdentifier,
		Title:       createLectureRequest.Title,
		Description: createLectureRequest.Description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err = server.database.Exec(`
		INSERT INTO lectures (id, exam_id, title, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, lecture.ID, lecture.ExamID, lecture.Title, lecture.Description, lecture.CreatedAt, lecture.UpdatedAt)

	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to create lecture", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusCreated, lecture)
}

// handleListLectures lists all lectures for an exam
func (server *Server) handleListLectures(responseWriter http.ResponseWriter, request *http.Request) {
	pathVariables := mux.Vars(request)
	examIdentifier := pathVariables["examId"]

	rows, err := server.database.Query(`
		SELECT id, exam_id, title, description, created_at, updated_at
		FROM lectures
		WHERE exam_id = ?
		ORDER BY created_at DESC
	`, examIdentifier)
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to list lectures", nil)
		return
	}
	defer rows.Close()

	lectures := []models.Lecture{}
	for rows.Next() {
		var lecture models.Lecture
		if err := rows.Scan(&lecture.ID, &lecture.ExamID, &lecture.Title, &lecture.Description, &lecture.CreatedAt, &lecture.UpdatedAt); err != nil {
			server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to scan lecture", nil)
			return
		}
		lectures = append(lectures, lecture)
	}

	server.writeJSON(responseWriter, http.StatusOK, lectures)
}

// handleGetLecture retrieves a specific lecture
func (server *Server) handleGetLecture(responseWriter http.ResponseWriter, request *http.Request) {
	pathVariables := mux.Vars(request)
	lectureIdentifier := pathVariables["lectureId"]

	var lecture models.Lecture
	err := server.database.QueryRow(`
		SELECT id, exam_id, title, description, created_at, updated_at
		FROM lectures
		WHERE id = ?
	`, lectureIdentifier).Scan(&lecture.ID, &lecture.ExamID, &lecture.Title, &lecture.Description, &lecture.CreatedAt, &lecture.UpdatedAt)

	if err == sql.ErrNoRows {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Lecture not found", nil)
		return
	}
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get lecture", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusOK, lecture)
}

// handleUpdateLecture updates a lecture
func (server *Server) handleUpdateLecture(responseWriter http.ResponseWriter, request *http.Request) {
	pathVariables := mux.Vars(request)
	lectureIdentifier := pathVariables["lectureId"]

	var updateLectureRequest struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
	}

	if err := json.NewDecoder(request.Body).Decode(&updateLectureRequest); err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	// Check if lecture exists
	var exists bool
	err := server.database.QueryRow("SELECT EXISTS(SELECT 1 FROM lectures WHERE id = ?)", lectureIdentifier).Scan(&exists)
	if err != nil || !exists {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Lecture not found", nil)
		return
	}

	// Build update query dynamically
	updates := []interface{}{}
	query := "UPDATE lectures SET updated_at = ?"
	updates = append(updates, time.Now())

	if updateLectureRequest.Title != nil {
		query += ", title = ?"
		updates = append(updates, *updateLectureRequest.Title)
	}
	if updateLectureRequest.Description != nil {
		query += ", description = ?"
		updates = append(updates, *updateLectureRequest.Description)
	}

	query += " WHERE id = ?"
	updates = append(updates, lectureIdentifier)

	_, err = server.database.Exec(query, updates...)
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to update lecture", nil)
		return
	}

	// Fetch updated lecture
	var lecture models.Lecture
	err = server.database.QueryRow(`
		SELECT id, exam_id, title, description, created_at, updated_at
		FROM lectures
		WHERE id = ?
	`, lectureIdentifier).Scan(&lecture.ID, &lecture.ExamID, &lecture.Title, &lecture.Description, &lecture.CreatedAt, &lecture.UpdatedAt)

	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to fetch updated lecture", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusOK, lecture)
}

// handleDeleteLecture deletes a lecture and all associated data
func (server *Server) handleDeleteLecture(responseWriter http.ResponseWriter, request *http.Request) {
	pathVariables := mux.Vars(request)
	lectureIdentifier := pathVariables["lectureId"]

	result, err := server.database.Exec("DELETE FROM lectures WHERE id = ?", lectureIdentifier)
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to delete lecture", nil)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Lecture not found", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusOK, map[string]string{"message": "Lecture deleted successfully"})
}
