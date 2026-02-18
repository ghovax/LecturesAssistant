package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"lectures/internal/configuration"
	"lectures/internal/jobs"
	"lectures/internal/llm"
	"lectures/internal/markdown"
	"lectures/internal/models"
	"lectures/internal/prompts"
	"lectures/internal/tools"

	"github.com/gorilla/mux"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

// Server represents the API server
type Server struct {
	configuration     *configuration.Configuration
	database          *sql.DB
	jobQueue          *jobs.Queue
	router            *mux.Router
	wsHub             *Hub
	llmProvider       llm.Provider
	promptManager     *prompts.Manager
	toolGenerator     *tools.ToolGenerator
	markdownConverter markdown.MarkdownConverter
	// Security
	loginAttempts      map[string][]time.Time
	loginAttemptsMutex sync.Mutex
}

// NewServer creates a new API server
func NewServer(configuration *configuration.Configuration, database *sql.DB, jobQueue *jobs.Queue, llmProvider llm.Provider, promptManager *prompts.Manager, toolGenerator *tools.ToolGenerator, markdownConverter markdown.MarkdownConverter) *Server {
	server := &Server{
		configuration:     configuration,
		database:          database,
		jobQueue:          jobQueue,
		router:            mux.NewRouter(),
		wsHub:             NewHub(),
		llmProvider:       llmProvider,
		promptManager:     promptManager,
		toolGenerator:     toolGenerator,
		markdownConverter: markdownConverter,
		loginAttempts:     make(map[string][]time.Time),
	}

	go server.wsHub.Run()
	server.StartStagingCleanupWorker()
	server.loadSettingsFromDatabase()
	server.setupRoutes()
	return server
}

