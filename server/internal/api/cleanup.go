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
		for range ticker.C {
			server.cleanupOrphanedUploads()
			server.cleanupOrphanedJobDirectories()
			server.cleanupOrphanedDocumentDirectories()
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
		slog.Info("Upload staging cleanup completed", "deleted_directories", deletedCount)
	}
}

func (server *Server) cleanupOrphanedJobDirectories() {
	jobsDir := filepath.Join(os.TempDir(), "lectures-jobs")

	entries, err := os.ReadDir(jobsDir)
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Error("Failed to read jobs directory for cleanup", "error", err)
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

		// Delete if older than threshold (likely from crashed jobs or server restarts)
		if currentTime.Sub(info.ModTime()) > threshold {
			jobPath := filepath.Join(jobsDir, entry.Name())
			if err := os.RemoveAll(jobPath); err == nil {
				deletedCount++
			} else {
				slog.Error("Failed to delete orphaned job directory", "path", jobPath, "error", err)
			}
		}
	}

	if deletedCount > 0 {
		slog.Info("Job directory cleanup completed", "deleted_directories", deletedCount)
	}
}

func (server *Server) cleanupOrphanedDocumentDirectories() {
	documentsDir := filepath.Join(os.TempDir(), "lectures-documents")

	entries, err := os.ReadDir(documentsDir)
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Error("Failed to read documents directory for cleanup", "error", err)
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

		// Delete if older than threshold (PNG images from document processing)
		if currentTime.Sub(info.ModTime()) > threshold {
			docPath := filepath.Join(documentsDir, entry.Name())
			if err := os.RemoveAll(docPath); err == nil {
				deletedCount++
			} else {
				slog.Error("Failed to delete orphaned document directory", "path", docPath, "error", err)
			}
		}
	}

	if deletedCount > 0 {
		slog.Info("Document directory cleanup completed", "deleted_directories", deletedCount)
	}
}
