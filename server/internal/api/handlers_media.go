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
func (s *Server) handleUploadMedia(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	lectureID := vars["lecture_id"]

	// Verify lecture exists
	var lectureExists bool
	err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM lectures WHERE id = ?)", lectureID).Scan(&lectureExists)
	if err != nil || !lectureExists {
		s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Lecture not found", nil)
		return
	}

	// Parse multipart form
	if err := r.ParseMultipartForm(100 << 20); err != nil { // 100 MB limit
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Failed to parse form", nil)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "No file provided", nil)
		return
	}
	defer file.Close()

	mediaType := r.FormValue("media_type")
	if mediaType != "audio" && mediaType != "video" {
		s.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "media_type must be 'audio' or 'video'", nil)
		return
	}

	// Get next sequence order
	var maxOrder sql.NullInt64
	err = s.db.QueryRow("SELECT MAX(sequence_order) FROM lecture_media WHERE lecture_id = ?", lectureID).Scan(&maxOrder)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get sequence order", nil)
		return
	}

	sequenceOrder := 0
	if maxOrder.Valid {
		sequenceOrder = int(maxOrder.Int64) + 1
	}

	// Save file
	mediaID := uuid.New().String()
	ext := filepath.Ext(header.Filename)
	filename := fmt.Sprintf("%s%s", mediaID, ext)

	lectureDir := filepath.Join(s.config.Storage.DataDirectory, "files", "lectures", lectureID, "media")
	if err := os.MkdirAll(lectureDir, 0755); err != nil {
		s.writeError(w, http.StatusInternalServerError, "FILE_ERROR", "Failed to create directory", nil)
		return
	}

	filePath := filepath.Join(lectureDir, filename)
	dst, err := os.Create(filePath)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "FILE_ERROR", "Failed to create file", nil)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		s.writeError(w, http.StatusInternalServerError, "FILE_ERROR", "Failed to save file", nil)
		return
	}

	// Create database record
	media := models.LectureMedia{
		ID:            mediaID,
		LectureID:     lectureID,
		MediaType:     mediaType,
		SequenceOrder: sequenceOrder,
		FilePath:      filePath,
		CreatedAt:     time.Now(),
	}

	_, err = s.db.Exec(`
		INSERT INTO lecture_media (id, lecture_id, media_type, sequence_order, file_path, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, media.ID, media.LectureID, media.MediaType, media.SequenceOrder, media.FilePath, media.CreatedAt)

	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to create media record", nil)
		return
	}

	s.writeJSON(w, http.StatusCreated, media)
}

// handleListMedia lists all media files for a lecture
func (s *Server) handleListMedia(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	lectureID := vars["lecture_id"]

	rows, err := s.db.Query(`
		SELECT id, lecture_id, media_type, sequence_order, duration_milliseconds, file_path, created_at
		FROM lecture_media
		WHERE lecture_id = ?
		ORDER BY sequence_order ASC
	`, lectureID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to list media", nil)
		return
	}
	defer rows.Close()

	mediaList := []models.LectureMedia{}
	for rows.Next() {
		var media models.LectureMedia
		var duration sql.NullInt64
		if err := rows.Scan(&media.ID, &media.LectureID, &media.MediaType, &media.SequenceOrder, &duration, &media.FilePath, &media.CreatedAt); err != nil {
			s.writeError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to scan media", nil)
			return
		}
		if duration.Valid {
			media.DurationMilliseconds = duration.Int64
		}
		mediaList = append(mediaList, media)
	}

	s.writeJSON(w, http.StatusOK, mediaList)
}

// handleDeleteMedia deletes a media file
func (s *Server) handleDeleteMedia(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	mediaID := vars["media_id"]

	// Get file path before deleting
	var filePath string
	err := s.db.QueryRow("SELECT file_path FROM lecture_media WHERE id = ?", mediaID).Scan(&filePath)
	if err == sql.ErrNoRows {
		s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Media not found", nil)
		return
	}
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get media", nil)
		return
	}

	// Delete from database
	result, err := s.db.Exec("DELETE FROM lecture_media WHERE id = ?", mediaID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to delete media", nil)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		s.writeError(w, http.StatusNotFound, "NOT_FOUND", "Media not found", nil)
		return
	}

	// Delete file from filesystem
	_ = os.Remove(filePath)

	s.writeJSON(w, http.StatusOK, map[string]string{"message": "Media deleted successfully"})
}
