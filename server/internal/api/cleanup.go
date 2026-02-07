package api

import (
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// StartStagingCleanupWorker runs a background task to clean up old staging directories
func (server *Server) StartStagingCleanupWorker() {
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		for {
			select {
			case <-ticker.C:
				server.cleanupOrphanedUploads()
			}
		}
	}()
	slog.Info("Staging cleanup worker started")
}

func (server *Server) cleanupOrphanedUploads() {
	uploadsDir := filepath.Join(os.TempDir(), "lectures-uploads")

	entries, err := os.ReadDir(uploadsDir)
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Error("Failed to read uploads directory for cleanup", "error", err)
		}
		return
	}

	currentTime := time.Now()
	threshold := 24 * time.Hour
	deletedCount := 0

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Delete if older than threshold
		if currentTime.Sub(info.ModTime()) > threshold {
			uploadPath := filepath.Join(uploadsDir, entry.Name())
			if err := os.RemoveAll(uploadPath); err == nil {
				deletedCount++
			} else {
				slog.Error("Failed to delete orphaned upload directory", "path", uploadPath, "error", err)
			}
		}
	}

	if deletedCount > 0 {
		slog.Info("Staging cleanup completed", "deleted_directories", deletedCount)
	}
}
