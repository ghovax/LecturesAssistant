package api

import (
	"database/sql"
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
	config   *config.Config
	db       *sql.DB
	jobQueue *jobs.Queue
	router   *mux.Router
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, db *sql.DB, jobQueue *jobs.Queue) *Server {
	s := &Server{
		config:   cfg,
		db:       db,
		jobQueue: jobQueue,
		router:   mux.NewRouter(),
	}

	s.setupRoutes()
	return s
}

// Handler returns the HTTP handler
func (s *Server) Handler() http.Handler {
	return s.router
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Health check (no auth required)
	s.router.HandleFunc("/api/health", s.handleHealth).Methods("GET")

	// API routes (with middleware)
	api := s.router.PathPrefix("/api").Subrouter()
	api.Use(s.requestIDMiddleware)
	api.Use(s.loggingMiddleware)

	// Exams
	api.HandleFunc("/exams", s.handleCreateExam).Methods("POST")
	api.HandleFunc("/exams", s.handleListExams).Methods("GET")
	api.HandleFunc("/exams/{id}", s.handleGetExam).Methods("GET")
	api.HandleFunc("/exams/{id}", s.handleUpdateExam).Methods("PATCH")
	api.HandleFunc("/exams/{id}", s.handleDeleteExam).Methods("DELETE")

	// Lectures
	api.HandleFunc("/exams/{exam_id}/lectures", s.handleCreateLecture).Methods("POST")
	api.HandleFunc("/exams/{exam_id}/lectures", s.handleListLectures).Methods("GET")
	api.HandleFunc("/exams/{exam_id}/lectures/{lecture_id}", s.handleGetLecture).Methods("GET")
	api.HandleFunc("/exams/{exam_id}/lectures/{lecture_id}", s.handleUpdateLecture).Methods("PATCH")
	api.HandleFunc("/exams/{exam_id}/lectures/{lecture_id}", s.handleDeleteLecture).Methods("DELETE")

	// Media
	api.HandleFunc("/exams/{exam_id}/lectures/{lecture_id}/media/upload", s.handleUploadMedia).Methods("POST")
	api.HandleFunc("/exams/{exam_id}/lectures/{lecture_id}/media", s.handleListMedia).Methods("GET")
	api.HandleFunc("/exams/{exam_id}/lectures/{lecture_id}/media/{media_id}", s.handleDeleteMedia).Methods("DELETE")

	// Transcripts
	api.HandleFunc("/exams/{exam_id}/lectures/{lecture_id}/transcribe", s.handleTranscribe).Methods("POST")
	api.HandleFunc("/exams/{exam_id}/lectures/{lecture_id}/transcript", s.handleGetTranscript).Methods("GET")

	// Reference Documents
	api.HandleFunc("/exams/{exam_id}/lectures/{lecture_id}/documents/upload", s.handleUploadDocument).Methods("POST")
	api.HandleFunc("/exams/{exam_id}/lectures/{lecture_id}/documents", s.handleListDocuments).Methods("GET")
	api.HandleFunc("/exams/{exam_id}/lectures/{lecture_id}/documents/{document_id}", s.handleGetDocument).Methods("GET")
	api.HandleFunc("/exams/{exam_id}/lectures/{lecture_id}/documents/{document_id}", s.handleDeleteDocument).Methods("DELETE")

	// Tools
	api.HandleFunc("/exams/{exam_id}/tools", s.handleCreateTool).Methods("POST")
	api.HandleFunc("/exams/{exam_id}/tools", s.handleListTools).Methods("GET")
	api.HandleFunc("/exams/{exam_id}/tools/{tool_id}", s.handleGetTool).Methods("GET")
	api.HandleFunc("/exams/{exam_id}/tools/{tool_id}", s.handleDeleteTool).Methods("DELETE")

	// Chat
	api.HandleFunc("/exams/{exam_id}/chat/sessions", s.handleCreateChatSession).Methods("POST")
	api.HandleFunc("/exams/{exam_id}/chat/sessions", s.handleListChatSessions).Methods("GET")
	api.HandleFunc("/exams/{exam_id}/chat/sessions/{session_id}", s.handleGetChatSession).Methods("GET")
	api.HandleFunc("/exams/{exam_id}/chat/sessions/{session_id}", s.handleDeleteChatSession).Methods("DELETE")
	api.HandleFunc("/exams/{exam_id}/chat/sessions/{session_id}/messages", s.handleSendMessage).Methods("POST")

	// Jobs
	api.HandleFunc("/jobs", s.handleListJobs).Methods("GET")
	api.HandleFunc("/jobs/{id}", s.handleGetJob).Methods("GET")
	api.HandleFunc("/jobs/{id}", s.handleCancelJob).Methods("DELETE")

	// Settings
	api.HandleFunc("/settings", s.handleGetSettings).Methods("GET")
	api.HandleFunc("/settings", s.handleUpdateSettings).Methods("PATCH")

	// WebSocket
	api.HandleFunc("/ws", s.handleWebSocket).Methods("GET")
}

// Middleware

func (s *Server) requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := uuid.New().String()
		r.Header.Set("X-Request-ID", requestID)
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r)
	})
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		// Log request
		// log.Printf("%s %s %s", r.Method, r.RequestURI, time.Since(start))
		_ = start // Suppress unused warning for now
	})
}

// Utility functions

func (s *Server) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	response := models.APIResponse{
		Data: data,
		Meta: models.Meta{
			Timestamp: time.Now().Format(time.RFC3339),
			RequestID: w.Header().Get("X-Request-ID"),
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// In production, handle JSON marshal errors
	_ = writeJSONResponse(w, response)
}

func (s *Server) writeError(w http.ResponseWriter, status int, code, message string, details interface{}) {
	response := models.APIError{
		Error: models.ErrorDetails{
			Code:    code,
			Message: message,
			Details: details,
		},
		Meta: models.Meta{
			Timestamp: time.Now().Format(time.RFC3339),
			RequestID: w.Header().Get("X-Request-ID"),
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = writeJSONResponse(w, response)
}
