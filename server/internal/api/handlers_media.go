package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"

	"lectures/internal/models"
)

// handleListMedia lists all media files for a lecture
func (server *Server) handleListMedia(responseWriter http.ResponseWriter, request *http.Request) {
	lectureID := request.URL.Query().Get("lecture_id")
	if lectureID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "lecture_id is required", nil)
		return
	}

	userID := server.getUserID(request)

	mediaRows, databaseError := server.database.Query(`
		SELECT lecture_media.id, lecture_media.lecture_id, lecture_media.media_type, lecture_media.sequence_order, lecture_media.duration_milliseconds, lecture_media.file_path, lecture_media.original_filename, lecture_media.created_at
		FROM lecture_media
		JOIN lectures ON lecture_media.lecture_id = lectures.id
		JOIN exams ON lectures.exam_id = exams.id
		WHERE lecture_media.lecture_id = ? AND exams.user_id = ?
		ORDER BY lecture_media.sequence_order ASC
	`, lectureID, userID)
	if databaseError != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to list media", nil)
		return
	}
	defer mediaRows.Close()

	mediaList := []models.LectureMedia{}
	for mediaRows.Next() {
		var media models.LectureMedia
		var duration sql.NullInt64
		var originalFilename sql.NullString
		if err := mediaRows.Scan(&media.ID, &media.LectureID, &media.MediaType, &media.SequenceOrder, &duration, &media.FilePath, &originalFilename, &media.CreatedAt); err != nil {
			server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to scan media", nil)
			return
		}
		if duration.Valid {
			media.DurationMilliseconds = duration.Int64
		}
		if originalFilename.Valid {
			media.OriginalFilename = originalFilename.String
		}
		mediaList = append(mediaList, media)
	}

	server.writeJSON(responseWriter, http.StatusOK, mediaList)
}

// handleDeleteMedia deletes a specific lecture media file
func (server *Server) handleDeleteMedia(responseWriter http.ResponseWriter, request *http.Request) {
	var deleteRequest struct {
		MediaID   string `json:"media_id"`
		LectureID string `json:"lecture_id"`
	}
	if err := json.NewDecoder(request.Body).Decode(&deleteRequest); err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	if deleteRequest.MediaID == "" || deleteRequest.LectureID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "media_id and lecture_id are required", nil)
		return
	}

	userID := server.getUserID(request)

	// Get file path and verify ownership
	var filePath string
	err := server.database.QueryRow(`
		SELECT lecture_media.file_path FROM lecture_media 
		JOIN lectures ON lecture_media.lecture_id = lectures.id
		JOIN exams ON lectures.exam_id = exams.id
		WHERE lecture_media.id = ? AND lecture_media.lecture_id = ? AND exams.user_id = ?
	`, deleteRequest.MediaID, deleteRequest.LectureID, userID).Scan(&filePath)

	if err == sql.ErrNoRows {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Media not found", nil)
		return
	}
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to verify media", nil)
		return
	}

	// Delete from database
	_, err = server.database.Exec("DELETE FROM lecture_media WHERE id = ?", deleteRequest.MediaID)
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to delete media from database", nil)
		return
	}

	// Delete file
	_ = os.Remove(filePath)

	server.writeJSON(responseWriter, http.StatusOK, map[string]string{"message": "Media deleted successfully"})
}
