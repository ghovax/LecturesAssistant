package api

import (
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	config "lectures/internal/configuration"
	"lectures/internal/jobs"
	"lectures/internal/models"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// Server represents the API server
type Server struct {
	configuration *config.Configuration
	database      *sql.DB
	jobQueue      *jobs.Queue
	router        *mux.Router
}

// NewServer creates a new API server
func NewServer(configuration *config.Configuration, database *sql.DB, jobQueue *jobs.Queue) *Server {
	server := &Server{
		configuration: configuration,
		database:      database,
		jobQueue:      jobQueue,
		router:        mux.NewRouter(),
	}

	server.setupRoutes()
	return server
}

// Handler returns the HTTP handler
func (server *Server) Handler() http.Handler {
	return server.router
}

// setupRoutes configures all API routes
func (server *Server) setupRoutes() {
	// Health check (no auth required)
	server.router.HandleFunc("/api/health", server.handleHealth).Methods("GET")

	// API routes (with middleware)
	apiRouter := server.router.PathPrefix("/api").Subrouter()
	apiRouter.Use(server.requestIDMiddleware)
	apiRouter.Use(server.loggingMiddleware)

	// Exams
	apiRouter.HandleFunc("/exams", server.handleCreateExam).Methods("POST")
	apiRouter.HandleFunc("/exams", server.handleListExams).Methods("GET")
	apiRouter.HandleFunc("/exams/{id}", server.handleGetExam).Methods("GET")
	apiRouter.HandleFunc("/exams/{id}", server.handleUpdateExam).Methods("PATCH")
	apiRouter.HandleFunc("/exams/{id}", server.handleDeleteExam).Methods("DELETE")

	// Lectures
	apiRouter.HandleFunc("/exams/{examId}/lectures", server.handleCreateLecture).Methods("POST")
	apiRouter.HandleFunc("/exams/{examId}/lectures", server.handleListLectures).Methods("GET")
	apiRouter.HandleFunc("/exams/{examId}/lectures/{lectureId}", server.handleGetLecture).Methods("GET")
	apiRouter.HandleFunc("/exams/{examId}/lectures/{lectureId}", server.handleUpdateLecture).Methods("PATCH")
	apiRouter.HandleFunc("/exams/{examId}/lectures/{lectureId}", server.handleDeleteLecture).Methods("DELETE")

	// Media
	apiRouter.HandleFunc("/exams/{examId}/lectures/{lectureId}/media", server.handleListMedia).Methods("GET")

	// Transcripts
	apiRouter.HandleFunc("/exams/{examId}/lectures/{lectureId}/transcript", server.handleGetTranscript).Methods("GET")

	// Reference Documents
	apiRouter.HandleFunc("/exams/{examId}/lectures/{lectureId}/documents", server.handleListDocuments).Methods("GET")
	apiRouter.HandleFunc("/exams/{examId}/lectures/{lectureId}/documents/{documentId}", server.handleGetDocument).Methods("GET")
	apiRouter.HandleFunc("/exams/{examId}/lectures/{lectureId}/documents/{documentId}/pages", server.handleGetDocumentPages).Methods("GET")
	apiRouter.HandleFunc("/exams/{examId}/lectures/{lectureId}/documents/{documentId}/pages/{pageNumber}/image", server.handleGetPageImage).Methods("GET")

	// Tools
	apiRouter.HandleFunc("/exams/{examId}/tools", server.handleCreateTool).Methods("POST")
	apiRouter.HandleFunc("/exams/{examId}/tools", server.handleListTools).Methods("GET")
	apiRouter.HandleFunc("/exams/{examId}/tools/{toolId}", server.handleGetTool).Methods("GET")
	apiRouter.HandleFunc("/exams/{examId}/tools/{toolId}", server.handleDeleteTool).Methods("DELETE")

	// Chat
	apiRouter.HandleFunc("/exams/{examId}/chat/sessions", server.handleCreateChatSession).Methods("POST")
	apiRouter.HandleFunc("/exams/{examId}/chat/sessions", server.handleListChatSessions).Methods("GET")
	apiRouter.HandleFunc("/exams/{examId}/chat/sessions/{sessionId}", server.handleGetChatSession).Methods("GET")
	apiRouter.HandleFunc("/exams/{examId}/chat/sessions/{sessionId}", server.handleDeleteChatSession).Methods("DELETE")
	apiRouter.HandleFunc("/exams/{examId}/chat/sessions/{sessionId}/messages", server.handleSendMessage).Methods("POST")

	// Jobs
	apiRouter.HandleFunc("/jobs", server.handleListJobs).Methods("GET")
	apiRouter.HandleFunc("/jobs/{id}", server.handleGetJob).Methods("GET")
	apiRouter.HandleFunc("/jobs/{id}", server.handleCancelJob).Methods("DELETE")

	// Settings
	apiRouter.HandleFunc("/settings", server.handleGetSettings).Methods("GET")
	apiRouter.HandleFunc("/settings", server.handleUpdateSettings).Methods("PATCH")

	// WebSocket
	apiRouter.HandleFunc("/socket", server.handleWebSocket).Methods("GET")
}

// Middleware

func (server *Server) requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		requestID := uuid.New().String()
		request.Header.Set("X-Request-ID", requestID)
		responseWriter.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(responseWriter, request)
	})
}

func (server *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		startTime := time.Now()
		next.ServeHTTP(responseWriter, request)
		// Log request
		slog.Info("Request processed",
			"method", request.Method,
			"uri", request.RequestURI,
			"duration", time.Since(startTime),
		)
	})
}

// Utility functions

func (server *Server) writeJSON(responseWriter http.ResponseWriter, statusCode int, data interface{}) {
	response := models.APIResponse{
		Data: data,
		Meta: models.Meta{
			Timestamp: time.Now().Format(time.RFC3339),
			RequestID: responseWriter.Header().Get("X-Request-ID"),
		},
	}
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(statusCode)
	// In production, handle JSON marshal errors
	_ = writeJSONResponse(responseWriter, response)
}

func (server *Server) writeError(responseWriter http.ResponseWriter, statusCode int, code, message string, details interface{}) {
	response := models.APIError{
		Error: models.ErrorDetails{
			Code:    code,
			Message: message,
			Details: details,
		},
		Meta: models.Meta{
			Timestamp: time.Now().Format(time.RFC3339),
			RequestID: responseWriter.Header().Get("X-Request-ID"),
		},
	}
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(statusCode)
	_ = writeJSONResponse(responseWriter, response)
}
