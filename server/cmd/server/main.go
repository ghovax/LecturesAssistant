package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"lectures/internal/api"
	config "lectures/internal/configuration"
	"lectures/internal/database"
	"lectures/internal/jobs"
	"lectures/internal/models"
	"lectures/internal/transcription"
)

func main() {
	// Parse command-line flags
	configurationPath := flag.String("config", "", "Path to configuration file")
	flag.Parse()

	// Load configuration
	loadedConfiguration, err := config.Load(*configurationPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Ensure data directory exists
	if err := ensureDataDirectory(loadedConfiguration.Storage.DataDirectory); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Initialize JSON logging to a file
	logFilePath := filepath.Join(loadedConfiguration.Storage.DataDirectory, "server.log")
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()

	// MultiWriter to log to both file and stdout
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	logger := slog.New(slog.NewJSONHandler(multiWriter, nil))
	slog.SetDefault(logger)

	// Initialize database
	databasePath := filepath.Join(loadedConfiguration.Storage.DataDirectory, "database.db")
	initializedDatabase, err := database.Initialize(databasePath)
	if err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer initializedDatabase.Close()

	// Initialize transcription provider and service
	var transcriptionProvider transcription.Provider
	switch loadedConfiguration.Transcription.Provider {
	case "whisper-local":
		transcriptionProvider = transcription.NewWhisperProvider(
			loadedConfiguration.Transcription.Whisper.Model,
			loadedConfiguration.Transcription.Whisper.Device,
		)
	default:
		slog.Warn("Unknown transcription provider, falling back to whisper-local", "provider", loadedConfiguration.Transcription.Provider)
		transcriptionProvider = transcription.NewWhisperProvider("base", "auto")
	}

	transcriptionService := transcription.NewService(loadedConfiguration, transcriptionProvider)

	// Initialize job queue
	backgroundJobQueue := jobs.NewQueue(initializedDatabase, 4) // 4 concurrent workers

	// Register job handlers
	backgroundJobQueue.RegisterHandler(models.JobTypeTranscribeMedia, func(context context.Context, job *models.Job, updateFn func(int, string)) error {
		var payload struct {
			LectureID string `json:"lecture_id"`
		}
		if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil {
			return fmt.Errorf("failed to unmarshal job payload: %w", err)
		}

		// 1. Get lecture media files in order
		rows, err := initializedDatabase.Query(`
			SELECT id, lecture_id, media_type, sequence_order, file_path, created_at
			FROM lecture_media
			WHERE lecture_id = ?
			ORDER BY sequence_order ASC
		`, payload.LectureID)
		if err != nil {
			return fmt.Errorf("failed to query media files: %w", err)
		}
		defer rows.Close()

		var mediaFiles []models.LectureMedia
		for rows.Next() {
			var media models.LectureMedia
			if err := rows.Scan(&media.ID, &media.LectureID, &media.MediaType, &media.SequenceOrder, &media.FilePath, &media.CreatedAt); err != nil {
				return fmt.Errorf("failed to scan media file: %w", err)
			}
			mediaFiles = append(mediaFiles, media)
		}

		if len(mediaFiles) == 0 {
			return fmt.Errorf("no media files found for lecture: %s", payload.LectureID)
		}

		// 2. Create transcript record if not exists
		transcriptID := uuid.New().String()
		_, err = initializedDatabase.Exec(`
			INSERT OR IGNORE INTO transcripts (id, lecture_id, status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?)
		`, transcriptID, payload.LectureID, "processing", time.Now(), time.Now())
		if err != nil {
			return fmt.Errorf("failed to create transcript: %w", err)
		}

		// Get the actual transcript ID (in case it already existed)
		err = initializedDatabase.QueryRow("SELECT id FROM transcripts WHERE lecture_id = ?", payload.LectureID).Scan(&transcriptID)
		if err != nil {
			return fmt.Errorf("failed to get transcript ID: %w", err)
		}

		// Update transcript status to processing
		_, err = initializedDatabase.Exec("UPDATE transcripts SET status = ?, updated_at = ? WHERE id = ?", "processing", time.Now(), transcriptID)
		if err != nil {
			return fmt.Errorf("failed to update transcript status: %w", err)
		}

		// 3. Create temporary directory for transcription
		temporaryDirectory := filepath.Join(loadedConfiguration.Storage.DataDirectory, "tmp", job.ID)
		if err := os.MkdirAll(temporaryDirectory, 0755); err != nil {
			return fmt.Errorf("failed to create temporary directory: %w", err)
		}
		defer os.RemoveAll(temporaryDirectory)

		// 4. Run transcription
		segments, err := transcriptionService.TranscribeLecture(context, mediaFiles, temporaryDirectory, updateFn)
		if err != nil {
			initializedDatabase.Exec("UPDATE transcripts SET status = ?, updated_at = ? WHERE id = ?", "failed", time.Now(), transcriptID)
			return fmt.Errorf("transcription service failed: %w", err)
		}

		// 5. Store segments in database
		databasePointer, err := initializedDatabase.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer databasePointer.Rollback()

		// Delete existing segments if any
		_, err = databasePointer.Exec("DELETE FROM transcript_segments WHERE transcript_id = ?", transcriptID)
		if err != nil {
			return fmt.Errorf("failed to delete old segments: %w", err)
		}

		for _, segment := range segments {
			_, err = databasePointer.Exec(`
				INSERT INTO transcript_segments (transcript_id, media_id, start_millisecond, end_millisecond, original_start_milliseconds, original_end_milliseconds, text, confidence, speaker)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, transcriptID, segment.MediaID, segment.StartMillisecond, segment.EndMillisecond, segment.OriginalStartMilliseconds, segment.OriginalEndMilliseconds, segment.Text, segment.Confidence, segment.Speaker)
			if err != nil {
				return fmt.Errorf("failed to insert segment: %w", err)
			}
		}

		// 6. Finalize transcript
		_, err = databasePointer.Exec("UPDATE transcripts SET status = ?, updated_at = ? WHERE id = ?", "completed", time.Now(), transcriptID)
		if err != nil {
			return fmt.Errorf("failed to finalize transcript status: %w", err)
		}

		if err := databasePointer.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}

		updateFn(100, "Transcription completed successfully")
		return nil
	})

	backgroundJobQueue.Start()
	defer backgroundJobQueue.Stop()

	// Create API server
	apiServer := api.NewServer(loadedConfiguration, initializedDatabase, backgroundJobQueue)

	// Start HTTP server
	serverAddress := fmt.Sprintf("%s:%d", loadedConfiguration.Server.Host, loadedConfiguration.Server.Port)
	slog.Info("Server starting", "address", serverAddress)
	slog.Info("Data directory", "directory", loadedConfiguration.Storage.DataDirectory)

	if err := http.ListenAndServe(serverAddress, apiServer.Handler()); err != nil {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
}

func ensureDataDirectory(directoryPath string) error {
	// Expand home directory
	if len(directoryPath) > 0 && directoryPath[0] == '~' {
		homeDirectory, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		directoryPath = filepath.Join(homeDirectory, directoryPath[1:])
	}

	// Create necessary subdirectories
	targetDirectories := []string{
		directoryPath,
		filepath.Join(directoryPath, "files", "lectures"),
		filepath.Join(directoryPath, "files", "exports"),
		filepath.Join(directoryPath, "models"),
	}

	for _, directory := range targetDirectories {
		if err := os.MkdirAll(directory, 0755); err != nil {
			return err
		}
	}

	return nil
}
