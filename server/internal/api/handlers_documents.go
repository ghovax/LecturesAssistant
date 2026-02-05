package api

import (
	"database/sql"
	"net/http"
	"os"
	"strconv"

	"lectures/internal/models"

	"github.com/gorilla/mux"
)

// handleListDocuments lists all reference documents for a lecture
func (server *Server) handleListDocuments(responseWriter http.ResponseWriter, request *http.Request) {
	pathVariables := mux.Vars(request)
	lectureIdentifier := pathVariables["lectureId"]

	documentRows, databaseError := server.database.Query(`
		SELECT id, lecture_id, document_type, title, file_path, page_count, extraction_status, created_at, updated_at
		FROM reference_documents
		WHERE lecture_id = ?
	`, lectureIdentifier)
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
	pathVariables := mux.Vars(request)
	documentIdentifier := pathVariables["documentId"]

	var document models.ReferenceDocument
	err := server.database.QueryRow(`
		SELECT id, lecture_id, document_type, title, file_path, page_count, extraction_status, created_at, updated_at
		FROM reference_documents
		WHERE id = ?
	`, documentIdentifier).Scan(&document.ID, &document.LectureID, &document.DocumentType, &document.Title, &document.FilePath, &document.PageCount, &document.ExtractionStatus, &document.CreatedAt, &document.UpdatedAt)

	if err == sql.ErrNoRows {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Document not found", nil)
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
	pathVariables := mux.Vars(request)
	documentIdentifier := pathVariables["documentId"]

	pageRows, databaseError := server.database.Query(`
		SELECT id, document_id, page_number, image_path, extracted_text
		FROM reference_pages
		WHERE document_id = ?
		ORDER BY page_number ASC
	`, documentIdentifier)
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
	pathVariables := mux.Vars(request)
	documentIdentifier := pathVariables["documentId"]
	pageNumberString := pathVariables["pageNumber"]
	pageNumber, _ := strconv.Atoi(pageNumberString)

	var imagePath string
	err := server.database.QueryRow(`
		SELECT image_path
		FROM reference_pages
		WHERE document_id = ? AND page_number = ?
	`, documentIdentifier, pageNumber).Scan(&imagePath)

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
