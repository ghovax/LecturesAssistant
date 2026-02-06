package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
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
		temporaryDirectory := filepath.Join(config.Storage.DataDirectory, "tmp", job.ID)
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

		// 6. Finalize transcript
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

			// 3. Create output directory for pages
			outputDirectory := filepath.Join(config.Storage.DataDirectory, "files", "lectures", payload.LectureID, "documents", document.ID)

			// 4. Run document processing
			pages, err := documentProcessor.ProcessDocument(jobContext, document, outputDirectory, payload.LanguageCode, func(progress int, message string) {
				updateProgress(progress, "Extracting and processing document pages...", metadata, models.JobMetrics{})
			})
			if err != nil {
				database.Exec("UPDATE reference_documents SET extraction_status = ?, updated_at = ? WHERE id = ?", "failed", time.Now(), document.ID)
				database.Exec("UPDATE lectures SET status = ?, updated_at = ? WHERE id = ?", "failed", time.Now(), payload.LectureID)
				return fmt.Errorf("document processor failed for %s: %w", document.Title, err)
			}

			// 5. Store pages in database
			databaseTransaction, err := database.Begin()
			if err != nil {
				return fmt.Errorf("failed to begin transaction: %w", err)
			}
			defer databaseTransaction.Rollback()

			// Delete existing pages if any
			_, err = databaseTransaction.Exec("DELETE FROM reference_pages WHERE document_id = ?", document.ID)
			if err != nil {
				return fmt.Errorf("failed to delete old pages: %w", err)
			}

			for _, currentPage := range pages {
				_, err = databaseTransaction.Exec(`
					INSERT INTO reference_pages (document_id, page_number, image_path, extracted_text)
					VALUES (?, ?, ?, ?)
				`, document.ID, currentPage.PageNumber, currentPage.ImagePath, currentPage.ExtractedText)
				if err != nil {
					return fmt.Errorf("failed to insert page: %w", err)
				}
			}

			// 6. Update document as completed
			_, err = databaseTransaction.Exec("UPDATE reference_documents SET extraction_status = ?, page_count = ?, updated_at = ? WHERE id = ?", "completed", len(pages), time.Now(), document.ID)
			if err != nil {
				return fmt.Errorf("failed to finalize document status: %w", err)
			}

			if err := databaseTransaction.Commit(); err != nil {
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
			LectureID           string `json:"lecture_id"`
			ExamID              string `json:"exam_id"`
			Type                string `json:"type"`
			Length              string `json:"length"`
			LanguageCode        string `json:"language_code"`
						EnableTriangulation string `json:"enable_triangulation"`
						AdherenceThreshold  string `json:"adherence_threshold"`
						MaximumRetries      string `json:"maximum_retries"`
						// Models
						ModelTriangulation string `json:"model_triangulation"`
						ModelStructure     string `json:"model_structure"`
						ModelGeneration    string `json:"model_generation"`
						ModelAdherence     string `json:"model_adherence"`
						ModelPolishing     string `json:"model_polishing"`
					}
					if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil {
						return fmt.Errorf("failed to unmarshal job payload: %w", err)
					}
			
					threshold, _ := strconv.Atoi(payload.AdherenceThreshold)
					maximumRetries, _ := strconv.Atoi(payload.MaximumRetries)
					options := models.GenerationOptions{
						EnableTriangulation: payload.EnableTriangulation == "true",
						AdherenceThreshold:  threshold,
						MaximumRetries:      maximumRetries,
						ModelTriangulation:  payload.ModelTriangulation,
						ModelStructure:      payload.ModelStructure,
						ModelGeneration:     payload.ModelGeneration,
						ModelAdherence:      payload.ModelAdherence,
						ModelPolishing:      payload.ModelPolishing,
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
		finalToolContent, citations := markdownReconstructor.ParseCitations(toolContent)

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
		err := database.QueryRow("SELECT id, type, title, content, created_at FROM tools WHERE id = ?", payload.ToolID).Scan(&tool.ID, &tool.Type, &tool.Title, &tool.Content, &tool.CreatedAt)
		if err != nil {
			return fmt.Errorf("failed to get tool: %w", err)
		}

		exportDirectory := filepath.Join(config.Storage.DataDirectory, "files", "exports", tool.ID)
		if err := os.MkdirAll(exportDirectory, 0755); err != nil {
			return fmt.Errorf("failed to create export directory: %w", err)
		}
		pdfPath := filepath.Join(exportDirectory, "export.pdf")

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

		updateProgress(50, "Generating PDF document...", nil, models.JobMetrics{})
		options := markdown.ConversionOptions{
			Language:     config.LLM.Language,
			CreationDate: tool.CreatedAt,
		}

		err = markdownConverter.HTMLToPDF(htmlContent, pdfPath, options)
		if err != nil {
			return fmt.Errorf("failed to generate PDF: %w", err)
		}

		updateProgress(100, "Export completed", map[string]string{"pdf_path": pdfPath}, models.JobMetrics{})
		job.Result = fmt.Sprintf(`{"pdf_path": "%s"}`, pdfPath)
		return nil
	})
}
