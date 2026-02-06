package api

import (
	"database/sql"
	"net/http"

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
		SELECT lecture_media.id, lecture_media.lecture_id, lecture_media.media_type, lecture_media.sequence_order, lecture_media.duration_milliseconds, lecture_media.file_path, lecture_media.created_at
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
		if err := mediaRows.Scan(&media.ID, &media.LectureID, &media.MediaType, &media.SequenceOrder, &duration, &media.FilePath, &media.CreatedAt); err != nil {
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
