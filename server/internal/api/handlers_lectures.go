package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lectures/internal/media"
	"lectures/internal/models"

	"github.com/google/uuid"
)

// handleCreateLecture creates a new lecture and binds staged uploads to it
func (server *Server) handleCreateLecture(responseWriter http.ResponseWriter, request *http.Request) {
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

	examID := request.FormValue("exam_id")
	if examID == "" {
		examID = request.URL.Query().Get("exam_id")
	}

	if examID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "exam_id is required", nil)
		return
	}

	title := request.FormValue("title")
	if title == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Title is required", nil)
		return
	}

	description := request.FormValue("description")
	specifiedDateStr := request.FormValue("specified_date")
	var specifiedDate *time.Time
	if specifiedDateStr != "" {
		if parsedDate, err := time.Parse(time.RFC3339, specifiedDateStr); err == nil {
			specifiedDate = &parsedDate
		} else if parsedDate, err := time.Parse("2006-01-02", specifiedDateStr); err == nil {
			specifiedDate = &parsedDate
		}
	}

	// Clean title and description
	cleanedTitle, cleanedDescription, metrics, _ := server.toolGenerator.CorrectProjectTitleDescription(request.Context(), title, description, "")
	slog.Info("Lecture title/description polished",
		"input_tokens", metrics.InputTokens,
		"output_tokens", metrics.OutputTokens,
		"estimated_cost_usd", metrics.EstimatedCost)

	userID := server.getUserID(request)

	// Verify exam exists and belongs to user
	var examExists bool
	err := server.database.QueryRow("SELECT EXISTS(SELECT 1 FROM exams WHERE id = ? AND user_id = ?)", examID, userID).Scan(&examExists)
	if err != nil || !examExists {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Exam not found", nil)
		return
	}

	// 1. Create the Lecture
	lectureID := uuid.New().String()
	lecture := models.Lecture{
		ID:            lectureID,
		ExamID:        examID,
		Title:         cleanedTitle,
		Description:   cleanedDescription,
		SpecifiedDate: specifiedDate,
		Status:        "processing",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	transaction, err := server.database.Begin()
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Transaction failed", nil)
		return
	}
	defer transaction.Rollback()

	_, err = transaction.Exec(`
		INSERT INTO lectures (id, exam_id, title, description, specified_date, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, lecture.ID, lecture.ExamID, lecture.Title, lecture.Description, lecture.SpecifiedDate, lecture.Status, lecture.CreatedAt, lecture.UpdatedAt)

	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to save lecture", nil)
		return
	}

	// 2. Bind Staged Media
	for uploadIndex, uploadID := range request.Form["media_upload_ids"] {
		if err := server.commitStagedUpload(transaction, lectureID, uploadID, "media", uploadIndex); err != nil {
			server.writeError(responseWriter, http.StatusInternalServerError, "FILE_UPLOAD_ERROR", "Failed to bind media: "+uploadID, nil)
			return
		}
	}

	// 3. Bind Staged Documents
	for _, uploadID := range request.Form["document_upload_ids"] {
		if err := server.commitStagedUpload(transaction, lectureID, uploadID, "document", 0); err != nil {
			server.writeError(responseWriter, http.StatusInternalServerError, "FILE_UPLOAD_ERROR", "Failed to bind document: "+uploadID, nil)
			return
		}
	}

	// 4. Handle Direct Multipart Files (Implicitly stage then bind)
	for uploadIndex, fileHeader := range request.MultipartForm.File["media"] {
		uploadID := server.stageMultipartFile(fileHeader)
		if err := server.commitStagedUpload(transaction, lectureID, uploadID, "media", len(request.Form["media_upload_ids"])+uploadIndex); err != nil {
			server.writeError(responseWriter, http.StatusInternalServerError, "FILE_UPLOAD_ERROR", "Failed to process direct media", nil)
			return
		}
	}
	for _, fileHeader := range request.MultipartForm.File["documents"] {
		uploadID := server.stageMultipartFile(fileHeader)
		if err := server.commitStagedUpload(transaction, lectureID, uploadID, "document", 0); err != nil {
			server.writeError(responseWriter, http.StatusInternalServerError, "FILE_UPLOAD_ERROR", "Failed to process direct document", nil)
			return
		}
	}

	if err := transaction.Commit(); err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Commit failed", nil)
		return
	}

	// 5. Trigger Async Jobs
	server.jobQueue.Enqueue(userID, models.JobTypeTranscribeMedia, map[string]string{"lecture_id": lectureID}, examID, lectureID)
	server.jobQueue.Enqueue(userID, models.JobTypeIngestDocuments, map[string]string{"lecture_id": lectureID, "language_code": server.configuration.LLM.Language}, examID, lectureID)

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
	uploadDirectory := filepath.Join(os.TempDir(), "lectures-uploads", uploadID)
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

	uploadDirectory := filepath.Join(os.TempDir(), "lectures-uploads", uploadID)

	// Read metadata to get total expected size for global progress tracking
	var metadata struct {
		FileSize int64 `json:"file_size_bytes"`
	}
	metaBytes, _ := os.ReadFile(filepath.Join(uploadDirectory, "metadata.json"))
	json.Unmarshal(metaBytes, &metadata)

	dataFilePath := filepath.Join(uploadDirectory, "upload.data")
	dataFile, err := os.OpenFile(dataFilePath, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Upload session not found", nil)
		return
	}
	defer dataFile.Close()

	// Determine current offset for progress reporting
	info, _ := dataFile.Stat()
	currentOffset := info.Size()

	// Verify we are not exceeding the declared file size
	if metadata.FileSize > 0 && currentOffset+request.ContentLength > metadata.FileSize {
		server.writeError(responseWriter, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", fmt.Sprintf("Appending this chunk would exceed the declared file size of %d bytes", metadata.FileSize), nil)
		return
	}

	progressReader := &ProgressReader{
		Reader:     request.Body,
		Total:      metadata.FileSize,
		BytesRead:  currentOffset, // Start from existing bytes
		UploadID:   uploadID,
		Hub:        server.wsHub,
		LastUpdate: time.Now(),
		LastRead:   currentOffset,
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

	uploadDirectory := filepath.Join(os.TempDir(), "lectures-uploads", stageRequest.UploadID)

	// Verify file exists
	info, err := os.Stat(filepath.Join(uploadDirectory, "upload.data"))
	if err != nil {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Staged file not found", nil)
		return
	}

	// Verify file size matches metadata
	var metadata struct {
		FileSize int64 `json:"file_size_bytes"`
	}
	metaBytes, _ := os.ReadFile(filepath.Join(uploadDirectory, "metadata.json"))
	json.Unmarshal(metaBytes, &metadata)

	if metadata.FileSize > 0 && info.Size() != metadata.FileSize {
		server.writeError(responseWriter, http.StatusUnprocessableEntity, "INVALID_SIZE", fmt.Sprintf("Final size %d does not match expected size %d", info.Size(), metadata.FileSize), nil)
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
	uploadDirectory := filepath.Join(os.TempDir(), "lectures-uploads", uploadID)
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
	uploadDirectory := filepath.Join(os.TempDir(), "lectures-uploads", uploadID)
	defer os.RemoveAll(uploadDirectory)

	metadataBytes, err := os.ReadFile(filepath.Join(uploadDirectory, "metadata.json"))
	if err != nil {
		return fmt.Errorf("failed to read metadata: %w", err)
	}
	var metadata struct {
		Filename string `json:"filename"`
	}
	json.Unmarshal(metadataBytes, &metadata)

	// Move file to permanent storage
	destinationDirectory := filepath.Join(server.configuration.Storage.DataDirectory, "files", "lectures", lectureID, targetType+"s")
	os.MkdirAll(destinationDirectory, 0755)

	fileID := uuid.New().String()
	rawExtension := filepath.Ext(metadata.Filename)

	// Sanitize extension: only allow lowercase alphanumeric extensions from our supported list
	cleanExtension := strings.ToLower(strings.TrimPrefix(rawExtension, "."))
	isSupported := false

	if targetType == "media" {
		for _, extension := range server.configuration.Uploads.Media.SupportedFormats.Video {
			if extension == cleanExtension {
				isSupported = true
				break
			}
		}
		for _, extension := range server.configuration.Uploads.Media.SupportedFormats.Audio {
			if extension == cleanExtension {
				isSupported = true
				break
			}
		}
	} else {
		for _, extension := range server.configuration.Uploads.Documents.SupportedFormats {
			if extension == cleanExtension {
				isSupported = true
				break
			}
		}
	}

	if !isSupported {
		return fmt.Errorf("unsupported or malicious file extension: %s", cleanExtension)
	}

	destinationPath := filepath.Join(destinationDirectory, fileID+"."+cleanExtension)

	stagedPath := filepath.Join(uploadDirectory, "upload.data")
	if err := os.Rename(stagedPath, destinationPath); err != nil {
		// Fallback to copy
		sourceFile, err := os.Open(stagedPath)
		if err != nil {
			return fmt.Errorf("failed to open source for copy: %w", err)
		}
		defer sourceFile.Close()

		destinationFile, err := os.Create(destinationPath)
		if err != nil {
			return fmt.Errorf("failed to create destination for copy: %w", err)
		}
		defer destinationFile.Close()

		if _, err := io.Copy(destinationFile, sourceFile); err != nil {
			return fmt.Errorf("failed to copy file: %w", err)
		}
	}

	// Insert Database Metadata
	if targetType == "media" {
		mediaType := "audio"
		for _, videoExtension := range server.configuration.Uploads.Media.SupportedFormats.Video {
			if videoExtension == cleanExtension {
				mediaType = "video"
				break
			}
		}

		// Extract duration using ffprobe
		durationMs := int64(0)
		if extractedDuration, err := media.GetDurationMilliseconds(destinationPath); err == nil {
			durationMs = extractedDuration
			slog.Info("Extracted media duration", "file_id", fileID, "duration_milliseconds", durationMs, "duration_seconds", durationMs/1000)
		} else {
			slog.Warn("Failed to extract media duration, setting to 0", "file_id", fileID, "error", err)
		}

		_, err = transaction.Exec(`
			INSERT INTO lecture_media (id, lecture_id, media_type, sequence_order, duration_milliseconds, file_path, original_filename, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, fileID, lectureID, mediaType, sequenceOrder, durationMs, destinationPath, metadata.Filename, time.Now())
	} else {
		documentType := cleanExtension
		// Keep spaces for readability, but replace dashes with underscores
		// to ensure the citation parser (which splits on dashes) works correctly.
		normalizedTitle := strings.ReplaceAll(metadata.Filename, "-", "_")

		// Remove characters that are dangerous in filenames.
		unsafeChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", "\x00", "\n", "\r", "\t"}
		for _, char := range unsafeChars {
			normalizedTitle = strings.ReplaceAll(normalizedTitle, char, "_")
		}
		normalizedTitle = strings.Trim(normalizedTitle, " .")

		_, err = transaction.Exec(`
			INSERT INTO reference_documents (id, lecture_id, document_type, title, file_path, original_filename, page_count, extraction_status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, fileID, lectureID, documentType, normalizedTitle, destinationPath, metadata.Filename, 0, "pending", time.Now(), time.Now())
	}

	if err != nil {
		return fmt.Errorf("failed to insert metadata: %w", err)
	}

	return nil
}

// handleListLectures lists all lectures for an exam (must belong to the user)
func (server *Server) handleListLectures(responseWriter http.ResponseWriter, request *http.Request) {
	examID := request.URL.Query().Get("exam_id")
	if examID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "exam_id is required", nil)
		return
	}

	userID := server.getUserID(request)

	lectureRows, databaseError := server.database.Query(`
		SELECT lectures.id, lectures.exam_id, lectures.title, lectures.description, lectures.created_at, lectures.updated_at
		FROM lectures
		JOIN exams ON lectures.exam_id = exams.id
		WHERE lectures.exam_id = ? AND exams.user_id = ?
		ORDER BY lectures.created_at DESC
	`, examID, userID)
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
	lectureID := request.URL.Query().Get("lecture_id")
	examID := request.URL.Query().Get("exam_id")

	if lectureID == "" || examID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "lecture_id and exam_id are required", nil)
		return
	}

	userID := server.getUserID(request)

	var lecture models.Lecture
	err := server.database.QueryRow(`
		SELECT lectures.id, lectures.exam_id, lectures.title, lectures.description, lectures.status, lectures.created_at, lectures.updated_at
		FROM lectures
		JOIN exams ON lectures.exam_id = exams.id
		WHERE lectures.id = ? AND lectures.exam_id = ? AND exams.user_id = ?
	`, lectureID, examID, userID).Scan(&lecture.ID, &lecture.ExamID, &lecture.Title, &lecture.Description, &lecture.Status, &lecture.CreatedAt, &lecture.UpdatedAt)

	if err == sql.ErrNoRows {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Lecture not found in this exam", nil)
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
	var updateRequest struct {
		LectureID     string  `json:"lecture_id"`
		ExamID        string  `json:"exam_id"`
		Title         *string `json:"title"`
		Description   *string `json:"description"`
		SpecifiedDate *string `json:"specified_date"`
	}

	if err := json.NewDecoder(request.Body).Decode(&updateRequest); err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	if updateRequest.LectureID == "" || updateRequest.ExamID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "lecture_id and exam_id are required", nil)
		return
	}

	userID := server.getUserID(request)

	// Check if lecture exists and belongs to exam and user
	var exists bool
	err := server.database.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM lectures 
			JOIN exams ON lectures.exam_id = exams.id
			WHERE lectures.id = ? AND lectures.exam_id = ? AND exams.user_id = ?
		)
	`, updateRequest.LectureID, updateRequest.ExamID, userID).Scan(&exists)
	if err != nil || !exists {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Lecture not found in this exam", nil)
		return
	}

	// Build update query dynamically
	updates := []any{}
	query := "UPDATE lectures SET updated_at = ?"
	updates = append(updates, time.Now())

	if updateRequest.Title != nil || updateRequest.Description != nil {
		currentTitle := ""
		currentDescription := ""
		server.database.QueryRow("SELECT title, description FROM lectures WHERE id = ?", updateRequest.LectureID).Scan(&currentTitle, &currentDescription)

		newTitle := currentTitle
		if updateRequest.Title != nil {
			newTitle = *updateRequest.Title
		}
		newDescription := currentDescription
		if updateRequest.Description != nil {
			newDescription = *updateRequest.Description
		}

		cleanedTitle, cleanedDescription, metrics, _ := server.toolGenerator.CorrectProjectTitleDescription(request.Context(), newTitle, newDescription, "")
		slog.Info("Lecture title/description updated and polished",
			"lectureID", updateRequest.LectureID,
			"input_tokens", metrics.InputTokens,
			"output_tokens", metrics.OutputTokens,
			"estimated_cost_usd", metrics.EstimatedCost)

		if updateRequest.Title != nil {
			query += ", title = ?"
			updates = append(updates, cleanedTitle)
		}
		if updateRequest.Description != nil {
			query += ", description = ?"
			updates = append(updates, cleanedDescription)
		}
	}

	if updateRequest.SpecifiedDate != nil {
		var specifiedDate *time.Time
		if *updateRequest.SpecifiedDate != "" {
			if parsedDate, err := time.Parse(time.RFC3339, *updateRequest.SpecifiedDate); err == nil {
				specifiedDate = &parsedDate
			} else if parsedDate, err := time.Parse("2006-01-02", *updateRequest.SpecifiedDate); err == nil {
				specifiedDate = &parsedDate
			}
		}
		query += ", specified_date = ?"
		updates = append(updates, specifiedDate)
	}

	query += " WHERE id = ? AND exam_id = ?"
	updates = append(updates, updateRequest.LectureID, updateRequest.ExamID)

	_, err = server.database.Exec(query, updates...)
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to update lecture", nil)
		return
	}

	// Fetch updated lecture
	var lecture models.Lecture
	err = server.database.QueryRow(`
		SELECT id, exam_id, title, description, status, created_at, updated_at
		FROM lectures
		WHERE id = ?
	`, updateRequest.LectureID).Scan(&lecture.ID, &lecture.ExamID, &lecture.Title, &lecture.Description, &lecture.Status, &lecture.CreatedAt, &lecture.UpdatedAt)

	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to fetch updated lecture", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusOK, lecture)
}

