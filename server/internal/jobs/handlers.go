package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
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
) {
	queue.RegisterHandler(models.JobTypeTranscribeMedia, func(jobContext context.Context, job *models.Job, updateProgress func(int, string, any, models.JobMetrics)) error {
		var payload struct {
			LectureID string `json:"lecture_id"`
		}
		if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil {
			return fmt.Errorf("failed to unmarshal job payload: %w", err)
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
		defer mediaRows.Close()

		var mediaFiles []models.LectureMedia
		for mediaRows.Next() {
			var media models.LectureMedia
			if err := mediaRows.Scan(&media.ID, &media.LectureID, &media.MediaType, &media.SequenceOrder, &media.FilePath, &media.CreatedAt); err != nil {
				return fmt.Errorf("failed to scan media file: %w", err)
			}
			mediaFiles = append(mediaFiles, media)
		}

		if len(mediaFiles) == 0 {
			return fmt.Errorf("no media files found for lecture: %s", payload.LectureID)
		}

		// 2. Create transcript record if not exists
		transcriptID := uuid.New().String()
		_, err := database.Exec(`
			INSERT OR IGNORE INTO transcripts (id, lecture_id, status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?)
		`, transcriptID, payload.LectureID, "processing", time.Now(), time.Now())
		if err != nil {
			return fmt.Errorf("failed to create transcript: %w", err)
		}

		// Get the actual transcript ID (in case it already existed)
		err = database.QueryRow("SELECT id FROM transcripts WHERE lecture_id = ?", payload.LectureID).Scan(&transcriptID)
		if err != nil {
			return fmt.Errorf("failed to get transcript ID: %w", err)
		}

		// Update transcript status to processing
		_, err = database.Exec("UPDATE transcripts SET status = ?, updated_at = ? WHERE id = ?", "processing", time.Now(), transcriptID)
		if err != nil {
			return fmt.Errorf("failed to update transcript status: %w", err)
		}

		// 3. Create temporary directory for transcription
		temporaryDirectory := filepath.Join(os.TempDir(), "lectures-jobs", job.ID)
		if err := os.MkdirAll(temporaryDirectory, 0755); err != nil {
			return fmt.Errorf("failed to create temporary directory: %w", err)
		}
		defer os.RemoveAll(temporaryDirectory)

		// 4. Run transcription
		segments, err := transcriptionService.TranscribeLecture(jobContext, mediaFiles, temporaryDirectory, func(progress int, message string, metadata any) {
			updateProgress(progress, "Transcribing media files...", metadata, models.JobMetrics{})
		})
		if err != nil {
			database.Exec("UPDATE transcripts SET status = ?, updated_at = ? WHERE id = ?", "failed", time.Now(), transcriptID)
			database.Exec("UPDATE lectures SET status = ?, updated_at = ? WHERE id = ?", "failed", time.Now(), payload.LectureID)
			return fmt.Errorf("transcription service failed: %w", err)
		}

		// 5. Store segments in database
		databaseTransaction, err := database.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer databaseTransaction.Rollback()

		// Delete existing segments if any
		_, err = databaseTransaction.Exec("DELETE FROM transcript_segments WHERE transcript_id = ?", transcriptID)
		if err != nil {
			return fmt.Errorf("failed to delete old segments: %w", err)
		}

		for _, segment := range segments {
			_, err = databaseTransaction.Exec(`
				INSERT INTO transcript_segments (transcript_id, media_id, start_millisecond, end_millisecond, original_start_milliseconds, original_end_milliseconds, text, confidence, speaker)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, transcriptID, segment.MediaID, segment.StartMillisecond, segment.EndMillisecond, segment.OriginalStartMilliseconds, segment.OriginalEndMilliseconds, segment.Text, segment.Confidence, segment.Speaker)
			if err != nil {
				return fmt.Errorf("failed to insert segment: %w", err)
			}
		}

		// 6. Update media file durations based on segment end times
		for _, media := range mediaFiles {
			// Find the last segment for this media file
			var lastEndTime int64
			err := databaseTransaction.QueryRow(`
				SELECT MAX(end_millisecond)
				FROM transcript_segments
				WHERE media_id = ?
			`, media.ID).Scan(&lastEndTime)

			if err != nil {
				slog.Warn("Failed to query max segment end time", "media_id", media.ID, "error", err)
				continue
			}

			slog.Info("Found media segment end time", "media_id", media.ID, "last_end_milliseconds", lastEndTime, "last_end_seconds", lastEndTime/1000)

			if lastEndTime > 0 {
				_, err = databaseTransaction.Exec(`
					UPDATE lecture_media
					SET duration_milliseconds = ?
					WHERE id = ?
				`, lastEndTime, media.ID)

				if err != nil {
					slog.Warn("Failed to update media duration", "media_id", media.ID, "error", err)
				} else {
					slog.Info("Updated media duration", "media_id", media.ID, "duration_milliseconds", lastEndTime, "duration_seconds", lastEndTime/1000)
				}
			} else {
				slog.Warn("Media has no segments or zero duration", "media_id", media.ID)
			}
		}

		// 7. Finalize transcript
		_, err = databaseTransaction.Exec("UPDATE transcripts SET status = ?, updated_at = ? WHERE id = ?", "completed", time.Now(), transcriptID)
		if err != nil {
			return fmt.Errorf("failed to finalize transcript status: %w", err)
		}

		if err := databaseTransaction.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}

		if checkReadiness != nil {
			checkReadiness(database, payload.LectureID)
		}
		updateProgress(100, "Transcription completed", nil, models.JobMetrics{})
		return nil
	})

	queue.RegisterHandler(models.JobTypeIngestDocuments, func(jobContext context.Context, job *models.Job, updateProgress func(int, string, any, models.JobMetrics)) error {
		var payload struct {
			LectureID    string `json:"lecture_id"`
			LanguageCode string `json:"language_code"`
		}
		if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil {
			return fmt.Errorf("failed to unmarshal job payload: %w", err)
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
		defer documentRows.Close()

		var documentsList []models.ReferenceDocument
		for documentRows.Next() {
			var document models.ReferenceDocument
			if err := documentRows.Scan(&document.ID, &document.LectureID, &document.DocumentType, &document.Title, &document.FilePath, &document.PageCount, &document.ExtractionStatus, &document.CreatedAt, &document.UpdatedAt); err != nil {
				return fmt.Errorf("failed to scan document: %w", err)
			}
			documentsList = append(documentsList, document)
		}

		totalDocuments := len(documentsList)
		for documentIndex, document := range documentsList {
			metadata := map[string]any{
				"document_index":  documentIndex + 1,
				"total_documents": totalDocuments,
				"document_title":  document.Title,
			}
			updateProgress(int(float64(documentIndex)/float64(totalDocuments)*100), "Ingesting reference documents...", metadata, models.JobMetrics{})

			// 2. Update status to processing
			_, err := database.Exec("UPDATE reference_documents SET extraction_status = ?, updated_at = ? WHERE id = ?", "processing", time.Now(), document.ID)
			if err != nil {
				return fmt.Errorf("failed to update document status: %w", err)
			}

			// 3. Create output directory for pages in system tmp
			outputDirectory := filepath.Join(os.TempDir(), "lectures-documents", document.ID)

			// 4. Run document processing
			pages, err := documentProcessor.ProcessDocument(jobContext, document, outputDirectory, payload.LanguageCode, func(progress int, message string) {
				updateProgress(progress, "Extracting and processing document pages...", metadata, models.JobMetrics{})
			})
			if err != nil {
				// Clean up PNG images on failure
				os.RemoveAll(outputDirectory)
				slog.Warn("Cleaned up document images after processing failure", "document_id", document.ID, "output_directory", outputDirectory)

				database.Exec("UPDATE reference_documents SET extraction_status = ?, updated_at = ? WHERE id = ?", "failed", time.Now(), document.ID)
				database.Exec("UPDATE lectures SET status = ?, updated_at = ? WHERE id = ?", "failed", time.Now(), payload.LectureID)
				return fmt.Errorf("document processor failed for %s: %w", document.Title, err)
			}

			// 5. Store pages in database
			databaseTransaction, err := database.Begin()
			if err != nil {
				// Clean up PNG images since we can't store the data
				os.RemoveAll(outputDirectory)
				slog.Warn("Cleaned up document images after database transaction begin failure", "document_id", document.ID, "output_directory", outputDirectory)
				return fmt.Errorf("failed to begin transaction: %w", err)
			}
			defer databaseTransaction.Rollback()

			// Delete existing pages if any
			_, err = databaseTransaction.Exec("DELETE FROM reference_pages WHERE document_id = ?", document.ID)
			if err != nil {
				// Clean up PNG images since we can't store the new data
				os.RemoveAll(outputDirectory)
				slog.Warn("Cleaned up document images after database delete failure", "document_id", document.ID, "output_directory", outputDirectory)
				return fmt.Errorf("failed to delete old pages: %w", err)
			}

			for _, currentPage := range pages {
				_, err = databaseTransaction.Exec(`
					INSERT INTO reference_pages (document_id, page_number, image_path, extracted_text)
					VALUES (?, ?, ?, ?)
				`, document.ID, currentPage.PageNumber, currentPage.ImagePath, currentPage.ExtractedText)
				if err != nil {
					// Clean up PNG images since we can't store the page data
					os.RemoveAll(outputDirectory)
					slog.Warn("Cleaned up document images after page insert failure", "document_id", document.ID, "output_directory", outputDirectory)
					return fmt.Errorf("failed to insert page: %w", err)
				}
			}

			// 6. Update document as completed
			_, err = databaseTransaction.Exec("UPDATE reference_documents SET extraction_status = ?, page_count = ?, updated_at = ? WHERE id = ?", "completed", len(pages), time.Now(), document.ID)
			if err != nil {
				// Clean up PNG images since we can't finalize the document
				os.RemoveAll(outputDirectory)
				slog.Warn("Cleaned up document images after document status update failure", "document_id", document.ID, "output_directory", outputDirectory)
				return fmt.Errorf("failed to finalize document status: %w", err)
			}

			if err := databaseTransaction.Commit(); err != nil {
				// Clean up PNG images since we can't commit the data
				os.RemoveAll(outputDirectory)
				slog.Warn("Cleaned up document images after transaction commit failure", "document_id", document.ID, "output_directory", outputDirectory)
				return fmt.Errorf("failed to commit transaction: %w", err)
			}
		}

		if checkReadiness != nil {
			checkReadiness(database, payload.LectureID)
		}
		updateProgress(100, "Document ingestion completed", nil, models.JobMetrics{})
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
		if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil {
			return fmt.Errorf("failed to unmarshal job payload: %w", err)
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
		err := database.QueryRow("SELECT id, exam_id, title, description FROM lectures WHERE id = ?", payload.LectureID).Scan(&lecture.ID, &lecture.ExamID, &lecture.Title, &lecture.Description)
		if err != nil {
			return fmt.Errorf("failed to get lecture: %w", err)
		}

		transcriptRows, databaseError := database.Query(`
			SELECT text FROM transcript_segments 
			WHERE transcript_id = (SELECT id FROM transcripts WHERE lecture_id = ?)
			ORDER BY start_millisecond ASC
		`, payload.LectureID)
		if databaseError != nil {
			return fmt.Errorf("failed to query transcript: %w", databaseError)
		}
		defer transcriptRows.Close()

		var transcriptBuilder strings.Builder
		for transcriptRows.Next() {
			var text string
			if err := transcriptRows.Scan(&text); err == nil {
				transcriptBuilder.WriteString(text + " ")
			}
		}

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
		defer documentRows.Close()

		markdownReconstructor := markdown.NewReconstructor()
		rootNode := &markdown.Node{Type: markdown.NodeDocument}
		currentDocumentTitle := ""

		for documentRows.Next() {
			var title, text string
			var pageNumber int
			if err := documentRows.Scan(&title, &pageNumber, &text); err == nil {
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

		referenceFilesContent := markdownReconstructor.Reconstruct(rootNode)

		var toolContent, toolTitle string
		var genErr error

		switch payload.Type {
		case "flashcard":
			toolContent, toolTitle, genErr = toolGenerator.GenerateFlashcards(jobContext, lecture, transcriptBuilder.String(), referenceFilesContent, payload.LanguageCode, options, func(progress int, message string, metadata any, metrics models.JobMetrics) {
				updateProgress(progress, message, metadata, metrics)
			})
		case "quiz":
			toolContent, toolTitle, genErr = toolGenerator.GenerateQuiz(jobContext, lecture, transcriptBuilder.String(), referenceFilesContent, payload.LanguageCode, options, func(progress int, message string, metadata any, metrics models.JobMetrics) {
				updateProgress(progress, message, metadata, metrics)
			})
		default:
			toolContent, toolTitle, genErr = toolGenerator.GenerateStudyGuide(jobContext, lecture, transcriptBuilder.String(), referenceFilesContent, payload.Length, payload.LanguageCode, options, func(progress int, message string, metadata any, metrics models.JobMetrics) {
				updateProgress(progress, message, metadata, metrics)
			})
		}

		if genErr != nil {
			return fmt.Errorf("tool generation failed: %w", genErr)
		}

		// Parse citations and convert to standard footnotes
		slog.Debug("Before ParseCitations", "content_length", len(toolContent), "has_triple_braces", strings.Contains(toolContent, "{{{"))
		finalToolContent, citations := markdownReconstructor.ParseCitations(toolContent)
		slog.Debug("After ParseCitations", "citations_found", len(citations))

		// Improve footnotes using AI if it's a guide and we have citations
		if payload.Type == "guide" && len(citations) > 0 {
			updatedCitations, _, err := toolGenerator.ProcessFootnotesAI(jobContext, citations, options)
			if err == nil {
				citations = updatedCitations
			}
		}

		// If it's a guide, append the footnote definitions to the end
		if payload.Type == "guide" {
			finalToolContent = markdownReconstructor.AppendCitations(finalToolContent, citations)
		}

		toolID := uuid.New().String()

		_, err = database.Exec(`
			INSERT INTO tools (id, exam_id, type, title, content, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, toolID, payload.ExamID, payload.Type, toolTitle, finalToolContent, time.Now(), time.Now())
		if err != nil {
			return fmt.Errorf("failed to store tool: %w", err)
		}

		job.Result = fmt.Sprintf(`{"tool_id": "%s"}`, toolID)
		return nil
	})

	queue.RegisterHandler(models.JobTypePublishMaterial, func(jobContext context.Context, job *models.Job, updateProgress func(int, string, any, models.JobMetrics)) error {
		var payload struct {
			ToolID string `json:"tool_id"`
		}
		if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil {
			return fmt.Errorf("failed to unmarshal job payload: %w", err)
		}

		var tool models.Tool
		var examID string
		err := database.QueryRow("SELECT id, exam_id, type, title, content, created_at FROM tools WHERE id = ?", payload.ToolID).Scan(&tool.ID, &examID, &tool.Type, &tool.Title, &tool.Content, &tool.CreatedAt)
		if err != nil {
			return fmt.Errorf("failed to get tool: %w", err)
		}

		exportDirectory := filepath.Join(config.Storage.DataDirectory, "files", "exports", tool.ID)
		if err := os.MkdirAll(exportDirectory, 0755); err != nil {
			return fmt.Errorf("failed to create export directory: %w", err)
		}

		// Use sanitized tool title as filename
		safeFilename := sanitizeFilename(tool.Title) + ".pdf"
		pdfPath := filepath.Join(exportDirectory, safeFilename)
		slog.Info("Exporting PDF", "tool_title", tool.Title, "filename", safeFilename, "path", pdfPath)

		// 3. Prepare content for PDF (convert JSON to Markdown if needed)
		contentToConvert := tool.Content
		if tool.Type == "flashcard" || tool.Type == "quiz" {
			markdownReconstructor := markdown.NewReconstructor()
			rootNode := &markdown.Node{Type: markdown.NodeDocument}

			rootNode.Children = append(rootNode.Children, &markdown.Node{
				Type:    markdown.NodeHeading,
				Level:   1,
				Content: tool.Title,
			})

			switch tool.Type {
			case "flashcard":
				var flashcards []map[string]string
				if err := json.Unmarshal([]byte(tool.Content), &flashcards); err == nil {
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
				if err := json.Unmarshal([]byte(tool.Content), &quiz); err == nil {
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

		updateProgress(20, "Converting markdown to HTML...", nil, models.JobMetrics{})
		htmlContent, err := markdownConverter.MarkdownToHTML(contentToConvert)
		if err != nil {
			return fmt.Errorf("failed to convert to HTML: %w", err)
		}

		updateProgress(30, "Gathering lecture metadata...", nil, models.JobMetrics{})

		// Get lectures for this exam to collect metadata
		lectureRows, err := database.Query("SELECT id FROM lectures WHERE exam_id = ? AND status = 'ready'", examID)
		if err != nil {
			return fmt.Errorf("failed to query lectures: %w", err)
		}
		defer lectureRows.Close()

		var audioFiles []markdown.AudioFileMetadata
		var referenceFiles []markdown.ReferenceFileMetadata
		var transcriptBuilder strings.Builder

		for lectureRows.Next() {
			var lectureID string
			if err := lectureRows.Scan(&lectureID); err != nil {
				slog.Error("Failed to scan lecture ID", "error", err)
				continue
			}
			slog.Debug("Processing lecture for metadata", "lecture_id", lectureID)

			// Get media files (audio/video)
			mediaRows, err := database.Query("SELECT original_filename, file_path, duration_milliseconds FROM lecture_media WHERE lecture_id = ? ORDER BY sequence_order", lectureID)
			if err == nil {
				defer mediaRows.Close()
				for mediaRows.Next() {
					var originalFilename sql.NullString
					var filePath string
					var durationMs int64
					if err := mediaRows.Scan(&originalFilename, &filePath, &durationMs); err == nil {
						// Use original filename if available, otherwise extract from file_path
						filename := filepath.Base(filePath)
						if originalFilename.Valid && originalFilename.String != "" {
							filename = originalFilename.String
						}

						durationSeconds := durationMs / 1000
						slog.Debug("Found audio file", "filename", filename, "duration_seconds", durationSeconds)
						audioFiles = append(audioFiles, markdown.AudioFileMetadata{
							Filename: filename,
							Duration: durationSeconds,
						})
					}
				}
			} else {
				slog.Warn("Failed to query media files", "lecture_id", lectureID, "error", err)
			}

			// Get reference documents
			docRows, err := database.Query("SELECT title, original_filename, page_count FROM reference_documents WHERE lecture_id = ?", lectureID)
			if err == nil {
				defer docRows.Close()
				for docRows.Next() {
					var title string
					var originalFilename sql.NullString
					var pageCount int
					if err := docRows.Scan(&title, &originalFilename, &pageCount); err == nil {
						// Use original filename if available, otherwise use title
						filename := title
						if originalFilename.Valid && originalFilename.String != "" {
							filename = originalFilename.String
						}
						slog.Debug("Found reference file", "filename", filename, "pages", pageCount)
						referenceFiles = append(referenceFiles, markdown.ReferenceFileMetadata{
							Filename:  filename,
							PageCount: pageCount,
						})
					}
				}
			} else {
				slog.Warn("Failed to query documents", "lecture_id", lectureID, "error", err)
			}

			// Get transcript for abstract generation - join transcript_segments
			var transcriptID string
			err = database.QueryRow("SELECT id FROM transcripts WHERE lecture_id = ? AND status = 'completed'", lectureID).Scan(&transcriptID)
			if err == nil {
				segmentRows, err := database.Query("SELECT text FROM transcript_segments WHERE transcript_id = ? ORDER BY start_millisecond", transcriptID)
				if err == nil {
					defer segmentRows.Close()
					for segmentRows.Next() {
						var segmentText string
						if err := segmentRows.Scan(&segmentText); err == nil {
							transcriptBuilder.WriteString(segmentText)
							transcriptBuilder.WriteString(" ")
						}
					}
					slog.Debug("Found transcript", "lecture_id", lectureID, "length", transcriptBuilder.Len())
				}
			}
		}

		slog.Info("Collected metadata", "audio_files", len(audioFiles), "reference_files", len(referenceFiles), "transcript_length", transcriptBuilder.Len())
		for i, af := range audioFiles {
			slog.Debug("Audio file metadata", "index", i, "filename", af.Filename, "duration_seconds", af.Duration)
		}
		for i, rf := range referenceFiles {
			slog.Debug("Reference file metadata", "index", i, "filename", rf.Filename, "page_count", rf.PageCount)
		}

		// Generate abstract from transcript
		updateProgress(40, "Generating document abstract...", nil, models.JobMetrics{})
		abstract := ""
		if transcriptBuilder.Len() > 0 && toolGenerator != nil {
			slog.Debug("Generating abstract from transcript")
			generatedAbstract, _, err := toolGenerator.GenerateAbstract(jobContext, transcriptBuilder.String(), config.LLM.Language, "")
			if err == nil {
				abstract = generatedAbstract
				slog.Info("Generated abstract", "abstract", abstract)
			} else {
				slog.Error("Failed to generate abstract", "error", err)
			}
		} else {
			slog.Warn("Skipping abstract generation", "has_transcript", transcriptBuilder.Len() > 0, "has_generator", toolGenerator != nil)
		}

		updateProgress(50, "Generating PDF document...", nil, models.JobMetrics{})
		options := markdown.ConversionOptions{
			Language:       config.LLM.Language,
			Description:    abstract,
			CreationDate:   tool.CreatedAt,
			ReferenceFiles: referenceFiles,
			AudioFiles:     audioFiles,
		}

		slog.Info("PDF conversion options",
			"language", options.Language,
			"has_abstract", options.Description != "",
			"audio_files_count", len(options.AudioFiles),
			"reference_files_count", len(options.ReferenceFiles))

		err = markdownConverter.HTMLToPDF(htmlContent, pdfPath, options)
		if err != nil {
			return fmt.Errorf("failed to generate PDF: %w", err)
		}

		updateProgress(100, "Export completed", map[string]string{"pdf_path": pdfPath}, models.JobMetrics{})
		job.Result = fmt.Sprintf(`{"pdf_path": "%s"}`, pdfPath)
		return nil
	})
}
