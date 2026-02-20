package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"lectures/internal/models"
)

// handleListDocuments lists all reference documents for a lecture
func (server *Server) handleListDocuments(responseWriter http.ResponseWriter, request *http.Request) {
	lectureID := request.URL.Query().Get("lecture_id")
	if lectureID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "lecture_id is required", nil)
		return
	}

	userID := server.getUserID(request)

	documentRows, databaseError := server.database.Query(`
		SELECT reference_documents.id, reference_documents.lecture_id, reference_documents.document_type, reference_documents.title, reference_documents.file_path, reference_documents.page_count, reference_documents.extraction_status, reference_documents.estimated_cost, reference_documents.created_at, reference_documents.updated_at
		FROM reference_documents
		JOIN lectures ON reference_documents.lecture_id = lectures.id
		JOIN exams ON lectures.exam_id = exams.id
		WHERE reference_documents.lecture_id = ? AND exams.user_id = ?
	`, lectureID, userID)
	if databaseError != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to list documents", nil)
		return
	}
	defer documentRows.Close()

	var documentsList = []models.ReferenceDocument{}
	for documentRows.Next() {
		var document models.ReferenceDocument
		if err := documentRows.Scan(&document.ID, &document.LectureID, &document.DocumentType, &document.Title, &document.FilePath, &document.PageCount, &document.ExtractionStatus, &document.EstimatedCost, &document.CreatedAt, &document.UpdatedAt); err != nil {
			continue
		}
		documentsList = append(documentsList, document)
	}

	server.writeJSON(responseWriter, http.StatusOK, documentsList)
}

// handleGetDocument retrieves a specific document metadata
func (server *Server) handleGetDocument(responseWriter http.ResponseWriter, request *http.Request) {
	documentID := request.URL.Query().Get("document_id")
	lectureID := request.URL.Query().Get("lecture_id")

	if documentID == "" || lectureID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "document_id and lecture_id are required", nil)
		return
	}

	userID := server.getUserID(request)

	var document models.ReferenceDocument
	err := server.database.QueryRow(`
		SELECT reference_documents.id, reference_documents.lecture_id, reference_documents.document_type, reference_documents.title, reference_documents.file_path, reference_documents.page_count, reference_documents.extraction_status, reference_documents.estimated_cost, reference_documents.created_at, reference_documents.updated_at
		FROM reference_documents
		JOIN lectures ON reference_documents.lecture_id = lectures.id
		JOIN exams ON lectures.exam_id = exams.id
		WHERE reference_documents.id = ? AND reference_documents.lecture_id = ? AND exams.user_id = ?
	`, documentID, lectureID, userID).Scan(&document.ID, &document.LectureID, &document.DocumentType, &document.Title, &document.FilePath, &document.PageCount, &document.ExtractionStatus, &document.EstimatedCost, &document.CreatedAt, &document.UpdatedAt)

	if err == sql.ErrNoRows {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Document not found in this lecture", nil)
		return
	}
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get document", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusOK, document)
}