// handleDeleteLecture deletes a lecture and all associated data
func (server *Server) handleDeleteLecture(responseWriter http.ResponseWriter, request *http.Request) {
	var deleteRequest struct {
		LectureID string `json:"lecture_id"`
		ExamID    string `json:"exam_id"`
	}
	if err := json.NewDecoder(request.Body).Decode(&deleteRequest); err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid body", nil)
		return
	}

	if deleteRequest.LectureID == "" || deleteRequest.ExamID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "lecture_id and exam_id are required", nil)
		return
	}

	userID := server.getUserID(request)

	// Check if lecture is currently processing or belongs to another exam/user
	var status string
	var currentExamID string
	err := server.database.QueryRow(`
		SELECT status, exam_id FROM lectures 
		JOIN exams ON lectures.exam_id = exams.id
		WHERE lectures.id = ? AND exams.user_id = ?
	`, deleteRequest.LectureID, userID).Scan(&status, &currentExamID)

	if err == sql.ErrNoRows || currentExamID != deleteRequest.ExamID {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Lecture not found in this exam", nil)
		return
	}

	// 1. Find and cancel any active jobs for this lecture
	jobRows, err := server.database.Query(`
		SELECT id FROM jobs 
		WHERE (status = 'PENDING' OR status = 'RUNNING') 
		AND payload LIKE '%' || ? || '%'
	`, deleteRequest.LectureID)
	if err == nil {
		for jobRows.Next() {
			var jobID string
			if scanErr := jobRows.Scan(&jobID); scanErr == nil {
				server.jobQueue.CancelJob(jobID)
			}
		}
		jobRows.Close()
	}

	// 2. Delete from database (cascades to lecture_media, transcripts, reference_documents)
	result, err := server.database.Exec("DELETE FROM lectures WHERE id = ? AND exam_id = ?", deleteRequest.LectureID, deleteRequest.ExamID)
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
	lectureDirectory := filepath.Join(server.configuration.Storage.DataDirectory, "files", "lectures", deleteRequest.LectureID)
	_ = os.RemoveAll(lectureDirectory)

	server.writeJSON(responseWriter, http.StatusOK, map[string]string{"message": "Lecture deleted successfully"})
}

