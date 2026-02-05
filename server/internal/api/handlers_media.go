package api

import (
	"database/sql"
	"net/http"

	"lectures/internal/models"

	"github.com/gorilla/mux"
)

// handleListMedia lists all media files for a lecture
func (server *Server) handleListMedia(responseWriter http.ResponseWriter, request *http.Request) {
	pathVariables := mux.Vars(request)
	lectureIdentifier := pathVariables["lectureId"]

	mediaRows, databaseError := server.database.Query(`
		SELECT id, lecture_id, media_type, sequence_order, duration_milliseconds, file_path, created_at
		FROM lecture_media
		WHERE lecture_id = ?
		ORDER BY sequence_order ASC
	`, lectureIdentifier)
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
