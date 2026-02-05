package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"lectures/internal/api"
	"lectures/internal/configuration"
	"lectures/internal/database"
	"lectures/internal/documents"
	"lectures/internal/jobs"
	"lectures/internal/llm"
	"lectures/internal/markdown"
	"lectures/internal/models"
	"lectures/internal/prompts"
	"lectures/internal/tools"
	"lectures/internal/transcription"
)

func main() {
	// Parse command-line flags
	configurationPath := flag.String("configuration", "", "Path to configuration file")
	flag.Parse()

	// Load configuration
	loadedConfiguration, err := configuration.Load(*configurationPath)
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

	// Initialize prompt manager
	promptManager := prompts.NewManager("prompts")

	// Initialize LLM provider
	var llmProvider llm.Provider
	var defaultLLMModel string

	switch loadedConfiguration.LLM.Provider {
	case "openrouter":
		llmProvider = llm.NewOpenRouterProvider(loadedConfiguration.LLM.OpenRouter.APIKey)
		defaultLLMModel = loadedConfiguration.LLM.OpenRouter.DefaultModel
	case "ollama":
		llmProvider = llm.NewOllamaProvider(loadedConfiguration.LLM.Ollama.BaseURL)
		defaultLLMModel = loadedConfiguration.LLM.Ollama.DefaultModel
	default:
		slog.Warn("Unknown LLM provider, falling back to openrouter with empty key", "provider", loadedConfiguration.LLM.Provider)
		llmProvider = llm.NewOpenRouterProvider("")
		defaultLLMModel = loadedConfiguration.LLM.OpenRouter.DefaultModel
	}

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

	transcriptionService := transcription.NewService(loadedConfiguration, transcriptionProvider, llmProvider, promptManager)

	// Initialize document processor
	documentProcessor := documents.NewProcessor(llmProvider, defaultLLMModel, promptManager)

	// Initialize markdown converter
	markdownConverter := markdown.NewConverter(loadedConfiguration.Storage.DataDirectory)

	// Check dependencies
	if err := transcriptionService.CheckDependencies(); err != nil {
		slog.Error("Transcription dependencies check failed", "error", err)
		os.Exit(1)
	}
	if err := documentProcessor.CheckDependencies(); err != nil {
		slog.Error("Document processor dependencies check failed", "error", err)
		os.Exit(1)
	}
	if err := markdownConverter.CheckDependencies(); err != nil {
		slog.Warn("Markdown converter dependencies check failed (PDF export may not work)", "error", err)
	}

	// Initialize tool generator
	toolGenerator := tools.NewToolGenerator(loadedConfiguration, llmProvider, promptManager)

	// Initialize job queue
	backgroundJobQueue := jobs.NewQueue(initializedDatabase, 4) // 4 concurrent workers

	// Register job handlers
	backgroundJobQueue.RegisterHandler(models.JobTypeTranscribeMedia, func(jobContext context.Context, job *models.Job, updateProgress func(int, string, any, jobs.JobMetrics)) error {
		var payload struct {
			LectureID string `json:"lecture_id"`
		}
		if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil {
			return fmt.Errorf("failed to unmarshal job payload: %w", err)
		}

		// 1. Get lecture media files in order
		mediaRows, databaseError := initializedDatabase.Query(`
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
		segments, err := transcriptionService.TranscribeLecture(jobContext, mediaFiles, temporaryDirectory, func(progress int, message string, metadata any) {
			updateProgress(progress, "Transcribing media files...", metadata, jobs.JobMetrics{})
		})
		if err != nil {
			initializedDatabase.Exec("UPDATE transcripts SET status = ?, updated_at = ? WHERE id = ?", "failed", time.Now(), transcriptID)
			initializedDatabase.Exec("UPDATE lectures SET status = ?, updated_at = ? WHERE id = ?", "failed", time.Now(), payload.LectureID)
			return fmt.Errorf("transcription service failed: %w", err)
		}

		// 5. Store segments in database
		databaseTransaction, err := initializedDatabase.Begin()
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

		checkLectureReadiness(initializedDatabase, payload.LectureID)
		updateProgress(100, "Transcription completed", nil, jobs.JobMetrics{})
		return nil
	})

	backgroundJobQueue.RegisterHandler(models.JobTypeIngestDocuments, func(jobContext context.Context, job *models.Job, updateProgress func(int, string, any, jobs.JobMetrics)) error {
		var payload struct {
			LectureID    string `json:"lecture_id"`
			LanguageCode string `json:"language_code"`
		}
		if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil {
			return fmt.Errorf("failed to unmarshal job payload: %w", err)
		}

		if payload.LanguageCode == "" {
			payload.LanguageCode = loadedConfiguration.LLM.Language
		}

		// 1. Get reference documents for the lecture
		documentRows, databaseError := initializedDatabase.Query(`
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
			updateProgress(int(float64(documentIndex)/float64(totalDocuments)*100), "Ingesting reference documents...", metadata, jobs.JobMetrics{})

			// 2. Update status to processing
			_, err = initializedDatabase.Exec("UPDATE reference_documents SET extraction_status = ?, updated_at = ? WHERE id = ?", "processing", time.Now(), document.ID)
			if err != nil {
				return fmt.Errorf("failed to update document status: %w", err)
			}

			// 3. Create output directory for pages
			outputDirectory := filepath.Join(loadedConfiguration.Storage.DataDirectory, "files", "lectures", payload.LectureID, "documents", document.ID)

			// 4. Run document processing
			pages, err := documentProcessor.ProcessDocument(jobContext, document, outputDirectory, payload.LanguageCode, func(progress int, message string) {
				updateProgress(progress, "Extracting and processing document pages...", metadata, jobs.JobMetrics{})
			})
			if err != nil {
				initializedDatabase.Exec("UPDATE reference_documents SET extraction_status = ?, updated_at = ? WHERE id = ?", "failed", time.Now(), document.ID)
				initializedDatabase.Exec("UPDATE lectures SET status = ?, updated_at = ? WHERE id = ?", "failed", time.Now(), payload.LectureID)
				return fmt.Errorf("document processor failed for %s: %w", document.Title, err)
			}

			// 5. Store pages in database
			databaseTransaction, err := initializedDatabase.Begin()
			if err != nil {
				return fmt.Errorf("failed to begin transaction: %w", err)
			}
			defer databaseTransaction.Rollback()

			// Delete existing pages if any
			_, err = databaseTransaction.Exec("DELETE FROM reference_pages WHERE document_id = ?", document.ID)
			if err != nil {
				return fmt.Errorf("failed to delete old pages: %w", err)
			}

			for _, page := range pages {
				_, err = databaseTransaction.Exec(`
					INSERT INTO reference_pages (document_id, page_number, image_path, extracted_text)
					VALUES (?, ?, ?, ?)
				`, document.ID, page.PageNumber, page.ImagePath, page.ExtractedText)
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

		checkLectureReadiness(initializedDatabase, payload.LectureID)
		updateProgress(100, "Document ingestion completed", nil, jobs.JobMetrics{})
		return nil
	})

	backgroundJobQueue.RegisterHandler(models.JobTypeBuildMaterial, func(jobContext context.Context, job *models.Job, updateProgress func(int, string, any, jobs.JobMetrics)) error {
		var payload struct {
			LectureID    string `json:"lecture_id"`
			ExamID       string `json:"exam_id"`
			Type         string `json:"type"`
			Length       string `json:"length"`
			LanguageCode string `json:"language_code"`
		}
		if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil {
			return fmt.Errorf("failed to unmarshal job payload: %w", err)
		}

		if payload.Type == "" {
			payload.Type = "guide"
		}

		var lecture models.Lecture
		err := initializedDatabase.QueryRow("SELECT id, exam_id, title, description FROM lectures WHERE id = ?", payload.LectureID).Scan(&lecture.ID, &lecture.ExamID, &lecture.Title, &lecture.Description)
		if err != nil {
			return fmt.Errorf("failed to get lecture: %w", err)
		}

		transcriptRows, databaseError := initializedDatabase.Query(`
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

		documentRows, databaseError := initializedDatabase.Query(`
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
			toolContent, toolTitle, genErr = toolGenerator.GenerateFlashcards(jobContext, lecture, transcriptBuilder.String(), referenceFilesContent, payload.LanguageCode, updateProgress)
		case "quiz":
			toolContent, toolTitle, genErr = toolGenerator.GenerateQuiz(jobContext, lecture, transcriptBuilder.String(), referenceFilesContent, payload.LanguageCode, updateProgress)
		default:
			toolContent, toolTitle, genErr = toolGenerator.GenerateStudyGuide(jobContext, lecture, transcriptBuilder.String(), referenceFilesContent, payload.Length, payload.LanguageCode, updateProgress)
		}

		if genErr != nil {
			return fmt.Errorf("tool generation failed: %w", genErr)
		}

		// Parse citations and convert to standard footnotes

		finalToolContent, citations := markdownReconstructor.ParseCitations(toolContent)

		// If it's a guide, append the footnote definitions to the end

		if payload.Type == "guide" {

			finalToolContent = markdownReconstructor.AppendCitations(finalToolContent, citations)

		}

		toolID := uuid.New().String()

		_, err = initializedDatabase.Exec(`
			INSERT INTO tools (id, exam_id, type, title, content, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, toolID, payload.ExamID, payload.Type, toolTitle, finalToolContent, time.Now(), time.Now())
		if err != nil {
			return fmt.Errorf("failed to store tool: %w", err)
		}

		job.Result = fmt.Sprintf(`{"tool_id": "%s"}`, toolID)
		return nil
	})

	backgroundJobQueue.RegisterHandler(models.JobTypePublishMaterial, func(jobContext context.Context, job *models.Job, updateProgress func(int, string, any, jobs.JobMetrics)) error {
		var payload struct {
			ToolID string `json:"tool_id"`
		}
		if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil {
			return fmt.Errorf("failed to unmarshal job payload: %w", err)
		}

		var tool models.Tool
		err := initializedDatabase.QueryRow("SELECT id, type, title, content, created_at FROM tools WHERE id = ?", payload.ToolID).Scan(&tool.ID, &tool.Type, &tool.Title, &tool.Content, &tool.CreatedAt)
		if err != nil {
			return fmt.Errorf("failed to get tool: %w", err)
		}

		exportDirectory := filepath.Join(loadedConfiguration.Storage.DataDirectory, "files", "exports", tool.ID)
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

		updateProgress(20, "Converting markdown to HTML...", nil, jobs.JobMetrics{})
		htmlContent, err := markdownConverter.MarkdownToHTML(contentToConvert)
		if err != nil {
			return fmt.Errorf("failed to convert to HTML: %w", err)
		}

		updateProgress(50, "Generating PDF document...", nil, jobs.JobMetrics{})
		options := markdown.ConversionOptions{
			Language:     loadedConfiguration.LLM.Language,
			CreationDate: tool.CreatedAt,
		}

		err = markdownConverter.HTMLToPDF(htmlContent, pdfPath, options)
		if err != nil {
			return fmt.Errorf("failed to generate PDF: %w", err)
		}

		updateProgress(100, "Export completed", map[string]string{"pdf_path": pdfPath}, jobs.JobMetrics{})
		job.Result = fmt.Sprintf(`{"pdf_path": "%s"}`, pdfPath)
		return nil
	})

	backgroundJobQueue.Start()
	defer backgroundJobQueue.Stop()

	// Create API server
	apiServer := api.NewServer(loadedConfiguration, initializedDatabase, backgroundJobQueue, llmProvider, promptManager)

	// Start HTTP server
	serverAddress := fmt.Sprintf("%s:%d", loadedConfiguration.Server.Host, loadedConfiguration.Server.Port)
	slog.Info("Server starting", "address", serverAddress)
	slog.Info("Data directory", "directory", loadedConfiguration.Storage.DataDirectory)

	if err := http.ListenAndServe(serverAddress, apiServer.Handler()); err != nil {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
}

func checkLectureReadiness(database *sql.DB, lectureID string) {
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