// loadSettingsFromDatabase reads settings from the database and updates the in-memory configuration
func (server *Server) loadSettingsFromDatabase() {
	rows, err := server.database.Query("SELECT key, value FROM settings")
	if err != nil {
		slog.Warn("Failed to query settings from database", "error", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var key, valueJSON string
		if err := rows.Scan(&key, &valueJSON); err != nil {
			continue
		}

		valueBytes := []byte(valueJSON)
		switch key {
		case "llm":
			json.Unmarshal(valueBytes, &server.configuration.LLM)
		case "transcription":
			json.Unmarshal(valueBytes, &server.configuration.Transcription)
		case "documents":
			json.Unmarshal(valueBytes, &server.configuration.Documents)
		case "safety":
			json.Unmarshal(valueBytes, &server.configuration.Safety)
		case "uploads":
			json.Unmarshal(valueBytes, &server.configuration.Uploads)
		case "providers":
			if err := json.Unmarshal(valueBytes, &server.configuration.Providers); err == nil {
				// Update OpenRouter API Key in the running provider
				if routingProvider, ok := server.llmProvider.(*llm.RoutingProvider); ok {
					if openRouterProvider, ok := routingProvider.GetProvider("openrouter").(*llm.OpenRouterProvider); ok {
						openRouterProvider.SetAPIKey(server.configuration.Providers.OpenRouter.APIKey)
					}
				}
			}
		}
	}
	slog.Info("Configurations recovered from database")
}

// syncConfigurationToDatabase flushes the current in-memory configuration to the database
func (server *Server) syncConfigurationToDatabase() {
	configs := map[string]any{
		"llm":           server.configuration.LLM,
		"transcription": server.configuration.Transcription,
		"documents":     server.configuration.Documents,
		"safety":        server.configuration.Safety,
		"providers":     server.configuration.Providers,
		"uploads":       server.configuration.Uploads,
	}

	for key, val := range configs {
		valueJSON, err := json.Marshal(val)
		if err != nil {
			continue
		}

		_, err = server.database.Exec(`
			INSERT INTO settings (key, value, updated_at)
			VALUES (?, ?, ?)
			ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at
		`, key, string(valueJSON), time.Now())

		if err != nil {
			slog.Error("Failed to sync config to database", "key", key, "error", err)
		}
	}
}

// Handler returns the HTTP handler
func (server *Server) Handler() http.Handler {
	return server.router
}

// Broadcast sends a message to a specific WebSocket channel
func (server *Server) Broadcast(channel string, msgType string, payload any) {
	server.wsHub.Broadcast(WSMessage{
		Type:      msgType,
		Channel:   channel,
		Payload:   payload,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}

// setupRoutes configures all API routes
func (server *Server) setupRoutes() {
	// Add global CORS middleware - must be first
	server.router.Use(server.corsMiddleware)

	// Explicitly handle OPTIONS for all routes globally to prevent 405
	server.router.PathPrefix("/").Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handled by corsMiddleware
	})

	// Public routes
	server.router.HandleFunc("/api/health", server.handleHealth).Methods("GET")
	server.router.HandleFunc("/api/auth/setup", server.handleAuthSetup).Methods("POST")
	server.router.HandleFunc("/api/auth/register", server.handleAuthRegister).Methods("POST")
	server.router.HandleFunc("/api/auth/login", server.handleAuthLogin).Methods("POST")
	server.router.HandleFunc("/api/auth/status", server.handleAuthStatus).Methods("GET")

	// API routes (with middleware)
	apiRouter := server.router.PathPrefix("/api").Subrouter()
	apiRouter.Use(server.requestIDMiddleware)
	apiRouter.Use(server.loggingMiddleware)
	apiRouter.Use(server.authMiddleware)

	// Auth (requires auth)
	apiRouter.HandleFunc("/auth/logout", server.handleAuthLogout).Methods("POST")
	apiRouter.HandleFunc("/auth/password", server.handleAuthChangePassword).Methods("PATCH")

	// Staged Upload Protocol
	apiRouter.HandleFunc("/uploads/prepare", server.handleUploadPrepare).Methods("POST")
	apiRouter.HandleFunc("/uploads/append", server.handleUploadAppend).Methods("POST")
	apiRouter.HandleFunc("/uploads/stage", server.handleUploadStage).Methods("POST")
	apiRouter.HandleFunc("/uploads/import", server.handleImport).Methods("POST")

	// Exams
	apiRouter.HandleFunc("/exams", server.handleCreateExam).Methods("POST")
	apiRouter.HandleFunc("/exams", server.handleListExams).Methods("GET")
	apiRouter.HandleFunc("/exams/details", server.handleGetExam).Methods("GET")
	apiRouter.HandleFunc("/exams", server.handleUpdateExam).Methods("PATCH")
	apiRouter.HandleFunc("/exams", server.handleDeleteExam).Methods("DELETE")
	apiRouter.HandleFunc("/exams/search", server.handleExamSearch).Methods("GET")
	apiRouter.HandleFunc("/exams/suggest", server.handleExamSuggest).Methods("POST")
	apiRouter.HandleFunc("/exams/concepts", server.handleGetExamConcepts).Methods("GET")

	// Lectures
	apiRouter.HandleFunc("/lectures", server.handleCreateLecture).Methods("POST")
	apiRouter.HandleFunc("/lectures", server.handleListLectures).Methods("GET")
	apiRouter.HandleFunc("/lectures/details", server.handleGetLecture).Methods("GET")
	apiRouter.HandleFunc("/lectures", server.handleUpdateLecture).Methods("PATCH")
	apiRouter.HandleFunc("/lectures", server.handleDeleteLecture).Methods("DELETE")
	apiRouter.HandleFunc("/lectures/retry-job", server.handleRetryLectureJob).Methods("POST")

	// Media (Listing/Ordering)
	apiRouter.HandleFunc("/media", server.handleListMedia).Methods("GET")
	apiRouter.HandleFunc("/media", server.handleDeleteMedia).Methods("DELETE")
	apiRouter.HandleFunc("/media/content", server.handleGetMediaContent).Methods("GET")

	// Transcripts
	apiRouter.HandleFunc("/transcripts", server.handleGetTranscript).Methods("GET")
	apiRouter.HandleFunc("/transcripts", server.handleUpdateTranscript).Methods("PATCH")
	apiRouter.HandleFunc("/transcripts/html", server.handleGetTranscriptHTML).Methods("GET")

	// Reference Documents (Listing/Meta)
	apiRouter.HandleFunc("/documents", server.handleListDocuments).Methods("GET")
	apiRouter.HandleFunc("/documents/details", server.handleGetDocument).Methods("GET")
	apiRouter.HandleFunc("/documents", server.handleDeleteDocument).Methods("DELETE")
	apiRouter.HandleFunc("/documents/pages", server.handleGetDocumentPages).Methods("GET")
	apiRouter.HandleFunc("/documents/pages/image", server.handleGetPageImage).Methods("GET")
	apiRouter.HandleFunc("/documents/pages/html", server.handleGetPageHTML).Methods("GET")

	// Tools
	apiRouter.HandleFunc("/tools", server.handleCreateTool).Methods("POST")
	apiRouter.HandleFunc("/tools", server.handleListTools).Methods("GET")
	apiRouter.HandleFunc("/tools/details", server.handleGetTool).Methods("GET")
	apiRouter.HandleFunc("/tools/details", server.handleUpdateTool).Methods("PATCH")
	apiRouter.HandleFunc("/tools/html", server.handleGetToolHTML).Methods("GET")
	apiRouter.HandleFunc("/tools", server.handleDeleteTool).Methods("DELETE")
	apiRouter.HandleFunc("/tools/export", server.handleExportTool).Methods("POST")
	apiRouter.HandleFunc("/transcripts/export", server.handleExportTranscript).Methods("POST")
	apiRouter.HandleFunc("/documents/export", server.handleExportDocument).Methods("POST")
	apiRouter.HandleFunc("/exports/download", server.handleDownloadExport).Methods("GET")

	// Chat
	apiRouter.HandleFunc("/chat/sessions", server.handleCreateChatSession).Methods("POST")
	apiRouter.HandleFunc("/chat/sessions", server.handleListChatSessions).Methods("GET")
	apiRouter.HandleFunc("/chat/sessions/details", server.handleGetChatSession).Methods("GET")
	apiRouter.HandleFunc("/chat/sessions/context", server.handleUpdateChatContext).Methods("PATCH")
	apiRouter.HandleFunc("/chat/sessions", server.handleDeleteChatSession).Methods("DELETE")
	apiRouter.HandleFunc("/chat/messages", server.handleSendMessage).Methods("POST")

	// Jobs
	apiRouter.HandleFunc("/jobs", server.handleListJobs).Methods("GET")
	apiRouter.HandleFunc("/jobs/details", server.handleGetJob).Methods("GET")
	apiRouter.HandleFunc("/jobs", server.handleCancelJob).Methods("DELETE")

	// System
	apiRouter.HandleFunc("/system/backup", server.handleBackupDatabase).Methods("GET")
	apiRouter.HandleFunc("/system/restore", server.handleRestoreDatabase).Methods("POST")

	// Settings
	apiRouter.HandleFunc("/settings", server.handleGetSettings).Methods("GET")
	apiRouter.HandleFunc("/settings", server.handleUpdateSettings).Methods("PATCH")

	// WebSocket
	apiRouter.HandleFunc("/socket", server.handleWebSocket).Methods("GET")

	// Static Frontend Serving (if configured)
	if server.configuration.Storage.WebDirectory != "" {
		if _, err := os.Stat(server.configuration.Storage.WebDirectory); err == nil {
			slog.Info("Serving static frontend from", "directory", server.configuration.Storage.WebDirectory)
			server.router.PathPrefix("/").Handler(server.spaHandler(server.configuration.Storage.WebDirectory))
		} else {
			slog.Warn("Configured WebDirectory does not exist", "directory", server.configuration.Storage.WebDirectory)
		}
	}
}

// spaHandler serves static files with fallback to index.html for SPA routing
func (server *Server) spaHandler(webDir string) http.Handler {
	fileServer := http.FileServer(http.Dir(webDir))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip API routes - they're already handled by more specific routes
		if strings.HasPrefix(r.URL.Path, "/api") {
			http.NotFound(w, r)
			return
		}

		// Try to open the requested file
		path := filepath.Join(webDir, r.URL.Path)
		if _, err := os.Stat(path); os.IsNotExist(err) || r.URL.Path == "/" {
			// File doesn't exist or root - serve index.html for SPA routing
			r.URL.Path = "/"
			// Use http.ServeFile to ensure index.html is served from the directory
			http.ServeFile(w, r, filepath.Join(webDir, "index.html"))
			return
		}

		fileServer.ServeHTTP(w, r)
	})
}

