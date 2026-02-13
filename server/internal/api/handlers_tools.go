package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"lectures/internal/markdown"
	"lectures/internal/models"
)

// BCP-47 Regex (basic validation)
var bcp47Regex = regexp.MustCompile(`^[a-zA-Z]{2,3}(?:-[a-zA-Z]{4})?(?:-[a-zA-Z]{2}|-[0-9]{3})?$`)

// handleCreateTool triggers a tool generation job
func (server *Server) handleCreateTool(responseWriter http.ResponseWriter, request *http.Request) {
	var createToolRequest struct {
		ExamID                  string `json:"exam_id"`
		LectureID               string `json:"lecture_id"`
		Type                    string `json:"type"` // "guide", "flashcard", "quiz"
		Length                  string `json:"length"`
		LanguageCode            string `json:"language_code"`
		EnableDocumentsMatching *bool  `json:"enable_documents_matching"`
		AdherenceThreshold      int    `json:"adherence_threshold"`
		MaximumRetries          int    `json:"maximum_retries"`
		// Models
		ModelDocumentsMatching string `json:"model_documents_matching"`
		ModelStructure         string `json:"model_structure"`
		ModelGeneration        string `json:"model_generation"`
		ModelAdherence         string `json:"model_adherence"`
		ModelPolishing         string `json:"model_polishing"`
	}

	if err := json.NewDecoder(request.Body).Decode(&createToolRequest); err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	if createToolRequest.ExamID == "" || createToolRequest.LectureID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "exam_id and lecture_id are required", nil)
		return
	}

	// Verify exam and lecture exist
	var lecture models.Lecture
	err := server.database.QueryRow("SELECT id, status FROM lectures WHERE id = ? AND exam_id = ?", createToolRequest.LectureID, createToolRequest.ExamID).Scan(&lecture.ID, &lecture.Status)
	if err != nil {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Lecture not found in this exam", nil)
		return
	}

	if lecture.Status != "ready" {
		server.writeError(responseWriter, http.StatusConflict, "LECTURE_NOT_READY", fmt.Sprintf("Lecture is currently in status: %s. Please wait for processing to complete.", lecture.Status), nil)
		return
	}

	// Default values
	if createToolRequest.Type == "" {
		createToolRequest.Type = "guide"
	}
	if createToolRequest.Length == "" {
		createToolRequest.Length = "medium"
	}
	if createToolRequest.LanguageCode == "" {
		createToolRequest.LanguageCode = server.configuration.LLM.Language
	}

	enableMatching := server.configuration.LLM.EnableDocumentsMatching
	if createToolRequest.EnableDocumentsMatching != nil {
		enableMatching = *createToolRequest.EnableDocumentsMatching
	}

	// Validate BCP-47 language code
	if !bcp47Regex.MatchString(createToolRequest.LanguageCode) {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid language_code format (BCP-47 required)", nil)
		return
	}

	userID := server.getUserID(request)

	// Enforce "one of each type" by deleting existing tool of the same type
	_, _ = server.database.Exec(`
		DELETE FROM tools 
		WHERE lecture_id = ? AND type = ? AND EXISTS (
			SELECT 1 FROM exams WHERE id = ? AND user_id = ?
		)
	`, createToolRequest.LectureID, createToolRequest.Type, createToolRequest.ExamID, userID)

	// Enqueue job
	jobIdentifier, err := server.jobQueue.Enqueue(userID, models.JobTypeBuildMaterial, map[string]string{
		"exam_id":                   createToolRequest.ExamID,
		"lecture_id":                createToolRequest.LectureID,
		"type":                      createToolRequest.Type,
		"length":                    createToolRequest.Length,
		"language_code":             createToolRequest.LanguageCode,
		"enable_documents_matching": fmt.Sprintf("%v", enableMatching),
		"adherence_threshold":       fmt.Sprintf("%d", createToolRequest.AdherenceThreshold),
		"maximum_retries":           fmt.Sprintf("%d", createToolRequest.MaximumRetries),
		"model_documents_matching":  createToolRequest.ModelDocumentsMatching,
		"model_structure":           createToolRequest.ModelStructure,
		"model_generation":          createToolRequest.ModelGeneration,
		"model_adherence":           createToolRequest.ModelAdherence,
		"model_polishing":           createToolRequest.ModelPolishing,
	}, createToolRequest.ExamID, createToolRequest.LectureID)

	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "BACKGROUND_JOB_ERROR", "Failed to create generation job", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusAccepted, map[string]string{
		"job_id":  jobIdentifier,
		"message": "Generation job created",
	})
}