// handleGetDocumentPages lists all pages for a document
func (server *Server) handleGetDocumentPages(responseWriter http.ResponseWriter, request *http.Request) {
	documentID := request.URL.Query().Get("document_id")
	lectureID := request.URL.Query().Get("lecture_id")

	if documentID == "" || lectureID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "document_id and lecture_id are required", nil)
		return
	}

	userID := server.getUserID(request)

	// Verify document belongs to lecture and user
	var exists bool
	err := server.database.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM reference_documents 
			JOIN lectures ON reference_documents.lecture_id = lectures.id
			JOIN exams ON lectures.exam_id = exams.id
			WHERE reference_documents.id = ? AND reference_documents.lecture_id = ? AND exams.user_id = ?
		)
	`, documentID, lectureID, userID).Scan(&exists)
	if err != nil || !exists {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Document not found in this lecture", nil)
		return
	}

	pageRows, databaseError := server.database.Query(`
		SELECT id, document_id, page_number, image_path, extracted_text
		FROM reference_pages
		WHERE document_id = ?
		ORDER BY page_number ASC
	`, documentID)
	if databaseError != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to list pages", nil)
		return
	}
	defer pageRows.Close()

	type pageResponse struct {
		ID            string `json:"id"`
		DocumentID    string `json:"document_id"`
		PageNumber    int    `json:"page_number"`
		ImagePath     string `json:"image_path"`
		ExtractedText string `json:"extracted_text"`
		ExtractedHTML string `json:"extracted_html"`
	}

	var pages []pageResponse
	for pageRows.Next() {
		var page models.ReferencePage
		var extractedText sql.NullString
		if err := pageRows.Scan(&page.ID, &page.DocumentID, &page.PageNumber, &page.ImagePath, &extractedText); err != nil {
			continue
		}

		if extractedText.Valid {
			page.ExtractedText = extractedText.String
		}

		// Convert extracted text to HTML
		htmlContent := page.ExtractedText
		if page.ExtractedText != "" {
			convertedHTML, err := server.markdownConverter.MarkdownToHTML(page.ExtractedText)
			if err == nil {
				htmlContent = convertedHTML
			}
		}

		pages = append(pages, pageResponse{
			ID:            strconv.Itoa(page.ID),
			DocumentID:    page.DocumentID,
			PageNumber:    page.PageNumber,
			ImagePath:     page.ImagePath,
			ExtractedText: page.ExtractedText,
			ExtractedHTML: htmlContent,
		})
	}

	server.writeJSON(responseWriter, http.StatusOK, pages)
}

// handleGetPageImage serves the actual image file for a page
func (server *Server) handleGetPageImage(responseWriter http.ResponseWriter, request *http.Request) {
	documentID := request.URL.Query().Get("document_id")
	lectureID := request.URL.Query().Get("lecture_id")
	pageNumberString := request.URL.Query().Get("page_number")

	slog.Info("Page image request", "document_id", documentID, "lecture_id", lectureID, "page_number", pageNumberString)

	if documentID == "" || lectureID == "" || pageNumberString == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "document_id, lecture_id and page_number are required", nil)
		return
	}

	// Try multiple token sources with validation (cookie -> header -> query param)
	// This handles cases where old cookies exist but the current session uses a different token
	sessionToken := server.getValidSessionToken(request)
	if sessionToken == "" {
		slog.Warn("Page image request without valid session token", "document_id", documentID, "lecture_id", lectureID)
		server.writeError(responseWriter, http.StatusUnauthorized, "AUTHENTICATION_ERROR", "Authentication required", nil)
		return
	}

	// Get user info from validated session
	var userID string
	var expiresAt time.Time
	err := server.database.QueryRow("SELECT user_id, expires_at FROM auth_sessions WHERE id = ?", sessionToken).Scan(&userID, &expiresAt)
	if err != nil {
		slog.Warn("Invalid session token for page image", "document_id", documentID, "lecture_id", lectureID)
		server.writeError(responseWriter, http.StatusUnauthorized, "AUTHENTICATION_ERROR", "Invalid session", nil)
		return
	}

	if time.Now().After(expiresAt) {
		slog.Warn("Expired session token for page image", "document_id", documentID, "lecture_id", lectureID)
		server.writeError(responseWriter, http.StatusUnauthorized, "AUTHENTICATION_ERROR", "Session expired", nil)
		return
	}

	// Verify document belongs to lecture and user
	var exists bool
	err = server.database.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM reference_documents
			JOIN lectures ON reference_documents.lecture_id = lectures.id
			JOIN exams ON lectures.exam_id = exams.id
			WHERE reference_documents.id = ? AND reference_documents.lecture_id = ? AND exams.user_id = ?
		)
	`, documentID, lectureID, userID).Scan(&exists)
	if err != nil {
		slog.Error("Database error checking document", "error", err, "document_id", documentID)
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to verify document", nil)
		return
	}
	if !exists {
		slog.Warn("Document not found or access denied", "document_id", documentID, "lecture_id", lectureID, "user_id", userID)
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Document not found in this lecture", nil)
		return
	}

	pageNumber, _ := strconv.Atoi(pageNumberString)

	var imagePath string
	var imageData []byte
	err = server.database.QueryRow(`
		SELECT image_path, image_data
		FROM reference_pages
		WHERE document_id = ? AND page_number = ?
	`, documentID, pageNumber).Scan(&imagePath, &imageData)

	if err == sql.ErrNoRows {
		slog.Warn("Page not found in database", "document_id", documentID, "page_number", pageNumber)
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Page not found", nil)
		return
	}
	if err != nil {
		slog.Error("Database error fetching page", "error", err, "document_id", documentID, "page_number", pageNumber)
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get page image", nil)
		return
	}

	slog.Info("Page found", "document_id", documentID, "page_number", pageNumber, "image_path", imagePath, "image_data_size", len(imageData))

	// Serve from DB BLOB if available (works after backup/restore)
	if len(imageData) > 0 {
		slog.Info("Serving image from database BLOB", "document_id", documentID, "page_number", pageNumber, "size_bytes", len(imageData))
		responseWriter.Header().Set("Content-Type", "image/png")
		responseWriter.Header().Set("Content-Length", fmt.Sprintf("%d", len(imageData)))
		responseWriter.Header().Set("Cache-Control", "public, max-age=86400")
		responseWriter.Write(imageData)
		return
	}

	// Image data not found in database
	slog.Error("Image data is empty in database", "document_id", documentID, "page_number", pageNumber, "image_path", imagePath)
	server.writeError(responseWriter, http.StatusNotFound, "IMAGE_NOT_FOUND", "Page image not found. The image may not have been processed or stored correctly.", nil)
}

