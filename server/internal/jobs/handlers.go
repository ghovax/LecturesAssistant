package jobs

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"lectures/internal/configuration"
	"lectures/internal/documents"
	"lectures/internal/markdown"
	"lectures/internal/models"
	"lectures/internal/tools"
	"lectures/internal/transcription"

	"github.com/google/uuid"
	"github.com/skip2/go-qrcode"
	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// sanitizeFilename replaces unsafe characters with underscores while keeping spaces
func sanitizeFilename(name string) string {
	// Characters that are unsafe in filenames across different filesystems
	unsafeChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", "#", "\x00", "\n", "\r", "\t"}

	result := name
	for _, char := range unsafeChars {
		result = strings.ReplaceAll(result, char, "_")
	}

	// Trim leading/trailing spaces and dots (problematic on some systems)
	result = strings.Trim(result, " .")

	// If the result is empty or only underscores, use a default
	if result == "" || strings.Trim(result, "_") == "" {
		result = "document"
	}

	return result
}

// RegisterHandlers registers all standard job handlers
func RegisterHandlers(
	queue *Queue,
	database *sql.DB,
	config *configuration.Configuration,
	transcriptionService *transcription.Service,
	documentProcessor *documents.Processor,
	toolGenerator *tools.ToolGenerator,
	markdownConverter markdown.MarkdownConverter,
	checkReadiness func(*sql.DB, string),
	broadcast func(string, string, any),
) {
	queue.RegisterHandler(models.JobTypeTranscribeMedia, func(jobContext context.Context, job *models.Job, updateProgress func(int, string, any, models.JobMetrics)) error {
		var payload struct {
			LectureID string `json:"lecture_id"`
		}
		if unmarshalingError := json.Unmarshal([]byte(job.Payload), &payload); unmarshalingError != nil {
			return fmt.Errorf("failed to unmarshal job payload: %w", unmarshalingError)
		}

		if broadcast != nil {
			broadcast("lecture:"+payload.LectureID, "lecture:processing", map[string]string{"lecture_id": payload.LectureID})
		}

		// 1. Get lecture media files in order
		mediaRows, databaseError := database.Query(`
			SELECT id, lecture_id, media_type, sequence_order, file_path, created_at
			FROM lecture_media
			WHERE lecture_id = ?
			ORDER BY sequence_order ASC
		`, payload.LectureID)
		if databaseError != nil {
			return fmt.Errorf("failed to query media files: %w", databaseError)
		}

		var mediaFiles []models.LectureMedia
		for mediaRows.Next() {
			var media models.LectureMedia
			if scanningError := mediaRows.Scan(&media.ID, &media.LectureID, &media.MediaType, &media.SequenceOrder, &media.FilePath, &media.CreatedAt); scanningError != nil {
				mediaRows.Close()
				return fmt.Errorf("failed to scan media file: %w", scanningError)
			}
			mediaFiles = append(mediaFiles, media)
		}
		mediaRows.Close()

		if len(mediaFiles) == 0 {
			return fmt.Errorf("no media files found for lecture: %s", payload.LectureID)
		}

		// 2. Create transcript record if not exists
		transcriptID := uuid.New().String()
		_, executionError := database.Exec(`
			INSERT OR IGNORE INTO transcripts (id, lecture_id, status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?)
		`, transcriptID, payload.LectureID, "processing", time.Now(), time.Now())
		if executionError != nil {
			return fmt.Errorf("failed to create transcript: %w", executionError)
		}

		// Get the actual transcript ID (in case it already existed)
		executionError = database.QueryRow("SELECT id FROM transcripts WHERE lecture_id = ?", payload.LectureID).Scan(&transcriptID)
		if executionError != nil {
			return fmt.Errorf("failed to get transcript ID: %w", executionError)
		}

		// Update transcript status to processing
		_, executionError = database.Exec("UPDATE transcripts SET status = ?, updated_at = ? WHERE id = ?", "processing", time.Now(), transcriptID)
		if executionError != nil {
			return fmt.Errorf("failed to update transcript status: %w", executionError)
		}

		// 3. Create temporary directory for transcription
		temporaryDirectory := filepath.Join(os.TempDir(), "lectures-jobs", job.ID)
		if mkdirError := os.MkdirAll(temporaryDirectory, 0755); mkdirError != nil {
			return fmt.Errorf("failed to create temporary directory: %w", mkdirError)
		}
		defer os.RemoveAll(temporaryDirectory)

		// 4. Run transcription
		segments, totalMetrics, transcriptionError := transcriptionService.TranscribeLecture(jobContext, mediaFiles, temporaryDirectory, func(progress int, message string, metadata any) {
			updateProgress(progress, "Transcribing media files...", metadata, models.JobMetrics{})
		})
		if transcriptionError != nil {
			database.Exec("UPDATE transcripts SET status = ?, updated_at = ? WHERE id = ?", "failed", time.Now(), transcriptID)
			database.Exec("UPDATE lectures SET status = ?, updated_at = ? WHERE id = ?", "failed", time.Now(), payload.LectureID)
			return fmt.Errorf("transcription service failed: %w", transcriptionError)
		}

		// 5. Store segments in database
		databaseTransaction, transactionError := database.Begin()
		if transactionError != nil {
			return fmt.Errorf("failed to begin transaction: %w", transactionError)
		}
		defer databaseTransaction.Rollback()

		// Delete existing segments if any
		_, transactionError = databaseTransaction.Exec("DELETE FROM transcript_segments WHERE transcript_id = ?", transcriptID)
		if transactionError != nil {
			return fmt.Errorf("failed to delete old segments: %w", transactionError)
		}

		for _, segment := range segments {
			_, transactionError = databaseTransaction.Exec(`
				INSERT INTO transcript_segments (transcript_id, media_id, start_millisecond, end_millisecond, original_start_milliseconds, original_end_milliseconds, text, confidence, speaker)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, transcriptID, segment.MediaID, segment.StartMillisecond, segment.EndMillisecond, segment.OriginalStartMilliseconds, segment.OriginalEndMilliseconds, segment.Text, segment.Confidence, segment.Speaker)
			if transactionError != nil {
				return fmt.Errorf("failed to insert segment: %w", transactionError)
			}
		}

		// 6. Update media file durations based on segment end times
		for _, media := range mediaFiles {
			// Find the last segment for this media file
			var lastEndTime int64
			queryError := databaseTransaction.QueryRow(`
				SELECT MAX(end_millisecond)
				FROM transcript_segments
				WHERE media_id = ?
			`, media.ID).Scan(&lastEndTime)

			if queryError != nil {
				slog.Warn("Failed to query max segment end time", "media_id", media.ID, "error", queryError)
				continue
			}

			slog.Info("Found media segment end time", "media_id", media.ID, "last_end_milliseconds", lastEndTime, "last_end_seconds", lastEndTime/1000)

			if lastEndTime > 0 {
				_, updateError := databaseTransaction.Exec(`
					UPDATE lecture_media
					SET duration_milliseconds = ?
					WHERE id = ?
				`, lastEndTime, media.ID)

				if updateError != nil {
					slog.Warn("Failed to update media duration", "media_id", media.ID, "error", updateError)
				} else {
					slog.Info("Updated media duration", "media_id", media.ID, "duration_milliseconds", lastEndTime, "duration_seconds", lastEndTime/1000)
				}
			} else {
				slog.Warn("Media has no segments or zero duration", "media_id", media.ID)
			}
		}

		// 7. Finalize transcript
		_, transactionError = databaseTransaction.Exec("UPDATE transcripts SET status = ?, updated_at = ? WHERE id = ?", "completed", time.Now(), transcriptID)
		if transactionError != nil {
			return fmt.Errorf("failed to finalize transcript status: %w", transactionError)
		}

		if commitError := databaseTransaction.Commit(); commitError != nil {
			return fmt.Errorf("failed to commit transaction: %w", commitError)
		}

		if checkReadiness != nil {
			checkReadiness(database, payload.LectureID)
		}

		if broadcast != nil {
			broadcast("lecture:"+payload.LectureID, "lecture:updated", map[string]string{"lecture_id": payload.LectureID, "reason": "transcription_complete"})
		}

		updateProgress(100, "Transcription completed", nil, totalMetrics)
		return nil
	})

	queue.RegisterHandler(models.JobTypeIngestDocuments, func(jobContext context.Context, job *models.Job, updateProgress func(int, string, any, models.JobMetrics)) error {
		var totalMetrics models.JobMetrics
		var payload struct {
			LectureID    string `json:"lecture_id"`
			LanguageCode string `json:"language_code"`
		}
		if unmarshalingError := json.Unmarshal([]byte(job.Payload), &payload); unmarshalingError != nil {
			return fmt.Errorf("failed to unmarshal job payload: %w", unmarshalingError)
		}

		if payload.LanguageCode == "" {
			payload.LanguageCode = config.LLM.Language
		}

		// 1. Get reference documents for the lecture
		documentRows, databaseError := database.Query(`
			SELECT id, lecture_id, document_type, title, file_path, page_count, extraction_status, created_at, updated_at
			FROM reference_documents
			WHERE lecture_id = ?
		`, payload.LectureID)
		if databaseError != nil {
			return fmt.Errorf("failed to query documents: %w", databaseError)
		}

		var documentsList []models.ReferenceDocument
		for documentRows.Next() {
			var document models.ReferenceDocument
			if scanningError := documentRows.Scan(&document.ID, &document.LectureID, &document.DocumentType, &document.Title, &document.FilePath, &document.PageCount, &document.ExtractionStatus, &document.CreatedAt, &document.UpdatedAt); scanningError != nil {
				documentRows.Close()
				return fmt.Errorf("failed to scan document: %w", scanningError)
			}
			documentsList = append(documentsList, document)
		}
		documentRows.Close()

		totalDocuments := len(documentsList)
		for documentIndex, document := range documentsList {
			metadata := map[string]any{
				"document_index":  documentIndex + 1,
				"total_documents": totalDocuments,
				"document_title":  document.Title,
			}
			updateProgress(int(float64(documentIndex)/float64(totalDocuments)*100), "Ingesting reference documents...", metadata, totalMetrics)

			// 2. Update status to processing
			_, executionError := database.Exec("UPDATE reference_documents SET extraction_status = ?, updated_at = ? WHERE id = ?", "processing", time.Now(), document.ID)
			if executionError != nil {
				return fmt.Errorf("failed to update document status: %w", executionError)
			}

			// 3. Create output directory for pages in system temporary directory
			outputDirectory := filepath.Join(os.TempDir(), "lectures-assistant", "pages", document.ID)

			// 4. Run document processing
			pages, docMetrics, processingError := documentProcessor.ProcessDocument(jobContext, document, outputDirectory, payload.LanguageCode, func(progress int, message string) {
				updateProgress(progress, "Extracting and processing document pages...", metadata, totalMetrics)
			})

			totalMetrics.InputTokens += docMetrics.InputTokens
			totalMetrics.OutputTokens += docMetrics.OutputTokens
			totalMetrics.EstimatedCost += docMetrics.EstimatedCost

			if processingError != nil {
				// Clean up PNG images on failure
				os.RemoveAll(outputDirectory)
				slog.Warn("Cleaned up document images after processing failure", "document_id", document.ID, "output_directory", outputDirectory)

				database.Exec("UPDATE reference_documents SET extraction_status = ?, updated_at = ? WHERE id = ?", "failed", time.Now(), document.ID)
				database.Exec("UPDATE lectures SET status = ?, updated_at = ? WHERE id = ?", "failed", time.Now(), payload.LectureID)
				return fmt.Errorf("document processor failed for %s: %w", document.Title, processingError)
			}

			// 5. Store pages in database
			databaseTransaction, transactionError := database.Begin()
			if transactionError != nil {
				// Clean up PNG images since we can't store the data
				os.RemoveAll(outputDirectory)
				slog.Warn("Cleaned up document images after database transaction begin failure", "document_id", document.ID, "output_directory", outputDirectory)
				return fmt.Errorf("failed to begin transaction: %w", transactionError)
			}
			defer databaseTransaction.Rollback()

			// Delete existing pages if any
			_, transactionError = databaseTransaction.Exec("DELETE FROM reference_pages WHERE document_id = ?", document.ID)
			if transactionError != nil {
				// Clean up PNG images since we can't store the new data
				os.RemoveAll(outputDirectory)
				slog.Warn("Cleaned up document images after database delete failure", "document_id", document.ID, "output_directory", outputDirectory)
				return fmt.Errorf("failed to delete old pages: %w", transactionError)
			}

			for _, currentPage := range pages {
				_, transactionError = databaseTransaction.Exec(`
					INSERT INTO reference_pages (document_id, page_number, image_path, extracted_text)
					VALUES (?, ?, ?, ?)
				`, document.ID, currentPage.PageNumber, currentPage.ImagePath, currentPage.ExtractedText)
				if transactionError != nil {
					// Clean up PNG images since we can't store the page data
					os.RemoveAll(outputDirectory)
					slog.Warn("Cleaned up document images after page insert failure", "document_id", document.ID, "output_directory", outputDirectory)
					return fmt.Errorf("failed to insert page: %w", transactionError)
				}
			}

			// 6. Update document as completed
			_, transactionError = databaseTransaction.Exec("UPDATE reference_documents SET extraction_status = ?, page_count = ?, updated_at = ? WHERE id = ?", "completed", len(pages), time.Now(), document.ID)
			if transactionError != nil {
				// Clean up PNG images since we can't finalize the document
				os.RemoveAll(outputDirectory)
				slog.Warn("Cleaned up document images after document status update failure", "document_id", document.ID, "output_directory", outputDirectory)
				return fmt.Errorf("failed to finalize document status: %w", transactionError)
			}

			if commitError := databaseTransaction.Commit(); commitError != nil {
				// Clean up PNG images since we can't commit the data
				os.RemoveAll(outputDirectory)
				slog.Warn("Cleaned up document images after transaction commit failure", "document_id", document.ID, "output_directory", outputDirectory)
				return fmt.Errorf("failed to commit transaction: %w", commitError)
			}
		}

		if checkReadiness != nil {
			checkReadiness(database, payload.LectureID)
		}

		if broadcast != nil {
			broadcast("lecture:"+payload.LectureID, "lecture:updated", map[string]string{"lecture_id": payload.LectureID, "reason": "documents_complete"})
		}

		updateProgress(100, "Document ingestion completed", nil, totalMetrics)
		return nil
	})

	queue.RegisterHandler(models.JobTypeBuildMaterial, func(jobContext context.Context, job *models.Job, updateProgress func(int, string, any, models.JobMetrics)) error {
		var payload struct {
			LectureID               string `json:"lecture_id"`
			ExamID                  string `json:"exam_id"`
			Type                    string `json:"type"`
			Length                  string `json:"length"`
			LanguageCode            string `json:"language_code"`
			EnableDocumentsMatching string `json:"enable_documents_matching"`
			AdherenceThreshold      string `json:"adherence_threshold"`
			MaximumRetries          string `json:"maximum_retries"`
			// Models
			ModelDocumentsMatching string `json:"model_documents_matching"`
			ModelStructure         string `json:"model_structure"`
			ModelGeneration        string `json:"model_generation"`
			ModelAdherence         string `json:"model_adherence"`
			ModelPolishing         string `json:"model_polishing"`
		}
		if unmarshalingError := json.Unmarshal([]byte(job.Payload), &payload); unmarshalingError != nil {
			return fmt.Errorf("failed to unmarshal job payload: %w", unmarshalingError)
		}

		threshold, _ := strconv.Atoi(payload.AdherenceThreshold)
		maximumRetries, _ := strconv.Atoi(payload.MaximumRetries)
		options := models.GenerationOptions{
			EnableDocumentsMatching: payload.EnableDocumentsMatching == "true",
			AdherenceThreshold:      threshold,
			MaximumRetries:          maximumRetries,
			ModelDocumentsMatching:  payload.ModelDocumentsMatching,
			ModelStructure:          payload.ModelStructure,
			ModelGeneration:         payload.ModelGeneration,
			ModelAdherence:          payload.ModelAdherence,
			ModelPolishing:          payload.ModelPolishing,
		}

		if payload.Type == "" {
			payload.Type = "guide"
		}

		var lecture models.Lecture
		queryError := database.QueryRow("SELECT id, exam_id, title, description FROM lectures WHERE id = ?", payload.LectureID).Scan(&lecture.ID, &lecture.ExamID, &lecture.Title, &lecture.Description)
		if queryError != nil {
			return fmt.Errorf("failed to get lecture: %w", queryError)
		}

		transcriptRows, databaseError := database.Query(`
			SELECT text FROM transcript_segments 
			WHERE transcript_id = (SELECT id FROM transcripts WHERE lecture_id = ?)
			ORDER BY start_millisecond ASC
		`, payload.LectureID)
		if databaseError != nil {
			return fmt.Errorf("failed to query transcript: %w", databaseError)
		}

		var transcriptBuilder strings.Builder
		for transcriptRows.Next() {
			var text string
			if scanningError := transcriptRows.Scan(&text); scanningError == nil {
				transcriptBuilder.WriteString(text + " ")
			}
		}
		transcriptRows.Close()

		documentRows, databaseError := database.Query(`
			SELECT reference_documents.title, reference_pages.page_number, reference_pages.extracted_text
			FROM reference_documents
			JOIN reference_pages ON reference_documents.id = reference_pages.document_id
			WHERE reference_documents.lecture_id = ?
			ORDER BY reference_documents.id, reference_pages.page_number ASC
		`, payload.LectureID)
		if databaseError != nil {
			return fmt.Errorf("failed to query reference pages: %w", databaseError)
		}

		markdownReconstructor := markdown.NewReconstructor()
		markdownReconstructor.Language = payload.LanguageCode
		rootNode := &markdown.Node{Type: markdown.NodeDocument}
		currentDocumentTitle := ""

		for documentRows.Next() {
			var title, text string
			var pageNumber int
			if scanningError := documentRows.Scan(&title, &pageNumber, &text); scanningError == nil {
				if title != currentDocumentTitle {
					rootNode.Children = append(rootNode.Children, &markdown.Node{
						Type:    markdown.NodeHeading,
						Level:   1,
						Content: "Reference File: " + title,
					})
					currentDocumentTitle = title
				}
				rootNode.Children = append(rootNode.Children, &markdown.Node{
					Type:    markdown.NodeHeading,
					Level:   2,
					Content: fmt.Sprintf("Page %d", pageNumber),
				})
				rootNode.Children = append(rootNode.Children, &markdown.Node{
					Type:    markdown.NodeParagraph,
					Content: strings.TrimSpace(text),
				})
			}
		}
		documentRows.Close()

		referenceFilesContent := markdownReconstructor.Reconstruct(rootNode)

		var toolContent, toolTitle string
		var totalMetrics models.JobMetrics
		var generationError error

		switch payload.Type {
		case "flashcard":
			toolContent, toolTitle, totalMetrics, generationError = toolGenerator.GenerateFlashcards(jobContext, lecture, transcriptBuilder.String(), referenceFilesContent, payload.LanguageCode, options, func(progress int, message string, metadata any, metrics models.JobMetrics) {
				updateProgress(progress, message, metadata, metrics)
			})
		case "quiz":
			toolContent, toolTitle, totalMetrics, generationError = toolGenerator.GenerateQuiz(jobContext, lecture, transcriptBuilder.String(), referenceFilesContent, payload.LanguageCode, options, func(progress int, message string, metadata any, metrics models.JobMetrics) {
				updateProgress(progress, message, metadata, metrics)
			})
		default:
			toolContent, toolTitle, generationError = toolGenerator.GenerateStudyGuide(jobContext, lecture, transcriptBuilder.String(), referenceFilesContent, payload.Length, payload.LanguageCode, options, func(progress int, message string, metadata any, metrics models.JobMetrics) {
				// Metrics are already aggregated inside GenerateStudyGuide and passed back via this callback
				totalMetrics = metrics
				updateProgress(progress, message, metadata, metrics)
			})
		}

		if generationError != nil {
			return fmt.Errorf("tool generation failed: %w", generationError)
		}

		// Parse citations and convert to standard footnotes
		slog.Debug("Before ParseCitations", "content_length", len(toolContent), "has_triple_braces", strings.Contains(toolContent, "{{{"))
		finalToolContent, citations := markdownReconstructor.ParseCitations(toolContent)
		slog.Debug("After ParseCitations", "citations_found", len(citations))

		// Improve footnotes using AI if it's a guide and we have citations
		if payload.Type == "guide" && len(citations) > 0 {
			updatedCitations, footnoteMetrics, err := toolGenerator.ProcessFootnotesAI(jobContext, citations, payload.LanguageCode, options)
			totalMetrics.InputTokens += footnoteMetrics.InputTokens
			totalMetrics.OutputTokens += footnoteMetrics.OutputTokens
			totalMetrics.EstimatedCost += footnoteMetrics.EstimatedCost
			if err == nil {
				citations = updatedCitations
			}
		}

		// If it's a guide, append the footnote definitions to the end
		if payload.Type == "guide" {
			finalToolContent = markdownReconstructor.AppendCitations(finalToolContent, citations)
		}

		updateProgress(95, "Finalizing tool...", nil, totalMetrics)

		toolID := uuid.New().String()

		transaction, err := database.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction for tool storage: %w", err)
		}
		defer transaction.Rollback()

		_, executionError := transaction.Exec(`
			INSERT INTO tools (id, exam_id, type, title, language_code, content, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, toolID, payload.ExamID, payload.Type, toolTitle, payload.LanguageCode, finalToolContent, time.Now(), time.Now())
		if executionError != nil {
			return fmt.Errorf("failed to store tool: %w", executionError)
		}

		// Store citation metadata in structured table
		for _, citation := range citations {
			metadataJSON, _ := json.Marshal(map[string]any{
				"footnote_number": citation.Number,
				"description":     citation.Description,
				"pages":           citation.Pages,
			})
			_, executionError = transaction.Exec(`
				INSERT INTO tool_source_references (tool_id, source_type, source_id, metadata)
				VALUES (?, ?, ?, ?)
			`, toolID, "document", citation.File, string(metadataJSON))
			if executionError != nil {
				slog.Error("Failed to store tool source reference", "toolID", toolID, "error", executionError)
			}
		}

		if commitError := transaction.Commit(); commitError != nil {
			return fmt.Errorf("failed to commit tool storage: %w", commitError)
		}

		if broadcast != nil {
			broadcast("course:"+payload.ExamID, "tool:created", map[string]string{"course_id": payload.ExamID, "tool_id": toolID})
		}

		job.Result = fmt.Sprintf(`{"tool_id": "%s"}`, toolID)
		return nil
	})

	queue.RegisterHandler(models.JobTypeDownloadGoogleDrive, func(jobContext context.Context, job *models.Job, updateProgress func(int, string, any, models.JobMetrics)) error {
		var payload struct {
			FileID     string `json:"file_id"`
			OAuthToken string `json:"oauth_token"`
			Filename   string `json:"filename"`
		}
		if unmarshalingError := json.Unmarshal([]byte(job.Payload), &payload); unmarshalingError != nil {
			return fmt.Errorf("failed to unmarshal job payload: %w", unmarshalingError)
		}

		// 1. Initialize Google Drive Service
		tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: payload.OAuthToken})
		driveService, serviceError := drive.NewService(jobContext, option.WithTokenSource(tokenSource))
		if serviceError != nil {
			return fmt.Errorf("failed to create drive service: %w", serviceError)
		}

		// 2. Fetch File Metadata (to get size)
		fileMetadata, metadataError := driveService.Files.Get(payload.FileID).Fields("size, name").Context(jobContext).Do()
		if metadataError != nil {
			return fmt.Errorf("failed to get file metadata: %w", metadataError)
		}

		if payload.Filename == "" {
			payload.Filename = fileMetadata.Name
		}

		// 3. Prepare Staging Area
		uploadID := job.ID // Use job ID as upload ID for simplicity
		uploadDirectory := filepath.Join(os.TempDir(), "lectures-uploads", uploadID)
		if mkdirError := os.MkdirAll(uploadDirectory, 0755); mkdirError != nil {
			return fmt.Errorf("failed to create staging directory: %w", mkdirError)
		}

		metadataFilePath := filepath.Join(uploadDirectory, "metadata.json")
		metadataFile, createFileError := os.Create(metadataFilePath)
		if createFileError != nil {
			return fmt.Errorf("failed to create metadata file: %w", createFileError)
		}
		json.NewEncoder(metadataFile).Encode(map[string]any{
			"filename":        payload.Filename,
			"file_size_bytes": fileMetadata.Size,
		})
		metadataFile.Close()

		dataFilePath := filepath.Join(uploadDirectory, "upload.data")
		dataFile, openFileError := os.Create(dataFilePath)
		if openFileError != nil {
			return fmt.Errorf("failed to create data file: %w", openFileError)
		}
		defer dataFile.Close()

		// 4. Start Download
		driveResponse, downloadError := driveService.Files.Get(payload.FileID).Download()
		if downloadError != nil {
			return fmt.Errorf("failed to start download: %w", downloadError)
		}
		defer driveResponse.Body.Close()

		// 5. Stream with Progress
		streamingBuffer := make([]byte, 1024*1024) // 1MB buffer
		var totalDownloadedBytes int64
		lastUpdateTimestamp := time.Now()

		for {
			bytesReadCount, readingError := driveResponse.Body.Read(streamingBuffer)
			if bytesReadCount > 0 {
				_, writingError := dataFile.Write(streamingBuffer[:bytesReadCount])
				if writingError != nil {
					return fmt.Errorf("failed to write to data file: %w", writingError)
				}
				totalDownloadedBytes += int64(bytesReadCount)

				// Throttled progress updates
				if time.Since(lastUpdateTimestamp) > 500*time.Millisecond || totalDownloadedBytes == fileMetadata.Size {
					downloadProgress := 0
					if fileMetadata.Size > 0 {
						downloadProgress = int(float64(totalDownloadedBytes) / float64(fileMetadata.Size) * 100)
					}
					updateProgress(downloadProgress, fmt.Sprintf("Downloading from Google Drive: %d/%d bytes", totalDownloadedBytes, fileMetadata.Size), map[string]any{
						"bytes_downloaded": totalDownloadedBytes,
						"total_bytes":      fileMetadata.Size,
						"upload_id":        uploadID,
					}, models.JobMetrics{})
					lastUpdateTimestamp = time.Now()
				}
			}

			if readingError != nil {
				if readingError == io.EOF {
					break
				}
				return fmt.Errorf("reading error during download: %w", readingError)
			}
		}

		job.Result = fmt.Sprintf(`{"upload_id": "%s", "filename": "%s"}`, uploadID, payload.Filename)
		return nil
	})

	queue.RegisterHandler(models.JobTypePublishMaterial, func(jobContext context.Context, job *models.Job, updateProgress func(int, string, any, models.JobMetrics)) error {
		var totalMetrics models.JobMetrics

		var payload struct {
			ToolID        string          `json:"tool_id"`
			LanguageCode  string          `json:"language_code"`
			Format        string          `json:"format"` // "pdf", "docx", "md"
			IncludeImages json.RawMessage `json:"include_images"`
			IncludeQRCode json.RawMessage `json:"include_qr_code"`
		}
		if unmarshalingError := json.Unmarshal([]byte(job.Payload), &payload); unmarshalingError != nil {
			return fmt.Errorf("failed to unmarshal job payload: %w", unmarshalingError)
		}

		// Parse include_images boolean (it might be a string "true" or a boolean true)
		includeImages := true
		if len(payload.IncludeImages) > 0 {
			rawStr := string(payload.IncludeImages)
			if rawStr == "false" || rawStr == `"false"` {
				includeImages = false
			}
		}

		// Parse include_qr_code boolean
		includeQRCode := false
		if len(payload.IncludeQRCode) > 0 {
			rawStr := string(payload.IncludeQRCode)
			if rawStr == "true" || rawStr == `"true"` {
				includeQRCode = true
			}
		}

		if payload.Format == "" {
			payload.Format = "pdf"
		}

		var tool models.Tool
		var examID string
		queryError := database.QueryRow("SELECT id, exam_id, type, title, language_code, content, created_at FROM tools WHERE id = ?", payload.ToolID).Scan(&tool.ID, &examID, &tool.Type, &tool.Title, &tool.LanguageCode, &tool.Content, &tool.CreatedAt)
		if queryError != nil {
			return fmt.Errorf("failed to get tool: %w", queryError)
		}

		// Fetch Exam Title
		var examTitle string
		if err := database.QueryRow("SELECT title FROM exams WHERE id = ?", examID).Scan(&examTitle); err != nil {
			slog.Warn("Failed to fetch exam title for metadata", "examID", examID, "error", err)
		}

		if payload.LanguageCode == "" {
			payload.LanguageCode = tool.LanguageCode
		}
		if payload.LanguageCode == "" {
			payload.LanguageCode = config.LLM.Language
		}

		exportDirectory := filepath.Join(config.Storage.DataDirectory, "files", "exports", tool.ID)
		if mkdirError := os.MkdirAll(exportDirectory, 0755); mkdirError != nil {
			return fmt.Errorf("failed to create export directory: %w", mkdirError)
		}

		// Use sanitized tool title as filename
		outputExtension := "." + payload.Format
		safeFilename := sanitizeFilename(tool.Title) + outputExtension
		outputPath := filepath.Join(exportDirectory, safeFilename)
		slog.Info("Exporting tool", "tool_title", tool.Title, "format", payload.Format, "filename", safeFilename, "path", outputPath)

		// 3. Prepare content for PDF/Docx/MD (convert JSON to Markdown if needed)
		contentToConvert := tool.Content
		if tool.Type == "flashcard" || tool.Type == "quiz" {
			markdownReconstructor := markdown.NewReconstructor()
			markdownReconstructor.Language = payload.LanguageCode
			rootNode := &markdown.Node{Type: markdown.NodeDocument}

			rootNode.Children = append(rootNode.Children, &markdown.Node{
				Type:    markdown.NodeHeading,
				Level:   1,
				Content: tool.Title,
			})

			switch tool.Type {
			case "flashcard":
				var flashcards []map[string]string
				if unmarshalingError := json.Unmarshal([]byte(tool.Content), &flashcards); unmarshalingError == nil {
					for _, flashcard := range flashcards {
						rootNode.Children = append(rootNode.Children, &markdown.Node{
							Type:    markdown.NodeHeading,
							Level:   2,
							Content: flashcard["front"],
						})
						rootNode.Children = append(rootNode.Children, &markdown.Node{
							Type:    markdown.NodeParagraph,
							Content: flashcard["back"],
						})
					}
				}
			case "quiz":
				var quiz []map[string]any
				if unmarshalingError := json.Unmarshal([]byte(tool.Content), &quiz); unmarshalingError == nil {
					for _, quizItem := range quiz {
						rootNode.Children = append(rootNode.Children, &markdown.Node{
							Type:    markdown.NodeHeading,
							Level:   2,
							Content: fmt.Sprintf("%v", quizItem["question"]),
						})

						if options, ok := quizItem["options"].([]any); ok {
							for _, option := range options {
								rootNode.Children = append(rootNode.Children, &markdown.Node{
									Type:     markdown.NodeListItem,
									Content:  fmt.Sprintf("%v", option),
									ListType: markdown.ListUnordered,
								})
							}
						}

						rootNode.Children = append(rootNode.Children, &markdown.Node{
							Type:    markdown.NodeParagraph,
							Content: fmt.Sprintf("**Correct Answer:** %v", quizItem["correct_answer"]),
						})
						rootNode.Children = append(rootNode.Children, &markdown.Node{
							Type:    markdown.NodeParagraph,
							Content: fmt.Sprintf("*Explanation:* %v", quizItem["explanation"]),
						})
					}
				}
			}
			contentToConvert = markdownReconstructor.Reconstruct(rootNode)
		}

		if tool.Type == "guide" {
			slog.Info("Starting tool content parsing and enrichment", "toolID", tool.ID)
			markdownParser := markdown.NewParser()
			ast := markdownParser.Parse(contentToConvert)

			if includeImages {
				// 1. Get structured citation metadata from DB (The Source of Truth)
				citationMetadata := make(map[int]struct {
					File  string
					Pages []int
				})
				refRows, err := database.Query("SELECT source_id, metadata FROM tool_source_references WHERE tool_id = ?", tool.ID)
				if err == nil {
					for refRows.Next() {
						var sourceID, metadataStr string
						if err := refRows.Scan(&sourceID, &metadataStr); err == nil {
							var meta struct {
								FootnoteNumber int   `json:"footnote_number"`
								Pages          []int `json:"pages"`
							}
							if json.Unmarshal([]byte(metadataStr), &meta) == nil {
								citationMetadata[meta.FootnoteNumber] = struct {
									File  string
									Pages []int
								}{File: sourceID, Pages: meta.Pages}
							}
						}
					}
					refRows.Close()
				}

				// 2. Manually enrich AST nodes with the reliable metadata
				var enrichNodeMetadata func(*markdown.Node)
				enrichNodeMetadata = func(node *markdown.Node) {
					if node.Type == markdown.NodeFootnote {
						if info, ok := citationMetadata[node.FootnoteNumber]; ok {
							node.SourceFile = info.File
							node.SourcePages = info.Pages
						}
					}
					for _, child := range node.Children {
						enrichNodeMetadata(child)
					}
				}
				enrichNodeMetadata(ast)

				// 3. Pre-fetch all page paths for this exam
				pageMap := make(map[string]string) // Key: "filename:page"
				slog.Info("Pre-fetching page image paths from database", "examID", examID)
				rows, err := database.Query(`
					SELECT reference_documents.original_filename, reference_documents.title, reference_pages.page_number, reference_pages.image_path
					FROM reference_pages
					JOIN reference_documents ON reference_pages.document_id = reference_documents.id
					JOIN lectures ON reference_documents.lecture_id = lectures.id
					WHERE lectures.exam_id = ?
				`, examID)
				if err == nil {
					for rows.Next() {
						var originalFilename, title, imagePath string
						var pageNumber int
						if err := rows.Scan(&originalFilename, &title, &pageNumber, &imagePath); err == nil {
							if originalFilename != "" {
								pageMap[fmt.Sprintf("%s:%d", originalFilename, pageNumber)] = imagePath
							}
							if title != "" {
								pageMap[fmt.Sprintf("%s:%d", title, pageNumber)] = imagePath
							}
						}
					}
					rows.Close()
				}
				slog.Info("Pre-fetched pages for enrichment", "count", len(pageMap))

				imageResolver := func(filename string, pageNumber int) string {
					key := fmt.Sprintf("%s:%d", filename, pageNumber)
					return pageMap[key]
				}

				slog.Info("Starting AST enrichment with cited images")
				markdown.EnrichWithCitedImages(ast, imageResolver)
				slog.Info("Finished AST enrichment with cited images")
			}

			markdownReconstructor := markdown.NewReconstructor()
			markdownReconstructor.Language = payload.LanguageCode
			contentToConvert = markdownReconstructor.Reconstruct(ast)
			slog.Info("Finished tool content reconstruction", "contentLength", len(contentToConvert))
		}

		updateProgress(30, "Gathering lecture metadata...", nil, models.JobMetrics{})

		// Get lectures for this exam to collect metadata
		lectureRows, databaseError := database.Query("SELECT id, specified_date FROM lectures WHERE exam_id = ? AND status = 'ready'", examID)
		if databaseError != nil {
			return fmt.Errorf("failed to query lectures: %w", databaseError)
		}

		type lectureMeta struct {
			id            string
			specifiedDate sql.NullTime
		}
		var lectures []lectureMeta
		for lectureRows.Next() {
			var lecture lectureMeta
			if err := lectureRows.Scan(&lecture.id, &lecture.specifiedDate); err == nil {
				lectures = append(lectures, lecture)
			}
		}
		lectureRows.Close()

		var audioFiles []markdown.AudioFileMetadata
		var referenceFiles []markdown.ReferenceFileMetadata
		var finalDate time.Time = tool.CreatedAt

		for _, lecture := range lectures {
			if lecture.specifiedDate.Valid {
				finalDate = lecture.specifiedDate.Time
			}

			// Get media files
			mediaRows, mediaQueryError := database.Query("SELECT original_filename, file_path, duration_milliseconds FROM lecture_media WHERE lecture_id = ? ORDER BY sequence_order", lecture.id)
			if mediaQueryError == nil {
				for mediaRows.Next() {
					var originalFilename sql.NullString
					var filePath string
					var durationMs int64
					if scanError := mediaRows.Scan(&originalFilename, &filePath, &durationMs); scanError == nil {
						filename := filepath.Base(filePath)
						if originalFilename.Valid && originalFilename.String != "" {
							filename = originalFilename.String
						}
						audioFiles = append(audioFiles, markdown.AudioFileMetadata{
							Filename: filename,
							Duration: durationMs / 1000,
						})
					}
				}
				mediaRows.Close()
			}

			// Get documents
			docRows, docQueryError := database.Query("SELECT title, original_filename, page_count FROM reference_documents WHERE lecture_id = ?", lecture.id)
			if docQueryError == nil {
				for docRows.Next() {
					var title string
					var originalFilename sql.NullString
					var pageCount int
					if scanError := docRows.Scan(&title, &originalFilename, &pageCount); scanError == nil {
						filename := title
						if originalFilename.Valid && originalFilename.String != "" {
							filename = originalFilename.String
						}
						referenceFiles = append(referenceFiles, markdown.ReferenceFileMetadata{
							Filename:  filename,
							PageCount: pageCount,
						})
					}
				}
				docRows.Close()
			}
		}

		// Generate abstract
		updateProgress(40, "Generating document abstract...", nil, totalMetrics)
		abstract := ""
		if contentToConvert != "" && toolGenerator != nil {
			generatedAbstract, abstractMetrics, generationError := toolGenerator.GenerateAbstract(jobContext, contentToConvert, payload.LanguageCode, "")
			if generationError == nil {
				abstract = generatedAbstract
				totalMetrics.InputTokens += abstractMetrics.InputTokens
				totalMetrics.OutputTokens += abstractMetrics.OutputTokens
				totalMetrics.EstimatedCost += abstractMetrics.EstimatedCost
			}
		}

		updateProgress(50, fmt.Sprintf("Generating %s document...", payload.Format), nil, models.JobMetrics{})
		options := markdown.ConversionOptions{
			Language:       payload.LanguageCode,
			Description:    abstract,
			CourseTitle:    examTitle,
			CreationDate:   finalDate,
			ReferenceFiles: referenceFiles,
			AudioFiles:     audioFiles,
		}

		originalContent := contentToConvert
		generateFile := func(currentContent string, currentOptions markdown.ConversionOptions) error {
			contentWithHeader := currentContent
			if payload.Format == "md" || payload.Format == "docx" {
				metadataHeader := markdownConverter.GenerateMetadataHeader(currentOptions)
				contentWithHeader = metadataHeader + currentContent
			}

			if payload.Format == "md" {
				return markdownConverter.SaveMarkdown(contentWithHeader, outputPath)
			}

			updateProgress(60, fmt.Sprintf("Converting %s document...", payload.Format), nil, models.JobMetrics{})
			htmlContent, err := markdownConverter.MarkdownToHTML(contentWithHeader)
			if err != nil {
				return fmt.Errorf("failed to convert to HTML: %w", err)
			}

			if payload.Format == "docx" {
				return markdownConverter.HTMLToDocx(htmlContent, outputPath, currentOptions)
			}
			return markdownConverter.HTMLToPDF(htmlContent, outputPath, currentOptions)
		}

		// 1. Initial generation
		if err := generateFile(originalContent, options); err != nil {
			return fmt.Errorf("initial generation failed: %w", err)
		}

		// 2. Optional second pass for QR Code
		if includeQRCode {
			slog.Info("QR Code generation requested", "toolID", tool.ID)
			updateProgress(70, "Uploading document for QR code generation...", nil, models.JobMetrics{})

			downloadURL, uploadError := uploadToTmpFiles(outputPath)
			if uploadError != nil {
				slog.Error("Failed to upload for QR code", "error", uploadError)
			} else {
				slog.Info("Document uploaded for QR code", "url", downloadURL)

				qrCodePath := filepath.Join(os.TempDir(), fmt.Sprintf("qrcode-%s.png", uuid.New().String()))
				if qrErr := qrcode.WriteFile(downloadURL, qrcode.Medium, 256, qrCodePath); qrErr != nil {
					slog.Error("Failed to generate QR code image", "error", qrErr)
				} else {
					updateProgress(85, "Re-generating document with QR code...", nil, models.JobMetrics{})
					options.QRCodePath = qrCodePath
					slog.Info("Re-generating document with QR code included", "format", payload.Format)

					if err := generateFile(originalContent, options); err != nil {
						slog.Error("Failed to re-generate with QR code", "error", err)
					}
					os.Remove(qrCodePath)
				}
			}
		}

		updateProgress(100, "Export completed", map[string]string{"file_path": outputPath, "format": payload.Format}, totalMetrics)

		slog.Info("Export completed with costs",
			"file_path", outputPath,
			"format", payload.Format,
			"input_tokens", totalMetrics.InputTokens,
			"output_tokens", totalMetrics.OutputTokens,
			"estimated_cost_usd", totalMetrics.EstimatedCost)
		job.Result = fmt.Sprintf(`{"file_path": "%s", "format": "%s"}`, outputPath, payload.Format)
		return nil
	})
}

