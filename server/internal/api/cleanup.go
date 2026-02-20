package api

import (
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// StartStagingCleanupWorker runs a background task to clean up old temp directories
func (server *Server) StartStagingCleanupWorker() {
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		for range ticker.C {
			cleanupTempDir(filepath.Join(os.TempDir(), "lectures-uploads"), "upload")
			cleanupTempDir(filepath.Join(os.TempDir(), "lectures-jobs"), "job")
			cleanupTempDir(filepath.Join(os.TempDir(), "lectures-documents"), "document")
			cleanupTempDir(filepath.Join(os.TempDir(), "lectures-exports"), "export")
			cleanupTempFiles(filepath.Join(os.TempDir(), "lectures-media-cache"), "media-cache")
		}
	}()
	slog.Info("Staging cleanup worker started")
}

// cleanupTempDir removes subdirectories older than 24 hours from the given directory.
func cleanupTempDir(dir string, label string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Error("Failed to read temp directory for cleanup", "dir", dir, "error", err)
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

		if currentTime.Sub(info.ModTime()) > threshold {
			entryPath := filepath.Join(dir, entry.Name())
			if err := os.RemoveAll(entryPath); err == nil {
				deletedCount++
			} else {
				slog.Error("Failed to delete orphaned directory", "path", entryPath, "error", err)
			}
		}
	}

	if deletedCount > 0 {
		slog.Info("Temp directory cleanup completed", "type", label, "deleted_directories", deletedCount)
	}
}

// cleanupTempFiles removes files (not directories) older than 24 hours from the given directory.
func cleanupTempFiles(dir string, label string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Error("Failed to read temp directory for cleanup", "dir", dir, "error", err)
		}
		return
	}

	currentTime := time.Now()
	threshold := 24 * time.Hour
	deletedCount := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if currentTime.Sub(info.ModTime()) > threshold {
			filePath := filepath.Join(dir, entry.Name())
			if err := os.Remove(filePath); err == nil {
				deletedCount++
			}
		}
	}

	if deletedCount > 0 {
		slog.Info("Temp file cleanup completed", "type", label, "deleted_files", deletedCount)
	}
}
