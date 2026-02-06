package api

import (
	"database/sql"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lectures/internal/models"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// handleCreateLecture creates a new lecture and binds staged uploads to it
func (server *Server) handleCreateLecture(responseWriter http.ResponseWriter, request *http.Request) {
	pathVariables := mux.Vars(request)
	examID := pathVariables["examId"]

	// Support upload progress tracking for direct multipart uploads
	uploadID := request.URL.Query().Get("upload_id")
	if uploadID != "" && request.ContentLength > 0 {
		request.Body = &ProgressReader{
			Reader:     request.Body,
			Total:      request.ContentLength,
			UploadID:   uploadID,
			Hub:        server.wsHub,
			LastUpdate: time.Now(),
		}
	}

	// Parse multipart form (up to 5GB) to support direct files + metadata + staged IDs
	if err := request.ParseMultipartForm(5120 << 20); err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Form too large", nil)
		return
	}

	title := request.FormValue("title")
	if title == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Title is required", nil)
		return
	}

	// Verify exam exists
	var examExists bool
	err := server.database.QueryRow("SELECT EXISTS(SELECT 1 FROM exams WHERE id = ?)", examID).Scan(&examExists)
	if err != nil || !examExists {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Exam not found", nil)
		return
	}

	// 1. Create the Lecture
	lectureID := uuid.New().String()
	lecture := models.Lecture{
		ID:          lectureID,
		ExamID:      examID,
		Title:       title,
		Description: request.FormValue("description"),
		Status:      "processing",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	transaction, err := server.database.Begin()
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Transaction failed", nil)
		return
	}
	defer transaction.Rollback()

	_, err = transaction.Exec(`
		INSERT INTO lectures (id, exam_id, title, description, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, lecture.ID, lecture.ExamID, lecture.Title, lecture.Description, lecture.Status, lecture.CreatedAt, lecture.UpdatedAt)

	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to save lecture", nil)
		return
	}

	// 2. Bind Staged Media
	for index, uploadID := range request.Form["media_upload_ids"] {
		if err := server.commitStagedUpload(transaction, lectureID, uploadID, "media", index); err != nil {
			server.writeError(responseWriter, http.StatusInternalServerError, "UPLOAD_ERROR", "Failed to bind media: "+uploadID, nil)
			return
		}
	}

	// 3. Bind Staged Documents
	for _, uploadID := range request.Form["document_upload_ids"] {
		if err := server.commitStagedUpload(transaction, lectureID, uploadID, "document", 0); err != nil {
			server.writeError(responseWriter, http.StatusInternalServerError, "UPLOAD_ERROR", "Failed to bind document: "+uploadID, nil)
			return
		}
	}

	// 4. Handle Direct Multipart Files (Implicitly stage then bind)
	for index, fileHeader := range request.MultipartForm.File["media"] {
		uploadID := server.stageMultipartFile(fileHeader)
		if err := server.commitStagedUpload(transaction, lectureID, uploadID, "media", len(request.Form["media_upload_ids"])+index); err != nil {
			server.writeError(responseWriter, http.StatusInternalServerError, "UPLOAD_ERROR", "Failed to process direct media", nil)
			return
		}
	}
	for _, fileHeader := range request.MultipartForm.File["documents"] {
		uploadID := server.stageMultipartFile(fileHeader)
		if err := server.commitStagedUpload(transaction, lectureID, uploadID, "document", 0); err != nil {
			server.writeError(responseWriter, http.StatusInternalServerError, "UPLOAD_ERROR", "Failed to process direct document", nil)
			return
		}
	}

	if err := transaction.Commit(); err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Commit failed", nil)
		return
	}

	// 5. Trigger Async Jobs
	server.jobQueue.Enqueue(models.JobTypeTranscribeMedia, map[string]string{"lecture_id": lectureID})
	server.jobQueue.Enqueue(models.JobTypeIngestDocuments, map[string]string{"lecture_id": lectureID, "language_code": server.configuration.LLM.Language})

	server.writeJSON(responseWriter, http.StatusCreated, lecture)
}

// handleUploadPrepare starts a robust staging session
func (server *Server) handleUploadPrepare(responseWriter http.ResponseWriter, request *http.Request) {
	var prepareRequest struct {
		Filename string `json:"filename"`
		FileSize int64  `json:"file_size_bytes"`
	}
	if err := json.NewDecoder(request.Body).Decode(&prepareRequest); err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid body", nil)
		return
	}

	uploadID := uuid.New().String()
	uploadDirectory := filepath.Join(server.configuration.Storage.DataDirectory, "tmp", "uploads", uploadID)
	os.MkdirAll(uploadDirectory, 0755)

	metadataFile, _ := os.Create(filepath.Join(uploadDirectory, "metadata.json"))
	json.NewEncoder(metadataFile).Encode(prepareRequest)
	metadataFile.Close()

	os.Create(filepath.Join(uploadDirectory, "upload.data"))

	server.writeJSON(responseWriter, http.StatusOK, map[string]any{
		"upload_id":        uploadID,
		"chunk_size_bytes": 10 * 1024 * 1024,
	})
}

// handleUploadAppend appends binary data with progress tracking
func (server *Server) handleUploadAppend(responseWriter http.ResponseWriter, request *http.Request) {
	uploadID := request.URL.Query().Get("upload_id")
	if uploadID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "upload_id query parameter is required", nil)
		return
	}

	uploadDirectory := filepath.Join(server.configuration.Storage.DataDirectory, "tmp", "uploads", uploadID)
	dataFilePath := filepath.Join(uploadDirectory, "upload.data")

	dataFile, err := os.OpenFile(dataFilePath, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Upload session not found", nil)
		return
	}
	defer dataFile.Close()

	progressReader := &ProgressReader{
		Reader:     request.Body,
		Total:      request.ContentLength,
		UploadID:   uploadID,
		Hub:        server.wsHub,
		LastUpdate: time.Now(),
	}

	io.Copy(dataFile, progressReader)
	server.writeJSON(responseWriter, http.StatusOK, map[string]string{"status": "data_appended"})
}

// handleUploadStage verifies the staged file via payload ID
func (server *Server) handleUploadStage(responseWriter http.ResponseWriter, request *http.Request) {
	var stageRequest struct {
		UploadID string `json:"upload_id"`
	}
	if err := json.NewDecoder(request.Body).Decode(&stageRequest); err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid body", nil)
		return
	}

	uploadDirectory := filepath.Join(server.configuration.Storage.DataDirectory, "tmp", "uploads", stageRequest.UploadID)

	if _, err := os.Stat(filepath.Join(uploadDirectory, "upload.data")); err != nil {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Staged file not found", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusOK, map[string]any{
		"upload_id": stageRequest.UploadID,
		"status":    "staged",
	})
}

// Internal Helpers (The "Staged Upload Interface")

func (server *Server) stageMultipartFile(fileHeader *multipart.FileHeader) string {
	uploadID := uuid.New().String()
	uploadDirectory := filepath.Join(server.configuration.Storage.DataDirectory, "tmp", "uploads", uploadID)
	os.MkdirAll(uploadDirectory, 0755)

	metadataFile, _ := os.Create(filepath.Join(uploadDirectory, "metadata.json"))
	json.NewEncoder(metadataFile).Encode(map[string]any{"filename": fileHeader.Filename, "file_size_bytes": fileHeader.Size})
	metadataFile.Close()

	sourceFile, _ := fileHeader.Open()
	defer sourceFile.Close()
	destinationFile, _ := os.Create(filepath.Join(uploadDirectory, "upload.data"))
	defer destinationFile.Close()
	io.Copy(destinationFile, sourceFile)

	return uploadID
}

func (server *Server) commitStagedUpload(transaction *sql.Tx, lectureID string, uploadID string, targetType string, sequenceOrder int) error {
	uploadDirectory := filepath.Join(server.configuration.Storage.DataDirectory, "tmp", "uploads", uploadID)

	metadataBytes, err := os.ReadFile(filepath.Join(uploadDirectory, "metadata.json"))
	if err != nil {
		return err
	}
	var metadata struct {
		Filename string `json:"filename"`
	}
	json.Unmarshal(metadataBytes, &metadata)

	// Move file to permanent storage
	destinationDirectory := filepath.Join(server.configuration.Storage.DataDirectory, "files", "lectures", lectureID, targetType+"s")
	os.MkdirAll(destinationDirectory, 0755)

	fileID := uuid.New().String()
	fileExtension := filepath.Ext(metadata.Filename)
	destinationPath := filepath.Join(destinationDirectory, fileID+fileExtension)

	if err := os.Rename(filepath.Join(uploadDirectory, "upload.data"), destinationPath); err != nil {
		// Fallback to copy if rename fails (e.g. cross-device)
		sourceFile, _ := os.Open(filepath.Join(uploadDirectory, "upload.data"))
		destinationFile, _ := os.Create(destinationPath)
		io.Copy(destinationFile, sourceFile)
		sourceFile.Close()
		destinationFile.Close()
	}

	// Insert Database Metadata
	if targetType == "media" {
		mediaType := "audio"
		for _, videoExtension := range server.configuration.Uploads.Media.SupportedFormats.Video {
			if "."+videoExtension == strings.ToLower(fileExtension) {
				mediaType = "video"
				break
			}
		}
		_, err = transaction.Exec(`
			INSERT INTO lecture_media (id, lecture_id, media_type, sequence_order, file_path, created_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, fileID, lectureID, mediaType, sequenceOrder, destinationPath, time.Now())
	} else {
		documentType := strings.TrimPrefix(strings.ToLower(fileExtension), ".")
		normalizedTitle := strings.ReplaceAll(metadata.Filename, " ", "_")
		_, err = transaction.Exec(`
			INSERT INTO reference_documents (id, lecture_id, document_type, title, file_path, page_count, extraction_status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, fileID, lectureID, documentType, normalizedTitle, destinationPath, 0, "pending", time.Now(), time.Now())
	}

	os.RemoveAll(uploadDirectory)
	return err
}

// handleListLectures lists all lectures for an exam
func (server *Server) handleListLectures(responseWriter http.ResponseWriter, request *http.Request) {
	pathVariables := mux.Vars(request)
	examIdentifier := pathVariables["examId"]

	lectureRows, databaseError := server.database.Query(`
		SELECT id, exam_id, title, description, created_at, updated_at
		FROM lectures
		WHERE exam_id = ?
		ORDER BY created_at DESC
	`, examIdentifier)
	if databaseError != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to list lectures", nil)
		return
	}
	defer lectureRows.Close()

	lectures := []models.Lecture{}
	for lectureRows.Next() {
		var lecture models.Lecture
		if err := lectureRows.Scan(&lecture.ID, &lecture.ExamID, &lecture.Title, &lecture.Description, &lecture.CreatedAt, &lecture.UpdatedAt); err != nil {
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

	// Check if lecture is currently processing
	var status string
	err := server.database.QueryRow("SELECT status FROM lectures WHERE id = ?", lectureIdentifier).Scan(&status)
	if err == nil && status == "processing" {
		server.writeError(responseWriter, http.StatusConflict, "LECTURE_BUSY", "Cannot delete lecture while it is being processed.", nil)
		return
	}

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

// handleGetTranscript retrieves the unified transcript for a lecture
func (server *Server) handleGetTranscript(responseWriter http.ResponseWriter, request *http.Request) {
	pathVariables := mux.Vars(request)
	lectureIdentifier := pathVariables["lectureId"]

	// Get transcript metadata
	var transcriptID, status string
	err := server.database.QueryRow(`
		SELECT id, status FROM transcripts WHERE lecture_id = ?
	`, lectureIdentifier).Scan(&transcriptID, &status)

	if err == sql.ErrNoRows {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Transcript not found", nil)
		return
	}
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get transcript", nil)
		return
	}

	// Get segments in order
	transcriptRows, databaseError := server.database.Query(`
		SELECT id, transcript_id, media_id, start_millisecond, end_millisecond, text, confidence, speaker
		FROM transcript_segments
		WHERE transcript_id = ?
		ORDER BY start_millisecond ASC
	`, transcriptID)
	if databaseError != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get segments", nil)
		return
	}
	defer transcriptRows.Close()

	var segments []map[string]any
	for transcriptRows.Next() {
		var id int
		var transcriptID, mediaID, text, speaker sql.NullString
		var startMs, endMs int64
		var confidence sql.NullFloat64

		if err := transcriptRows.Scan(&id, &transcriptID, &mediaID, &startMs, &endMs, &text, &confidence, &speaker); err != nil {
			continue
		}

		segment := map[string]any{
			"id":                id,
			"start_millisecond": startMs,
			"end_millisecond":   endMs,
			"text":              text.String,
		}
		if confidence.Valid {
			segment["confidence"] = confidence.Float64
		}
		if speaker.Valid {
			segment["speaker"] = speaker.String
		}
		segments = append(segments, segment)
	}

	server.writeJSON(responseWriter, http.StatusOK, map[string]any{
		"transcript_id": transcriptID,
		"status":        status,
		"segments":      segments,
	})
}
