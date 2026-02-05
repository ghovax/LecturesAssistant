package database

import (
	"database/sql"
	"log/slog"
	"time"
)

// CheckLectureReadiness checks if all processing for a lecture is complete and updates its status
func CheckLectureReadiness(database *sql.DB, lectureID string) {
	// 1. Check transcript status
	var transcriptStatus string
	err := database.QueryRow("SELECT status FROM transcripts WHERE lecture_id = ?", lectureID).Scan(&transcriptStatus)
	if err != nil && err != sql.ErrNoRows {
		return
	}

	// 2. Check all reference documents extraction status
	var pendingDocuments int
	database.QueryRow("SELECT COUNT(*) FROM reference_documents WHERE lecture_id = ? AND extraction_status != 'completed'", lectureID).Scan(&pendingDocuments)

	// A lecture is ready if the transcript is completed (if it exists)
	// AND all reference documents are completed
	isTranscriptReady := transcriptStatus == "completed" || transcriptStatus == ""
	isDocumentsReady := pendingDocuments == 0

	if isTranscriptReady && isDocumentsReady {
		_, _ = database.Exec("UPDATE lectures SET status = 'ready', updated_at = ? WHERE id = ?", time.Now(), lectureID)
		slog.Info("Lecture is now READY", "lectureID", lectureID)
	}
}