// handleDeleteDocument deletes a specific reference document and its files
func (server *Server) handleDeleteDocument(responseWriter http.ResponseWriter, request *http.Request) {
	var deleteRequest struct {
		DocumentID string `json:"document_id"`
		LectureID  string `json:"lecture_id"`
	}
	if err := json.NewDecoder(request.Body).Decode(&deleteRequest); err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	if deleteRequest.DocumentID == "" || deleteRequest.LectureID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "document_id and lecture_id are required", nil)
		return
	}

	userID := server.getUserID(request)

	// Get file path and verify ownership
	var filePath string
	err := server.database.QueryRow(`
		SELECT reference_documents.file_path FROM reference_documents 
		JOIN lectures ON reference_documents.lecture_id = lectures.id
		JOIN exams ON lectures.exam_id = exams.id
		WHERE reference_documents.id = ? AND reference_documents.lecture_id = ? AND exams.user_id = ?
	`, deleteRequest.DocumentID, deleteRequest.LectureID, userID).Scan(&filePath)

	if err == sql.ErrNoRows {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Document not found", nil)
		return
	}
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to verify document", nil)
		return
	}

	// Delete from database (cascades to reference_pages)
	_, err = server.database.Exec("DELETE FROM reference_documents WHERE id = ?", deleteRequest.DocumentID)
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to delete document from database", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusOK, map[string]string{"message": "Document deleted successfully"})
}

// handleGetPageHTML serves the extracted text of a page converted to HTML
func (server *Server) handleGetPageHTML(responseWriter http.ResponseWriter, request *http.Request) {
	documentID := request.URL.Query().Get("document_id")
	lectureID := request.URL.Query().Get("lecture_id")
	pageNumberString := request.URL.Query().Get("page_number")

	if documentID == "" || lectureID == "" || pageNumberString == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "document_id, lecture_id and page_number are required", nil)
		return
	}

	userID := server.getUserID(request)

	// Verify document ownership
	var docTitle string
	err := server.database.QueryRow(`
		SELECT reference_documents.title FROM reference_documents 
		JOIN lectures ON reference_documents.lecture_id = lectures.id
		JOIN exams ON lectures.exam_id = exams.id
		WHERE reference_documents.id = ? AND reference_documents.lecture_id = ? AND exams.user_id = ?
	`, documentID, lectureID, userID).Scan(&docTitle)

	if err == sql.ErrNoRows {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Document not found in this lecture", nil)
		return
	}
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to verify document", nil)
		return
	}

	pageNumber, _ := strconv.Atoi(pageNumberString)

	var extractedText sql.NullString
	err = server.database.QueryRow(`
		SELECT extracted_text
		FROM reference_pages
		WHERE document_id = ? AND page_number = ?
	`, documentID, pageNumber).Scan(&extractedText)

	if err == sql.ErrNoRows {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Page not found", nil)
		return
	}
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get page content", nil)
		return
	}

	text := ""
	if extractedText.Valid {
		text = extractedText.String
	}

	// Convert to HTML
	markdownText := fmt.Sprintf("# %s - Page %d\n\n%s", docTitle, pageNumber, text)
	htmlContent, err := server.markdownConverter.MarkdownToHTML(markdownText)
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "CONVERSION_ERROR", "Failed to convert page to HTML", nil)
		return
	}

	responseWriter.Header().Set("Content-Type", "text/html")
	responseWriter.WriteHeader(http.StatusOK)
	responseWriter.Write([]byte(htmlContent))
}
