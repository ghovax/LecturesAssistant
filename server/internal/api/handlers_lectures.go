package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lectures/internal/models"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// handleCreateLecture creates a new lecture with all its media and documents
func (server *Server) handleCreateLecture(responseWriter http.ResponseWriter, request *http.Request) {
	pathVariables := mux.Vars(request)
	examIdentifier := pathVariables["examId"]

	// Parse multipart form (max 5GB total for everything)
	if err := request.ParseMultipartForm(5120 << 20); err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Failed to parse form or files too large", nil)
		return
	}

	title := request.FormValue("title")
	description := request.FormValue("description")

	if title == "" {
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
		Title:       title,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Begin transaction to ensure atomic creation
	transaction, err := server.database.Begin()
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to start transaction", nil)
		return
	}
	defer transaction.Rollback()

	_, err = transaction.Exec(`
		INSERT INTO lectures (id, exam_id, title, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, lecture.ID, lecture.ExamID, lecture.Title, lecture.Description, lecture.CreatedAt, lecture.UpdatedAt)

	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to create lecture record", nil)
		return
	}

	// 1. Process Media Files
	mediaFiles := request.MultipartForm.File["media"]
	for mediaIndex, fileHeader := range mediaFiles {
		mediaIdentifier := uuid.New().String()
		fileExtension := filepath.Ext(fileHeader.Filename)
		filename := fmt.Sprintf("%s%s", mediaIdentifier, fileExtension)

		lectureMediaDirectory := filepath.Join(server.configuration.Storage.DataDirectory, "files", "lectures", lecture.ID, "media")
		if err := os.MkdirAll(lectureMediaDirectory, 0755); err != nil {
			server.writeError(responseWriter, http.StatusInternalServerError, "FILE_ERROR", "Failed to create media directory", nil)
			return
		}

		mediaFilePath := filepath.Join(lectureMediaDirectory, filename)

		// Determine media type based on extension
		mediaType := "audio"
		extensionLower := strings.ToLower(fileExtension)
		for _, videoExt := range server.configuration.Uploads.Media.SupportedFormats.Video {
			if "."+videoExt == extensionLower {
				mediaType = "video"
				break
			}
		}

		// Save the file
		sourceFile, err := fileHeader.Open()
		if err != nil {
			server.writeError(responseWriter, http.StatusInternalServerError, "FILE_ERROR", "Failed to open uploaded media file", nil)
			return
		}
		defer sourceFile.Close()

		destinationFile, err := os.Create(mediaFilePath)
		if err != nil {
			server.writeError(responseWriter, http.StatusInternalServerError, "FILE_ERROR", "Failed to create media file", nil)
			return
		}
		defer destinationFile.Close()

		if _, err := io.Copy(destinationFile, sourceFile); err != nil {
			server.writeError(responseWriter, http.StatusInternalServerError, "FILE_ERROR", "Failed to save media file", nil)
			return
		}

		_, err = transaction.Exec(`
			INSERT INTO lecture_media (id, lecture_id, media_type, sequence_order, file_path, created_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, mediaIdentifier, lecture.ID, mediaType, mediaIndex, mediaFilePath, time.Now())

		if err != nil {
			server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to create media record", nil)
			return
		}
	}

	// 2. Process Reference Documents
	documentFiles := request.MultipartForm.File["documents"]
	for _, fileHeader := range documentFiles {
		documentIdentifier := uuid.New().String()
		fileExtension := filepath.Ext(fileHeader.Filename)
		filename := fmt.Sprintf("%s%s", documentIdentifier, fileExtension)

		lectureDocumentsDirectory := filepath.Join(server.configuration.Storage.DataDirectory, "files", "lectures", lecture.ID, "documents")
		if err := os.MkdirAll(lectureDocumentsDirectory, 0755); err != nil {
			server.writeError(responseWriter, http.StatusInternalServerError, "FILE_ERROR", "Failed to create documents directory", nil)
			return
		}

		documentFilePath := filepath.Join(lectureDocumentsDirectory, filename)
		documentType := strings.TrimPrefix(strings.ToLower(fileExtension), ".")
		switch documentType {
		case "pdf":
			documentType = "pdf"
		case "pptx":
			documentType = "pptx"
		case "docx":
			documentType = "docx"
		default:
			documentType = "other"
		}

		// Save the file
		sourceFile, err := fileHeader.Open()
		if err != nil {
			server.writeError(responseWriter, http.StatusInternalServerError, "FILE_ERROR", "Failed to open uploaded document", nil)
			return
		}
		defer sourceFile.Close()

		destinationFile, err := os.Create(documentFilePath)
		if err != nil {
			server.writeError(responseWriter, http.StatusInternalServerError, "FILE_ERROR", "Failed to create document file", nil)
			return
		}
		defer destinationFile.Close()

		if _, err := io.Copy(destinationFile, sourceFile); err != nil {
			server.writeError(responseWriter, http.StatusInternalServerError, "FILE_ERROR", "Failed to save document file", nil)
			return
		}

		_, err = transaction.Exec(`
			INSERT INTO reference_documents (id, lecture_id, document_type, title, file_path, page_count, extraction_status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, documentIdentifier, lecture.ID, documentType, fileHeader.Filename, documentFilePath, 0, "pending", time.Now(), time.Now())

		if err != nil {
			server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to create document record", nil)
			return
		}
	}

	if err := transaction.Commit(); err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to commit transaction", nil)
		return
	}

	// Trigger processing jobs immediately
	if len(mediaFiles) > 0 {
		server.jobQueue.Enqueue(models.JobTypeTranscribeMedia, map[string]string{
			"lecture_id": lecture.ID,
		})
	}

	if len(documentFiles) > 0 {
		server.jobQueue.Enqueue(models.JobTypeIngestDocuments, map[string]string{
			"lecture_id":    lecture.ID,
			"language_code": server.configuration.LLM.Language,
		})
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
	updates := []any{}
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

	// Delete from database (cascades to lecture_media, transcripts, reference_documents)
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

	// Delete files from filesystem
	lectureDirectory := filepath.Join(server.configuration.Storage.DataDirectory, "files", "lectures", lectureIdentifier)
	_ = os.RemoveAll(lectureDirectory)

	server.writeJSON(responseWriter, http.StatusOK, map[string]string{"message": "Lecture deleted successfully"})
}
