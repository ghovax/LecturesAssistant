package api

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"lectures/internal/models"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// handleUploadMedia handles media file uploads
func (server *Server) handleUploadMedia(responseWriter http.ResponseWriter, request *http.Request) {
	pathVariables := mux.Vars(request)
	lectureIdentifier := pathVariables["lectureId"]

	// Verify lecture exists
	var lectureExists bool
	err := server.database.QueryRow("SELECT EXISTS(SELECT 1 FROM lectures WHERE id = ?)", lectureIdentifier).Scan(&lectureExists)
	if err != nil || !lectureExists {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Lecture not found", nil)
		return
	}

	// Parse multipart form
	if err := request.ParseMultipartForm(100 << 20); err != nil { // 100 MB limit
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Failed to parse form", nil)
		return
	}

	file, header, err := request.FormFile("file")
	if err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "No file provided", nil)
		return
	}
	defer file.Close()

	mediaType := request.FormValue("media_type")
	if mediaType != "audio" && mediaType != "video" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "media_type must be 'audio' or 'video'", nil)
		return
	}

	// Get next sequence order
	var maxOrder sql.NullInt64
	err = server.database.QueryRow("SELECT MAX(sequence_order) FROM lecture_media WHERE lecture_id = ?", lectureIdentifier).Scan(&maxOrder)
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get sequence order", nil)
		return
	}

	sequenceOrder := 0
	if maxOrder.Valid {
		sequenceOrder = int(maxOrder.Int64) + 1
	}

	// Save file
	mediaIdentifier := uuid.New().String()
	fileExtension := filepath.Ext(header.Filename)
	filename := fmt.Sprintf("%s%s", mediaIdentifier, fileExtension)

	lectureDirectory := filepath.Join(server.configuration.Storage.DataDirectory, "files", "lectures", lectureIdentifier, "media")
	if err := os.MkdirAll(lectureDirectory, 0755); err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "FILE_ERROR", "Failed to create directory", nil)
		return
	}

	filePath := filepath.Join(lectureDirectory, filename)
	destinationFile, err := os.Create(filePath)
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "FILE_ERROR", "Failed to create file", nil)
		return
	}
	defer destinationFile.Close()

	if _, err := io.Copy(destinationFile, file); err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "FILE_ERROR", "Failed to save file", nil)
		return
	}

	// Create database record
	media := models.LectureMedia{
		ID:            mediaIdentifier,
		LectureID:     lectureIdentifier,
		MediaType:     mediaType,
		SequenceOrder: sequenceOrder,
		FilePath:      filePath,
		CreatedAt:     time.Now(),
	}

	_, err = server.database.Exec(`
		INSERT INTO lecture_media (id, lecture_id, media_type, sequence_order, file_path, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, media.ID, media.LectureID, media.MediaType, media.SequenceOrder, media.FilePath, media.CreatedAt)

	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to create media record", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusCreated, media)
}

// handleListMedia lists all media files for a lecture
func (server *Server) handleListMedia(responseWriter http.ResponseWriter, request *http.Request) {
	pathVariables := mux.Vars(request)
	lectureIdentifier := pathVariables["lectureId"]

	rows, err := server.database.Query(`
		SELECT id, lecture_id, media_type, sequence_order, duration_milliseconds, file_path, created_at
		FROM lecture_media
		WHERE lecture_id = ?
		ORDER BY sequence_order ASC
	`, lectureIdentifier)
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to list media", nil)
		return
	}
	defer rows.Close()

	mediaList := []models.LectureMedia{}
	for rows.Next() {
		var media models.LectureMedia
		var duration sql.NullInt64
		if err := rows.Scan(&media.ID, &media.LectureID, &media.MediaType, &media.SequenceOrder, &duration, &media.FilePath, &media.CreatedAt); err != nil {
			server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to scan media", nil)
			return
		}
		if duration.Valid {
			media.DurationMilliseconds = duration.Int64
		}
		mediaList = append(mediaList, media)
	}

	server.writeJSON(responseWriter, http.StatusOK, mediaList)
}

// handleDeleteMedia deletes a media file
func (server *Server) handleDeleteMedia(responseWriter http.ResponseWriter, request *http.Request) {
	pathVariables := mux.Vars(request)
	mediaIdentifier := pathVariables["mediaId"]

	// Get file path before deleting
	var filePath string
	err := server.database.QueryRow("SELECT file_path FROM lecture_media WHERE id = ?", mediaIdentifier).Scan(&filePath)
	if err == sql.ErrNoRows {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Media not found", nil)
		return
	}
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get media", nil)
		return
	}

	// Delete from database
	result, err := server.database.Exec("DELETE FROM lecture_media WHERE id = ?", mediaIdentifier)
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to delete media", nil)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Media not found", nil)
		return
	}

	// Delete file from filesystem
	_ = os.Remove(filePath)

	server.writeJSON(responseWriter, http.StatusOK, map[string]string{"message": "Media deleted successfully"})
}