// handleListTools lists all tools for an exam or lecture (must belong to the user)
func (server *Server) handleListTools(responseWriter http.ResponseWriter, request *http.Request) {
	examID := request.URL.Query().Get("exam_id")
	lectureID := request.URL.Query().Get("lecture_id")

	if examID == "" && lectureID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "exam_id or lecture_id is required", nil)
		return
	}

	userID := server.getUserID(request)
	toolType := request.URL.Query().Get("type")

	query := `
		SELECT tools.id, tools.exam_id, tools.lecture_id, tools.type, tools.title, tools.language_code, tools.created_at, tools.updated_at
		FROM tools
		JOIN exams ON tools.exam_id = exams.id
		WHERE exams.user_id = ?
	`
	arguments := []any{userID}

	if examID != "" {
		query += " AND tools.exam_id = ?"
		arguments = append(arguments, examID)
	}
	if lectureID != "" {
		query += " AND tools.lecture_id = ?"
		arguments = append(arguments, lectureID)
	}

	if toolType != "" {
		query += " AND tools.type = ?"
		arguments = append(arguments, toolType)
	}

	query += " ORDER BY tools.created_at DESC"

	toolRows, databaseError := server.database.Query(query, arguments...)
	if databaseError != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to list tools", nil)
		return
	}
	defer toolRows.Close()

	var toolsList = []models.Tool{}
	for toolRows.Next() {
		var tool models.Tool
		var lID sql.NullString
		if err := toolRows.Scan(&tool.ID, &tool.ExamID, &lID, &tool.Type, &tool.Title, &tool.LanguageCode, &tool.CreatedAt, &tool.UpdatedAt); err != nil {
			continue
		}
		if lID.Valid {
			tool.LectureID = lID.String
		}
		toolsList = append(toolsList, tool)
	}

	server.writeJSON(responseWriter, http.StatusOK, toolsList)
}

// handleGetTool retrieves a specific tool
func (server *Server) handleGetTool(responseWriter http.ResponseWriter, request *http.Request) {
	toolID := request.URL.Query().Get("tool_id")
	examID := request.URL.Query().Get("exam_id")

	if toolID == "" || examID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "tool_id and exam_id are required", nil)
		return
	}

	userID := server.getUserID(request)

	var tool models.Tool
	err := server.database.QueryRow(`
		SELECT tools.id, tools.exam_id, tools.type, tools.title, tools.language_code, tools.content, tools.created_at, tools.updated_at
		FROM tools
		JOIN exams ON tools.exam_id = exams.id
		WHERE tools.id = ? AND tools.exam_id = ? AND exams.user_id = ?
	`, toolID, examID, userID).Scan(&tool.ID, &tool.ExamID, &tool.Type, &tool.Title, &tool.LanguageCode, &tool.Content, &tool.CreatedAt, &tool.UpdatedAt)

	if err == sql.ErrNoRows {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Tool not found in this exam", nil)
		return
	}
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get tool", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusOK, tool)
}

