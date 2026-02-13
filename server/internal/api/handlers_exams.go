package api

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"lectures/internal/models"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

// handleCreateExam creates a new exam
func (server *Server) handleCreateExam(responseWriter http.ResponseWriter, request *http.Request) {
	var createExamRequest struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Language    string `json:"language"`
	}

	if err := json.NewDecoder(request.Body).Decode(&createExamRequest); err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	if createExamRequest.Title == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Title is required", nil)
		return
	}

	// Clean title and description
	title, description, metrics, err := server.toolGenerator.CorrectProjectTitleDescription(request.Context(), createExamRequest.Title, createExamRequest.Description, "")
	if err != nil {
		slog.Error("Failed to polish exam title/description", "error", err)
	} else {
		slog.Info("Exam title/description polished",
			"input_tokens", metrics.InputTokens,
			"output_tokens", metrics.OutputTokens,
			"estimated_cost_usd", metrics.EstimatedCost)
	}

	userID := server.getUserID(request)

	examID, _ := gonanoid.New()
	exam := models.Exam{
		ID:            examID,
		UserID:        userID,
		Title:         title,
		Description:   description,
		Language:      createExamRequest.Language,
		EstimatedCost: metrics.EstimatedCost,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	_, err = server.database.Exec(`
		INSERT INTO exams (id, user_id, title, description, language, estimated_cost, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, exam.ID, exam.UserID, exam.Title, exam.Description, exam.Language, exam.EstimatedCost, exam.CreatedAt, exam.UpdatedAt)

	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to create exam", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusCreated, exam)
}

// handleListExams lists all exams for the current user
func (server *Server) handleListExams(responseWriter http.ResponseWriter, request *http.Request) {
	userID := server.getUserID(request)

	examRows, databaseError := server.database.Query(`
		SELECT id, user_id, title, description, language, estimated_cost, created_at, updated_at
		FROM exams
		WHERE user_id = ?
		ORDER BY created_at DESC
	`, userID)
	if databaseError != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to list exams", nil)
		return
	}
	defer examRows.Close()

	type examResponse struct {
		models.Exam
		DescriptionHTML string `json:"description_html"`
	}

	exams := []examResponse{}
	for examRows.Next() {
		var exam models.Exam
		var description, language sql.NullString
		if err := examRows.Scan(&exam.ID, &exam.UserID, &exam.Title, &description, &language, &exam.EstimatedCost, &exam.CreatedAt, &exam.UpdatedAt); err != nil {
			server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to scan exam", nil)
			return
		}
		if description.Valid {
			exam.Description = description.String
		}
		if language.Valid {
			exam.Language = language.String
		}

		// Convert description to HTML
		response := examResponse{Exam: exam}
		if exam.Description != "" {
			htmlContent, err := server.markdownConverter.MarkdownToHTML(exam.Description)
			if err == nil {
				response.DescriptionHTML = htmlContent
			} else {
				response.DescriptionHTML = exam.Description
			}
		}

		exams = append(exams, response)
	}

	server.writeJSON(responseWriter, http.StatusOK, exams)
}

// handleGetExam retrieves a specific exam for the current user
func (server *Server) handleGetExam(responseWriter http.ResponseWriter, request *http.Request) {
	examID := request.URL.Query().Get("exam_id")
	if examID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "exam_id is required", nil)
		return
	}

	userID := server.getUserID(request)

	var exam models.Exam
	var description, language sql.NullString
	err := server.database.QueryRow(`
		SELECT id, user_id, title, description, language, estimated_cost, created_at, updated_at
		FROM exams
		WHERE id = ? AND user_id = ?
	`, examID, userID).Scan(&exam.ID, &exam.UserID, &exam.Title, &description, &language, &exam.EstimatedCost, &exam.CreatedAt, &exam.UpdatedAt)

	if description.Valid {
		exam.Description = description.String
	}
	if language.Valid {
		exam.Language = language.String
	}

	if err == sql.ErrNoRows {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Exam not found", nil)
		return
	}
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get exam", nil)
		return
	}

	// Convert description to HTML
	type examResponse struct {
		models.Exam
		DescriptionHTML string `json:"description_html"`
	}

	response := examResponse{Exam: exam}
	if exam.Description != "" {
		htmlContent, err := server.markdownConverter.MarkdownToHTML(exam.Description)
		if err == nil {
			response.DescriptionHTML = htmlContent
		} else {
			response.DescriptionHTML = exam.Description
		}
	}

	server.writeJSON(responseWriter, http.StatusOK, response)
}

// handleUpdateExam updates an exam owned by the user
func (server *Server) handleUpdateExam(responseWriter http.ResponseWriter, request *http.Request) {
	var updateExamRequest struct {
		ExamID      string  `json:"exam_id"`
		Title       *string `json:"title"`
		Description *string `json:"description"`
	}

	if err := json.NewDecoder(request.Body).Decode(&updateExamRequest); err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	if updateExamRequest.ExamID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "exam_id is required", nil)
		return
	}

	userID := server.getUserID(request)

	// Check if exam exists and belongs to user
	var exists bool
	err := server.database.QueryRow("SELECT EXISTS(SELECT 1 FROM exams WHERE id = ? AND user_id = ?)", updateExamRequest.ExamID, userID).Scan(&exists)
	if err != nil || !exists {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Exam not found", nil)
		return
	}

	// Build update query dynamically
	updates := []any{}
	query := "UPDATE exams SET updated_at = ?"
	updates = append(updates, time.Now())

	if updateExamRequest.Title != nil || updateExamRequest.Description != nil {
		currentTitle := ""
		currentDescription := ""
		server.database.QueryRow("SELECT title, description FROM exams WHERE id = ? AND user_id = ?", updateExamRequest.ExamID, userID).Scan(&currentTitle, &currentDescription)

		newTitle := currentTitle
		if updateExamRequest.Title != nil {
			newTitle = *updateExamRequest.Title
		}
		newDescription := currentDescription
		if updateExamRequest.Description != nil {
			newDescription = *updateExamRequest.Description
		}

		cleanedTitle, cleanedDescription, metrics, err := server.toolGenerator.CorrectProjectTitleDescription(request.Context(), newTitle, newDescription, "")
		if err != nil {
			slog.Error("Failed to polish updated exam title/description", "examID", updateExamRequest.ExamID, "error", err)
			cleanedTitle = newTitle
			cleanedDescription = newDescription
		} else {
			slog.Info("Exam title/description updated and polished",
				"examID", updateExamRequest.ExamID,
				"input_tokens", metrics.InputTokens,
				"output_tokens", metrics.OutputTokens,
				"estimated_cost_usd", metrics.EstimatedCost)
		}

		if updateExamRequest.Title != nil {
			query += ", title = ?"
			updates = append(updates, cleanedTitle)
		}
		if updateExamRequest.Description != nil {
			query += ", description = ?"
			updates = append(updates, cleanedDescription)
		}
		query += ", estimated_cost = estimated_cost + ?"
		updates = append(updates, metrics.EstimatedCost)
	}

	query += " WHERE id = ? AND user_id = ?"
	updates = append(updates, updateExamRequest.ExamID, userID)

	_, err = server.database.Exec(query, updates...)
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to update exam", nil)
		return
	}

	// Fetch updated exam
	var exam models.Exam
	var description sql.NullString
	err = server.database.QueryRow(`
		SELECT id, user_id, title, description, created_at, updated_at
		FROM exams
		WHERE id = ? AND user_id = ?
	`, updateExamRequest.ExamID, userID).Scan(&exam.ID, &exam.UserID, &exam.Title, &description, &exam.CreatedAt, &exam.UpdatedAt)

	if description.Valid {
		exam.Description = description.String
	}

	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to fetch updated exam", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusOK, exam)
}

// handleDeleteExam deletes an exam and all associated data
func (server *Server) handleDeleteExam(responseWriter http.ResponseWriter, request *http.Request) {
	var deleteRequest struct {
		ExamID string `json:"exam_id"`
	}
	if err := json.NewDecoder(request.Body).Decode(&deleteRequest); err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid body", nil)
		return
	}

	if deleteRequest.ExamID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "exam_id is required", nil)
		return
	}

	userID := server.getUserID(request)

	// 1. Get all lecture IDs for this exam to clean up files later
	lectureRows, queryError := server.database.Query(`
		SELECT lectures.id FROM lectures 
		JOIN exams ON lectures.exam_id = exams.id
		WHERE exams.id = ? AND exams.user_id = ?
	`, deleteRequest.ExamID, userID)

	var lectureIdentifiers []string
	if queryError == nil {
		for lectureRows.Next() {
			var lectureIdentifier string
			if err := lectureRows.Scan(&lectureIdentifier); err == nil {
				lectureIdentifiers = append(lectureIdentifiers, lectureIdentifier)
			}
		}
		lectureRows.Close()
	}
	// 2. Delete from database
	result, err := server.database.Exec("DELETE FROM exams WHERE id = ? AND user_id = ?", deleteRequest.ExamID, userID)
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to delete exam", nil)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Exam not found", nil)
		return
	}

	// 3. Delete lecture files from filesystem
	for _, lectureIdentifier := range lectureIdentifiers {
		lectureDirectory := filepath.Join(server.configuration.Storage.DataDirectory, "files", "lectures", lectureIdentifier)
		_ = os.RemoveAll(lectureDirectory)
	}

	server.writeJSON(responseWriter, http.StatusOK, map[string]string{"message": "Exam deleted successfully"})
}

// handleExamSearch performs a global keyword search across all materials in an exam
func (server *Server) handleExamSearch(responseWriter http.ResponseWriter, request *http.Request) {
	examID := request.URL.Query().Get("exam_id")
	query := request.URL.Query().Get("query")

	if examID == "" || query == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "exam_id and query are required", nil)
		return
	}

	userID := server.getUserID(request)

	// Verify exam ownership
	var exists bool
	err := server.database.QueryRow("SELECT EXISTS(SELECT 1 FROM exams WHERE id = ? AND user_id = ?)", examID, userID).Scan(&exists)
	if err != nil || !exists {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Exam not found", nil)
		return
	}

	type searchResult struct {
		Type      string `json:"type"` // "transcript" or "document"
		LectureID string `json:"lecture_id"`
		Title     string `json:"title"`
		Snippet   string `json:"snippet"`
		Metadata  any    `json:"metadata,omitempty"`
	}
	var results []searchResult

	// 1. Search in Transcripts
	transcriptRows, err := server.database.Query(`
		SELECT lectures.id, lectures.title, transcript_segments.text, transcript_segments.start_millisecond
		FROM transcript_segments
		JOIN transcripts ON transcript_segments.transcript_id = transcripts.id
		JOIN lectures ON transcripts.lecture_id = lectures.id
		JOIN exams ON lectures.exam_id = exams.id
		WHERE lectures.exam_id = ? AND exams.user_id = ? AND transcript_segments.text LIKE ?
		LIMIT 50
	`, examID, userID, "%"+query+"%")

	if err == nil {
		for transcriptRows.Next() {
			var lID, lTitle, text string
			var startMs int64
			if err := transcriptRows.Scan(&lID, &lTitle, &text, &startMs); err == nil {
				results = append(results, searchResult{
					Type:      "transcript",
					LectureID: lID,
					Title:     lTitle,
					Snippet:   text,
					Metadata:  map[string]int64{"start_millisecond": startMs},
				})
			}
		}
		transcriptRows.Close()
	}

	// 2. Search in Document Pages
	documentRows, err := server.database.Query(`
		SELECT lectures.id, reference_documents.title, reference_pages.extracted_text, reference_pages.page_number, reference_documents.id
		FROM reference_pages
		JOIN reference_documents ON reference_pages.document_id = reference_documents.id
		JOIN lectures ON reference_documents.lecture_id = lectures.id
		JOIN exams ON lectures.exam_id = exams.id
		WHERE lectures.exam_id = ? AND exams.user_id = ? AND reference_pages.extracted_text LIKE ?
		LIMIT 50
	`, examID, userID, "%"+query+"%")

	if err == nil {
		for documentRows.Next() {
			var lID, docTitle, text, docID string
			var pageNum int
			if err := documentRows.Scan(&lID, &docTitle, &text, &pageNum, &docID); err == nil {
				results = append(results, searchResult{
					Type:      "document",
					LectureID: lID,
					Title:     docTitle,
					Snippet:   text,
					Metadata:  map[string]any{"page_number": pageNum, "document_id": docID},
				})
			}
		}
		documentRows.Close()
	}

	server.writeJSON(responseWriter, http.StatusOK, results)
}

// handleExamSuggest triggers an AI job to suggest better metadata for an exam
func (server *Server) handleExamSuggest(responseWriter http.ResponseWriter, request *http.Request) {
	var suggestRequest struct {
		ExamID string `json:"exam_id"`
	}
	if err := json.NewDecoder(request.Body).Decode(&suggestRequest); err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	if suggestRequest.ExamID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "exam_id is required", nil)
		return
	}

	userID := server.getUserID(request)

	// Verify exam ownership
	var exists bool
	err := server.database.QueryRow("SELECT EXISTS(SELECT 1 FROM exams WHERE id = ? AND user_id = ?)", suggestRequest.ExamID, userID).Scan(&exists)
	if err != nil || !exists {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Exam not found", nil)
		return
	}

	jobID, err := server.jobQueue.Enqueue(userID, models.JobTypeSuggest, map[string]string{
		"exam_id": suggestRequest.ExamID,
	}, suggestRequest.ExamID, "")

	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "BACKGROUND_JOB_ERROR", "Failed to enqueue suggest job", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusAccepted, map[string]string{
		"job_id":  jobID,
		"message": "Suggestion job created",
	})
}

// handleGetExamConcepts retrieves a "concept map" or glossary for an exam based on processed materials
func (server *Server) handleGetExamConcepts(responseWriter http.ResponseWriter, request *http.Request) {
	examID := request.URL.Query().Get("exam_id")
	if examID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "exam_id is required", nil)
		return
	}

	userID := server.getUserID(request)

	// Verify ownership
	var exists bool
	err := server.database.QueryRow("SELECT EXISTS(SELECT 1 FROM exams WHERE id = ? AND user_id = ?)", examID, userID).Scan(&exists)
	if err != nil || !exists {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Exam not found", nil)
		return
	}

	// Extract unique concepts from tool_source_references (which contains the citation metadata)
	rows, err := server.database.Query(`
		SELECT DISTINCT json_extract(metadata, '$.description') as concept
		FROM tool_source_references
		JOIN tools ON tool_source_references.tool_id = tools.id
		WHERE tools.exam_id = ? AND concept IS NOT NULL
		ORDER BY concept ASC
	`, examID)

	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retrieve concepts", nil)
		return
	}
	defer rows.Close()

	var concepts []string
	for rows.Next() {
		var concept string
		if err := rows.Scan(&concept); err == nil && concept != "" {
			concepts = append(concepts, concept)
		}
	}

	server.writeJSON(responseWriter, http.StatusOK, concepts)
}