func uploadToTmpFiles(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	fileName := filepath.Base(filePath)
	if strings.ToLower(filepath.Ext(fileName)) == ".md" {
		fileName = strings.TrimSuffix(fileName, filepath.Ext(fileName)) + ".txt"
	}

	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return "", err
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return "", err
	}
	writer.Close()

	req, err := http.NewRequest("POST", "https://tmpfiles.org/api/v1/upload", body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyPreview := string(respBody)
		if len(bodyPreview) > 200 {
			bodyPreview = bodyPreview[:200] + "..."
		}
		return "", fmt.Errorf("upload failed with status %s: %s", resp.Status, bodyPreview)
	}

	var result struct {
		Status string `json:"status"`
		Data   struct {
			URL string `json:"url"`
		} `json:"data"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		bodyPreview := string(respBody)
		if len(bodyPreview) > 200 {
			bodyPreview = bodyPreview[:200] + "..."
		}
		return "", fmt.Errorf("failed to decode tmpfiles response: %w. Body preview: %s", err, bodyPreview)
	}

	if result.Status != "success" {
		return "", fmt.Errorf("tmpfiles upload failed: %s", result.Message)
	}

	// Transform to direct download link: https://tmpfiles.org/12345/file -> https://tmpfiles.org/dl/12345/file
	directLink := strings.Replace(result.Data.URL, "https://tmpfiles.org/", "https://tmpfiles.org/dl/", 1)
	// Also handle http if returned
	directLink = strings.Replace(directLink, "http://tmpfiles.org/", "https://tmpfiles.org/dl/", 1)

	return directLink, nil
}