// handleUpdateTool allows manual refinement of tool content or title
func (server *Server) handleUpdateTool(responseWriter http.ResponseWriter, request *http.Request) {
	var updateRequest struct {
		ToolID  string  `json:"tool_id"`
		ExamID  string  `json:"exam_id"`
		Title   *string `json:"title"`
		Content *string `json:"content"`
	}

	if err := json.NewDecoder(request.Body).Decode(&updateRequest); err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	if updateRequest.ToolID == "" || updateRequest.ExamID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "tool_id and exam_id are required", nil)
		return
	}

	userID := server.getUserID(request)

	// Verify ownership
	var exists bool
	err := server.database.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM tools 
			JOIN exams ON tools.exam_id = exams.id
			WHERE tools.id = ? AND tools.exam_id = ? AND exams.user_id = ?
		)
	`, updateRequest.ToolID, updateRequest.ExamID, userID).Scan(&exists)

	if err != nil || !exists {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Tool not found in this exam", nil)
		return
	}

	query := "UPDATE tools SET updated_at = ?"
	args := []any{time.Now()}

	if updateRequest.Title != nil {
		query += ", title = ?"
		args = append(args, *updateRequest.Title)
	}
	if updateRequest.Content != nil {
		query += ", content = ?"
		args = append(args, *updateRequest.Content)
	}

	query += " WHERE id = ?"
	args = append(args, updateRequest.ToolID)

	_, err = server.database.Exec(query, args...)
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to update tool", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusOK, map[string]string{"message": "Tool updated successfully"})
}

// handleGetToolHTML retrieves a specific tool and converts its content to HTML
func (server *Server) handleGetToolHTML(responseWriter http.ResponseWriter, request *http.Request) {
	toolID := request.URL.Query().Get("tool_id")
	examID := request.URL.Query().Get("exam_id")

	if toolID == "" || examID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "tool_id and exam_id are required", nil)
		return
	}

	userID := server.getUserID(request)

	var tool models.Tool
	err := server.database.QueryRow(`
		SELECT tools.id, tools.title, tools.type, tools.content
		FROM tools
		JOIN exams ON tools.exam_id = exams.id
		WHERE tools.id = ? AND tools.exam_id = ? AND exams.user_id = ?
	`, toolID, examID, userID).Scan(&tool.ID, &tool.Title, &tool.Type, &tool.Content)

	if err == sql.ErrNoRows {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Tool not found", nil)
		return
	}
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get tool", nil)
		return
	}

	// For flashcards and quizzes, we return structured data with HTML fields
	if tool.Type == "flashcard" {
		var flashcards []map[string]string
		content := tool.Content
		// Attempt robust extraction if direct parse fails
		if err := json.Unmarshal([]byte(content), &flashcards); err != nil {
			// Try to extract from Markdown fences
			start := strings.Index(content, "[")
			end := strings.LastIndex(content, "]")
			if start != -1 && end != -1 && end > start {
				if err := json.Unmarshal([]byte(content[start:end+1]), &flashcards); err != nil {
					server.writeError(responseWriter, http.StatusInternalServerError, "JSON_ERROR", "Failed to parse flashcards after extraction", nil)
					return
				}
			} else {
				server.writeError(responseWriter, http.StatusInternalServerError, "JSON_ERROR", "Failed to parse flashcards", nil)
				return
			}
		}

		type flashcardHTML struct {
			FrontHTML string `json:"front_html"`
			BackHTML  string `json:"back_html"`
		}
		var result []flashcardHTML

		for _, fc := range flashcards {
			frontHTML, _ := server.markdownConverter.MarkdownToHTML(fc["front"])
			backHTML, _ := server.markdownConverter.MarkdownToHTML(fc["back"])
			result = append(result, flashcardHTML{
				FrontHTML: frontHTML,
				BackHTML:  backHTML,
			})
		}

		server.writeJSON(responseWriter, http.StatusOK, map[string]any{
			"tool_id": tool.ID,
			"title":   tool.Title,
			"type":    tool.Type,
			"content": result,
		})
		return
	}

	if tool.Type == "quiz" {
		var quizItems []map[string]any
		content := tool.Content
		if err := json.Unmarshal([]byte(content), &quizItems); err != nil {
			start := strings.Index(content, "[")
			end := strings.LastIndex(content, "]")
			if start != -1 && end != -1 && end > start {
				if err := json.Unmarshal([]byte(content[start:end+1]), &quizItems); err != nil {
					server.writeError(responseWriter, http.StatusInternalServerError, "JSON_ERROR", "Failed to parse quiz after extraction", nil)
					return
				}
			} else {
				server.writeError(responseWriter, http.StatusInternalServerError, "JSON_ERROR", "Failed to parse quiz", nil)
				return
			}
		}

		type quizItemHTML struct {
			QuestionHTML      string   `json:"question_html"`
			OptionsHTML       []string `json:"options_html"`
			CorrectAnswerHTML string   `json:"correct_answer_html"`
			ExplanationHTML   string   `json:"explanation_html"`
		}
		var result []quizItemHTML

		for _, item := range quizItems {
			questionHTML, _ := server.markdownConverter.MarkdownToHTML(fmt.Sprintf("%v", item["question"]))
			explanationHTML, _ := server.markdownConverter.MarkdownToHTML(fmt.Sprintf("%v", item["explanation"]))
			correctAnswerHTML, _ := server.markdownConverter.MarkdownToHTML(fmt.Sprintf("%v", item["correct_answer"]))

			var optionsHTML []string
			if options, ok := item["options"].([]any); ok {
				for _, opt := range options {
					oHTML, _ := server.markdownConverter.MarkdownToHTML(fmt.Sprintf("%v", opt))
					optionsHTML = append(optionsHTML, oHTML)
				}
			}

			result = append(result, quizItemHTML{
				QuestionHTML:      questionHTML,
				OptionsHTML:       optionsHTML,
				CorrectAnswerHTML: correctAnswerHTML,
				ExplanationHTML:   explanationHTML,
			})
		}

		server.writeJSON(responseWriter, http.StatusOK, map[string]any{
			"tool_id": tool.ID,
			"title":   tool.Title,
			"type":    tool.Type,
			"content": result,
		})
		return
	}

	// For guide (study guide), it's already Markdown, return structured data with HTML
	markdownText := tool.Content

	// For study guides, transform raw citations to footnotes at runtime
	if tool.Type == "guide" {
		markdownReconstructor := markdown.NewReconstructor()
		markdownReconstructor.Language = tool.LanguageCode

		// 1. Convert triple braces to references
		processedContent, textCitations := markdownReconstructor.ParseCitations(markdownText)

		// 2. Fetch improved citations from database
		rows, err := server.database.Query("SELECT metadata FROM tool_source_references WHERE tool_id = ?", tool.ID)
		if err == nil {
			improvedCitations := make(map[int]string)
			for rows.Next() {
				var metadataJSON string
				if err := rows.Scan(&metadataJSON); err == nil {
					var meta struct {
						FootnoteNumber int    `json:"footnote_number"`
						Description    string `json:"description"`
					}
					if json.Unmarshal([]byte(metadataJSON), &meta) == nil {
						improvedCitations[meta.FootnoteNumber] = meta.Description
					}
				}
			}
			rows.Close()

			// 3. Apply improved descriptions
			for i := range textCitations {
				if desc, ok := improvedCitations[textCitations[i].Number]; ok {
					textCitations[i].Description = desc
				}
			}
		}

		// 4. Finalize markdown with footnote definitions
		markdownText = markdownReconstructor.AppendCitations(processedContent, textCitations)
	}

	// Strip top-level title (H1) if present to follow "simply the content" requirement
	// Study guides generated by our pipeline start with a Level 1 heading.
	lines := strings.Split(markdownText, "\n")
	var contentLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			continue // Skip top-level title
		}
		contentLines = append(contentLines, line)
	}
	markdownWithoutTitle := strings.TrimSpace(strings.Join(contentLines, "\n"))

	htmlContent, err := server.markdownConverter.MarkdownToHTML(markdownWithoutTitle)
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "CONVERSION_ERROR", "Failed to convert tool to HTML", nil)
		return
	}

	// Fetch structured citations
	type citationMeta struct {
		Number      int    `json:"number"`
		ContentHTML string `json:"content_html"`
		SourceFile  string `json:"source_file"`
		SourcePages []int  `json:"source_pages"`
	}
	var citations []citationMeta

	rows, err := server.database.Query(`
		SELECT source_id, metadata FROM tool_source_references WHERE tool_id = ?
	`, tool.ID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var sourceID, metadataJSON string
			if err := rows.Scan(&sourceID, &metadataJSON); err == nil {
				var meta struct {
					FootnoteNumber int    `json:"footnote_number"`
					Description    string `json:"description"`
					Pages          []int  `json:"pages"`
				}
				if json.Unmarshal([]byte(metadataJSON), &meta) == nil {
					// Convert citation description to HTML to handle LaTeX/Markdown
					citationHTML, _ := server.markdownConverter.MarkdownToHTML(meta.Description)

					citations = append(citations, citationMeta{
						Number:      meta.FootnoteNumber,
						ContentHTML: citationHTML,
						SourceFile:  sourceID,
						SourcePages: meta.Pages,
					})
				}
			}
		}
	}

	server.writeJSON(responseWriter, http.StatusOK, map[string]any{
		"tool_id":      tool.ID,
		"title":        tool.Title,
		"type":         tool.Type,
		"content_html": htmlContent,
		"citations":    citations,
	})
}

// handleDeleteTool deletes a specific tool
func (server *Server) handleDeleteTool(responseWriter http.ResponseWriter, request *http.Request) {
	var deleteRequest struct {
		ToolID string `json:"tool_id"`
		ExamID string `json:"exam_id"`
	}
	if err := json.NewDecoder(request.Body).Decode(&deleteRequest); err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid body", nil)
		return
	}

	if deleteRequest.ToolID == "" || deleteRequest.ExamID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "tool_id and exam_id are required", nil)
		return
	}

	userID := server.getUserID(request)

	result, err := server.database.Exec(`
		DELETE FROM tools
		WHERE id = ? AND exam_id = ? AND EXISTS (
			SELECT 1 FROM exams WHERE id = ? AND user_id = ?
		)
	`, deleteRequest.ToolID, deleteRequest.ExamID, deleteRequest.ExamID, userID)
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to delete tool", nil)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Tool not found in this exam", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusOK, map[string]string{"message": "Tool deleted successfully"})
}

// handleExportTool triggers an export job for a specific tool (PDF, Docx, MD)
func (server *Server) handleExportTool(responseWriter http.ResponseWriter, request *http.Request) {
	var exportRequest struct {
		ToolID        string `json:"tool_id"`
		ExamID        string `json:"exam_id"`
		Format        string `json:"format"` // "pdf", "docx", "md"
		IncludeImages *bool  `json:"include_images"`
		IncludeQRCode *bool  `json:"include_qr_code"`
	}

	if decodingError := json.NewDecoder(request.Body).Decode(&exportRequest); decodingError != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	if exportRequest.ToolID == "" || exportRequest.ExamID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "tool_id and exam_id are required", nil)
		return
	}

	if exportRequest.Format == "" {
		exportRequest.Format = "pdf"
	}

	includeImages := true
	if exportRequest.IncludeImages != nil {
		includeImages = *exportRequest.IncludeImages
	}

	includeQRCode := false
	if exportRequest.IncludeQRCode != nil {
		includeQRCode = *exportRequest.IncludeQRCode
	}

	userID := server.getUserID(request)

	// Verify tool exists and belongs to the user
	var toolID string
	var languageCode sql.NullString
	var lectureID sql.NullString
	queryError := server.database.QueryRow(`
		SELECT tools.id, tools.language_code, tools.lecture_id
		FROM tools
		JOIN exams ON tools.exam_id = exams.id
		WHERE tools.id = ? AND tools.exam_id = ? AND exams.user_id = ?
	`, exportRequest.ToolID, exportRequest.ExamID, userID).Scan(&toolID, &languageCode, &lectureID)

	if queryError == sql.ErrNoRows {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Tool not found in this exam", nil)
		return
	}
	if queryError != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to verify tool", nil)
		return
	}

	lang := ""
	if languageCode.Valid {
		lang = languageCode.String
	}

	// Enqueue export job
	jobIdentifier, enqueuingError := server.jobQueue.Enqueue(userID, models.JobTypePublishMaterial, map[string]string{
		"tool_id":         exportRequest.ToolID,
		"language_code":   lang,
		"format":          exportRequest.Format,
		"include_images":  fmt.Sprintf("%v", includeImages),
		"include_qr_code": fmt.Sprintf("%v", includeQRCode),
	}, exportRequest.ExamID, lectureID.String)

	if enqueuingError != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "BACKGROUND_JOB_ERROR", "Failed to create export job", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusAccepted, map[string]string{
		"job_id":  jobIdentifier,
		"message": "Export job created",
	})
}

// handleExportTranscript triggers an export job for a lecture transcript
func (server *Server) handleExportTranscript(responseWriter http.ResponseWriter, request *http.Request) {
	var exportRequest struct {
		LectureID string `json:"lecture_id"`
		ExamID    string `json:"exam_id"`
		Format    string `json:"format"` // "pdf", "docx", "md"
	}

	if decodingError := json.NewDecoder(request.Body).Decode(&exportRequest); decodingError != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	if exportRequest.LectureID == "" || exportRequest.ExamID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "lecture_id and exam_id are required", nil)
		return
	}

	if exportRequest.Format == "" {
		exportRequest.Format = "pdf"
	}

	userID := server.getUserID(request)

	// Verify lecture exists and belongs to the user
	var lectureTitle string
	var languageCode sql.NullString
	queryError := server.database.QueryRow(`
		SELECT lectures.title, lectures.language
		FROM lectures
		JOIN exams ON lectures.exam_id = exams.id
		WHERE lectures.id = ? AND lectures.exam_id = ? AND exams.user_id = ?
	`, exportRequest.LectureID, exportRequest.ExamID, userID).Scan(&lectureTitle, &languageCode)

	if queryError == sql.ErrNoRows {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Lecture not found in this exam", nil)
		return
	}
	if queryError != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to verify lecture", nil)
		return
	}

	lang := ""
	if languageCode.Valid {
		lang = languageCode.String
	}

	if lang == "" {
		lang = server.configuration.LLM.Language
	}

	// Enqueue export job
	jobIdentifier, enqueuingError := server.jobQueue.Enqueue(userID, models.JobTypePublishMaterial, map[string]string{
		"lecture_id":    exportRequest.LectureID,
		"language_code": lang,
		"format":        exportRequest.Format,
	}, exportRequest.ExamID, exportRequest.LectureID)

	if enqueuingError != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "BACKGROUND_JOB_ERROR", "Failed to create export job", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusAccepted, map[string]string{
		"job_id":  jobIdentifier,
		"message": "Transcript export job created",
	})
}

// handleExportDocument triggers an export job for a reference document (images + interpretations)
func (server *Server) handleExportDocument(responseWriter http.ResponseWriter, request *http.Request) {
	var exportRequest struct {
		DocumentID string `json:"document_id"`
		LectureID  string `json:"lecture_id"`
		ExamID     string `json:"exam_id"`
		Format     string `json:"format"` // "pdf", "docx", "md"
	}

	if decodingError := json.NewDecoder(request.Body).Decode(&exportRequest); decodingError != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	if exportRequest.DocumentID == "" || exportRequest.LectureID == "" || exportRequest.ExamID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "document_id, lecture_id and exam_id are required", nil)
		return
	}

	if exportRequest.Format == "" {
		exportRequest.Format = "pdf"
	}

	userID := server.getUserID(request)

	// Verify document exists and belongs to the user
	var docTitle string
	var languageCode sql.NullString
	queryError := server.database.QueryRow(`
		SELECT reference_documents.title, lectures.language
		FROM reference_documents
		JOIN lectures ON reference_documents.lecture_id = lectures.id
		JOIN exams ON lectures.exam_id = exams.id
		WHERE reference_documents.id = ? AND reference_documents.lecture_id = ? AND exams.user_id = ?
	`, exportRequest.DocumentID, exportRequest.LectureID, userID).Scan(&docTitle, &languageCode)

	if queryError == sql.ErrNoRows {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Document not found in this lecture", nil)
		return
	}
	if queryError != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to verify document", nil)
		return
	}

	lang := ""
	if languageCode.Valid {
		lang = languageCode.String
	}

	if lang == "" {
		lang = server.configuration.LLM.Language
	}

	// Enqueue export job
	jobIdentifier, enqueuingError := server.jobQueue.Enqueue(userID, models.JobTypePublishMaterial, map[string]string{
		"document_id":   exportRequest.DocumentID,
		"lecture_id":    exportRequest.LectureID,
		"language_code": lang,
		"format":        exportRequest.Format,
	}, exportRequest.ExamID, exportRequest.LectureID)

	if enqueuingError != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "BACKGROUND_JOB_ERROR", "Failed to create export job", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusAccepted, map[string]string{
		"job_id":  jobIdentifier,
		"message": "Document export job created",
	})
}

// handleDownloadExport serves the generated export file
func (server *Server) handleDownloadExport(responseWriter http.ResponseWriter, request *http.Request) {
	filePath := request.URL.Query().Get("path")
	if filePath == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "path is required", nil)
		return
	}

	// Basic security check: ensure path is within data directory
	absoluteDataDir, _ := filepath.Abs(server.configuration.Storage.DataDirectory)
	absoluteFilePath, _ := filepath.Abs(filePath)

	// Robust prefix check using Rel
	rel, err := filepath.Rel(absoluteDataDir, absoluteFilePath)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		server.writeError(responseWriter, http.StatusForbidden, "ACCESS_DENIED", "Access to this file is forbidden", nil)
		return
	}

	if _, err := os.Stat(absoluteFilePath); os.IsNotExist(err) {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "File not found", nil)
		return
	}

	// Set content-disposition to force download with original filename
	fileName := filepath.Base(absoluteFilePath)
	// Properly escape filename for Content-Disposition
	encodedFileName := url.PathEscape(fileName)
	responseWriter.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"; filename*=UTF-8''%s", fileName, encodedFileName))

	// Set Content-Type based on extension for better blob handling
	ext := strings.ToLower(filepath.Ext(fileName))
	switch ext {
	case ".pdf":
		responseWriter.Header().Set("Content-Type", "application/pdf")
	case ".docx":
		responseWriter.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	case ".md":
		responseWriter.Header().Set("Content-Type", "text/markdown")
	}

	http.ServeFile(responseWriter, request, absoluteFilePath)
}