// Middleware

func (server *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		origin := request.Header.Get("Origin")

		if origin != "" {
			responseWriter.Header().Set("Access-Control-Allow-Origin", origin)
			responseWriter.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, PATCH")
			responseWriter.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Requested-With")
			responseWriter.Header().Set("Access-Control-Expose-Headers", "Content-Disposition")
			responseWriter.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		if request.Method == "OPTIONS" {
			responseWriter.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(responseWriter, request)
	})
}

func (server *Server) requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		requestID, _ := gonanoid.New()
		request.Header.Set("X-Request-ID", requestID)
		responseWriter.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(responseWriter, request)
	})
}

func (server *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		startTime := time.Now()
		next.ServeHTTP(responseWriter, request)

		// Strip query parameters for logging to prevent leaking sensitive info
		uri := request.RequestURI
		if questionMarkIndex := strings.Index(uri, "?"); questionMarkIndex != -1 {
			uri = uri[:questionMarkIndex]
		}

		// Log request
		slog.Debug("Request processed",
			"method", request.Method,
			"uri", uri,
			"duration", time.Since(startTime),
		)
	})
}

type contextKey string

const userIDKey contextKey = "user_id"

func (server *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		// Skip authentication for OPTIONS requests (preflight)
		if request.Method == "OPTIONS" {
			next.ServeHTTP(responseWriter, request)
			return
		}

		// CSRF Protection: Require custom header AND Origin validation for state-changing methods
		if request.Method == "POST" || request.Method == "PATCH" || request.Method == "DELETE" {
			// Check X-Requested-With header (set by XMLHttpRequest/fetch)
			if request.Header.Get("X-Requested-With") == "" {
				server.writeError(responseWriter, http.StatusForbidden, "CSRF_ERROR", "X-Requested-With header is required", nil)
				return
			}

			// Validate Origin header to prevent cross-site requests
			origin := request.Header.Get("Origin")
			if origin != "" {
				// Parse origin URL to validate it's from an allowed source
				originURL, err := parseURL(origin)
				if err == nil {
					// Check if origin is localhost/127.0.0.1 (development) or matches host header (production)
					host := request.Host
					isLocalhost := strings.Contains(originURL.Host, "localhost") || strings.Contains(originURL.Host, "127.0.0.1")
					isSameHost := originURL.Host == host || originURL.Host+":80" == host || originURL.Host+":443" == host

					if !isLocalhost && !isSameHost {
						slog.Warn("CSRF: Origin header mismatch", "origin", origin, "host", host)
						server.writeError(responseWriter, http.StatusForbidden, "CSRF_ERROR", "Origin header mismatch", nil)
						return
					}
				}
			}
		}

		sessionToken := server.getSessionToken(request)
		if sessionToken == "" {
			server.writeError(responseWriter, http.StatusUnauthorized, "AUTHENTICATION_ERROR", "Authentication required", nil)
			return
		}

		var userID string
		var expiresAt time.Time
		databaseError := server.database.QueryRow("SELECT user_id, expires_at FROM auth_sessions WHERE id = ?", sessionToken).Scan(&userID, &expiresAt)
		if databaseError != nil {
			server.writeError(responseWriter, http.StatusUnauthorized, "AUTHENTICATION_ERROR", "Invalid session", nil)
			return
		}

		if time.Now().After(expiresAt) {
			server.writeError(responseWriter, http.StatusUnauthorized, "AUTHENTICATION_ERROR", "Session expired", nil)
			return
		}

		// Update last activity
		_, _ = server.database.Exec("UPDATE auth_sessions SET last_activity = ? WHERE id = ?", time.Now(), sessionToken)

		// Inject user_id into context
		requestContext := context.WithValue(request.Context(), userIDKey, userID)
		next.ServeHTTP(responseWriter, request.WithContext(requestContext))
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

func (server *Server) getSessionToken(request *http.Request) string {
	// 1. Try cookie first (most secure, not logged)
	cookie, err := request.Cookie("session_token")
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}

	// 2. Try Authorization header (also secure)
	authHeader := request.Header.Get("Authorization")
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		return authHeader[7:]
	}

	// 3. Try query parameter (least secure - may be logged, deprecated but kept for WebSocket compatibility)
	if token := request.URL.Query().Get("session_token"); token != "" {
		// Log warning for security auditing (token exposure in URL)
		slog.Warn("Session token provided via query parameter - consider using cookies or Authorization header",
			"path", request.URL.Path, "method", request.Method)
		return token
	}

	return ""
}

func (server *Server) getUserID(request *http.Request) string {
	if userID, ok := request.Context().Value(userIDKey).(string); ok {
		return userID
	}
	return ""
}

// parseURL is a helper function to parse URL strings safely
func parseURL(rawURL string) (*url.URL, error) {
	// Add scheme if missing for valid parsing
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "http://" + rawURL
	}
	return url.Parse(rawURL)
}
