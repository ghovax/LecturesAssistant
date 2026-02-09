package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"strconv"

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
		SELECT reference_documents.id, reference_documents.lecture_id, reference_documents.document_type, reference_documents.title, reference_documents.file_path, reference_documents.page_count, reference_documents.extraction_status, reference_documents.created_at, reference_documents.updated_at
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

	var documentsList []models.ReferenceDocument
	for documentRows.Next() {
		var document models.ReferenceDocument
		if err := documentRows.Scan(&document.ID, &document.LectureID, &document.DocumentType, &document.Title, &document.FilePath, &document.PageCount, &document.ExtractionStatus, &document.CreatedAt, &document.UpdatedAt); err != nil {
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
		SELECT reference_documents.id, reference_documents.lecture_id, reference_documents.document_type, reference_documents.title, reference_documents.file_path, reference_documents.page_count, reference_documents.extraction_status, reference_documents.created_at, reference_documents.updated_at
		FROM reference_documents
		JOIN lectures ON reference_documents.lecture_id = lectures.id
		JOIN exams ON lectures.exam_id = exams.id
		WHERE reference_documents.id = ? AND reference_documents.lecture_id = ? AND exams.user_id = ?
	`, documentID, lectureID, userID).Scan(&document.ID, &document.LectureID, &document.DocumentType, &document.Title, &document.FilePath, &document.PageCount, &document.ExtractionStatus, &document.CreatedAt, &document.UpdatedAt)

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

	var pages []models.ReferencePage
	for pageRows.Next() {
		var page models.ReferencePage
		if err := pageRows.Scan(&page.ID, &page.DocumentID, &page.PageNumber, &page.ImagePath, &page.ExtractedText); err != nil {
			continue
		}
		pages = append(pages, page)
	}

	server.writeJSON(responseWriter, http.StatusOK, pages)
}

// handleGetPageImage serves the actual image file for a page
func (server *Server) handleGetPageImage(responseWriter http.ResponseWriter, request *http.Request) {
	documentID := request.URL.Query().Get("document_id")
	lectureID := request.URL.Query().Get("lecture_id")
	pageNumberString := request.URL.Query().Get("page_number")

	if documentID == "" || lectureID == "" || pageNumberString == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "document_id, lecture_id and page_number are required", nil)
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

	pageNumber, _ := strconv.Atoi(pageNumberString)

	var imagePath string
	err = server.database.QueryRow(`
		SELECT image_path
		FROM reference_pages
		WHERE document_id = ? AND page_number = ?
	`, documentID, pageNumber).Scan(&imagePath)

	if err == sql.ErrNoRows {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Page not found", nil)
		return
	}
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get page image", nil)
		return
	}

	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		server.writeError(responseWriter, http.StatusNotFound, "FILE_NOT_FOUND", "Image file not found on disk", nil)
		return
	}

	http.ServeFile(responseWriter, request, imagePath)
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

	var extractedText string
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

	// Convert to HTML
	markdownText := fmt.Sprintf("# %s - Page %d\n\n%s", docTitle, pageNumber, extractedText)
	htmlContent, err := server.markdownConverter.MarkdownToHTML(markdownText)
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "CONVERSION_ERROR", "Failed to convert page to HTML", nil)
		return
	}

	responseWriter.Header().Set("Content-Type", "text/html")
	responseWriter.WriteHeader(http.StatusOK)
	responseWriter.Write([]byte(htmlContent))
}