// handleGetTranscript retrieves the unified transcript for a lecture
func (server *Server) handleGetTranscript(responseWriter http.ResponseWriter, request *http.Request) {
	lectureID := request.URL.Query().Get("lecture_id")
	if lectureID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "lecture_id is required", nil)
		return
	}

	userID := server.getUserID(request)

	// Get transcript metadata and verify ownership
	var transcriptID, status string
	err := server.database.QueryRow(`
		SELECT transcripts.id, transcripts.status 
		FROM transcripts 
		JOIN lectures ON transcripts.lecture_id = lectures.id
		JOIN exams ON lectures.exam_id = exams.id
		WHERE transcripts.lecture_id = ? AND exams.user_id = ?
	`, lectureID, userID).Scan(&transcriptID, &status)

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
		var segmentInternalID int
		var segmentID, mediaID, text, speaker sql.NullString
		var startMs, endMs int64
		var confidence sql.NullFloat64

		if err := transcriptRows.Scan(&segmentInternalID, &segmentID, &mediaID, &startMs, &endMs, &text, &confidence, &speaker); err != nil {
			continue
		}

		segment := map[string]any{
			"id":                segmentInternalID,
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

// handleGetTranscriptHTML retrieves the unified transcript for a lecture converted to HTML with timestamps
func (server *Server) handleGetTranscriptHTML(responseWriter http.ResponseWriter, request *http.Request) {
	lectureID := request.URL.Query().Get("lecture_id")
	if lectureID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "lecture_id is required", nil)
		return
	}

	userID := server.getUserID(request)

	// Verify lecture ownership and get transcript metadata
	var transcriptID, status string
	err := server.database.QueryRow(`
		SELECT transcripts.id, transcripts.status 
		FROM transcripts 
		JOIN lectures ON transcripts.lecture_id = lectures.id
		JOIN exams ON lectures.exam_id = exams.id
		WHERE transcripts.lecture_id = ? AND exams.user_id = ?
	`, lectureID, userID).Scan(&transcriptID, &status)

	if err == sql.ErrNoRows {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Transcript not found", nil)
		return
	}
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to verify transcript", nil)
		return
	}

	// Get transcript segments
	transcriptRows, databaseError := server.database.Query(`
		SELECT id, start_millisecond, end_millisecond, text, confidence, speaker
		FROM transcript_segments
		WHERE transcript_id = ?
		ORDER BY start_millisecond ASC
	`, transcriptID)
	if databaseError != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get segments", nil)
		return
	}
	defer transcriptRows.Close()

	type segmentData struct {
		ID               int     `json:"id"`
		StartMillisecond int64   `json:"start_millisecond"`
		EndMillisecond   int64   `json:"end_millisecond"`
		Text             string  `json:"-"`
		TextHTML         string  `json:"text_html"`
		Confidence       float64 `json:"confidence,omitempty"`
		Speaker          string  `json:"speaker,omitempty"`
	}

	var segments []segmentData
	var markdownBuilder strings.Builder
	separator := "---SEGMENT-BREAK---"

	for transcriptRows.Next() {
		var s segmentData
		var text, speaker sql.NullString
		var confidence sql.NullFloat64

		if err := transcriptRows.Scan(&s.ID, &s.StartMillisecond, &s.EndMillisecond, &text, &confidence, &speaker); err != nil {
			continue
		}

		s.Text = text.String
		if confidence.Valid {
			s.Confidence = confidence.Float64
		}
		if speaker.Valid {
			s.Speaker = speaker.String
		}

		segments = append(segments, s)

		// Add to markdown for batch conversion
		if markdownBuilder.Len() > 0 {
			markdownBuilder.WriteString("\n\n" + separator + "\n\n")
		}
		markdownBuilder.WriteString(s.Text)
	}

	if len(segments) > 0 {
		// Batch convert markdown to HTML
		fullHTML, err := server.markdownConverter.MarkdownToHTML(markdownBuilder.String())
		if err != nil {
			server.writeError(responseWriter, http.StatusInternalServerError, "CONVERSION_ERROR", "Failed to convert transcript to HTML", nil)
			return
		}

		// Split back into segments
		htmlParts := strings.Split(fullHTML, separator)

		// Clean up HTML tags around parts
		for i, part := range htmlParts {
			cleanedPart := strings.TrimSpace(part)
			cleanedPart = strings.TrimPrefix(cleanedPart, "</p>")
			cleanedPart = strings.TrimSuffix(cleanedPart, "<p>")
			cleanedPart = strings.TrimSpace(cleanedPart)

			if i < len(segments) {
				segments[i].TextHTML = cleanedPart
			}
		}
	}

	server.writeJSON(responseWriter, http.StatusOK, map[string]any{
		"transcript_id": transcriptID,
		"status":        status,
		"segments":      segments,
	})
}
