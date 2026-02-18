package api

import (
	"database/sql"
	"fmt"
	"io"
	"lectures/internal/jobs"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// handleBackupDatabase creates a consistent backup of the SQLite database and serves it for download
func (server *Server) handleBackupDatabase(responseWriter http.ResponseWriter, request *http.Request) {
	// 1. Verify admin role
	userID := server.getUserID(request)
	var role string
	err := server.database.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&role)
	if err != nil {
		server.writeError(responseWriter, http.StatusUnauthorized, "AUTHENTICATION_ERROR", "Failed to verify user role", nil)
		return
	}

	if role != "admin" {
		server.writeError(responseWriter, http.StatusForbidden, "FORBIDDEN", "Only administrators can perform database backups", nil)
		return
	}

	// 2. Ensure all current in-memory configurations are synced to the database
	server.syncConfigurationToDatabase()

	// 3. Create a temporary backup file using SQLite's VACUUM INTO for consistency
	backupFilename := fmt.Sprintf("Backup_%s.db", time.Now().Format("20060102_150405"))
	backupPath := filepath.Join(os.TempDir(), backupFilename)

	// Execute VACUUM INTO to create a consistent copy of the database while it's running
	// We use a dedicated connection or just the main one. modernc.org/sqlite supports this.
	_, err = server.database.Exec(fmt.Sprintf("VACUUM INTO '%s'", backupPath))
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "BACKUP_ERROR", "Failed to generate database backup", err.Error())
		return
	}
	defer os.Remove(backupPath) // Clean up after serving

	// 4. Serve the file
	responseWriter.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", backupFilename))
	responseWriter.Header().Set("Content-Type", "application/x-sqlite3")
	http.ServeFile(responseWriter, request, backupPath)
}

// handleRestoreDatabase allows uploading an existing database file to restore a workspace
func (server *Server) handleRestoreDatabase(responseWriter http.ResponseWriter, request *http.Request) {
	// 1. Authorization: Only allow if not initialized OR if user is admin
	var userCount int
	server.database.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	initialized := userCount > 0

	if initialized {
		userID := server.getUserID(request)
		if userID == "" {
			server.writeError(responseWriter, http.StatusUnauthorized, "AUTHENTICATION_ERROR", "Authentication required", nil)
			return
		}
		var role string
		err := server.database.QueryRow("SELECT role FROM users WHERE id = ?", userID).Scan(&role)
		if err != nil || role != "admin" {
			server.writeError(responseWriter, http.StatusForbidden, "FORBIDDEN", "Only administrators can restore a database", nil)
			return
		}
	}

	// 2. Parse file
	file, _, err := request.FormFile("database")
	if err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Database file is required", nil)
		return
	}
	defer file.Close()

	// 3. Save to temporary location
	tempPath := filepath.Join(os.TempDir(), "restored_database.db")
	tempFile, err := os.Create(tempPath)
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "RESTORE_ERROR", "Failed to create temporary file", nil)
		return
	}
	defer os.Remove(tempPath)

	if _, err := io.Copy(tempFile, file); err != nil {
		tempFile.Close()
		server.writeError(responseWriter, http.StatusInternalServerError, "RESTORE_ERROR", "Failed to save uploaded file", nil)
		return
	}
	tempFile.Close()

	// 4. Close current database and replace it
	// WARNING: This is a disruptive operation.
	server.database.Close()

	realPath := filepath.Join(server.configuration.Storage.DataDirectory, "database.db")
	if err := os.Rename(tempPath, realPath); err != nil {
		// Attempt fallback copy if rename fails (e.g. across filesystems)
		if err := copyFile(tempPath, realPath); err != nil {
			server.writeError(responseWriter, http.StatusInternalServerError, "RESTORE_ERROR", "Failed to replace database file", nil)
			// Try to reopen current DB to avoid leaving system in broken state
			server.database, _ = sql.Open("sqlite", realPath) // Re-init needed
			return
		}
	}

	// 5. Re-initialize database connection
	newDB, err := sql.Open("sqlite", realPath+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=synchronous(NORMAL)")
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "RESTORE_ERROR", "Failed to re-open database after restore", nil)
		return
	}
	server.database = newDB

	// 6. Restart job queue with new database connection
	server.jobQueue.Stop()
	server.jobQueue = jobs.NewQueue(newDB, 4)
	jobs.RegisterHandlers(server.jobQueue, newDB, server.configuration, nil, nil, server.toolGenerator, server.markdownConverter, nil, nil)
	server.jobQueue.Start()

	// 7. Reload settings from restored database
	server.loadSettingsFromDatabase()

	server.writeJSON(responseWriter, http.StatusOK, map[string]string{
		"message": "Workspace restored successfully. You can now log in.",
	})
}

func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}
