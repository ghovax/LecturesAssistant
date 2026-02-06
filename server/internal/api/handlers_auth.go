package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// handleAuthSetup allows setting the initial password if not already set
func (server *Server) handleAuthSetup(responseWriter http.ResponseWriter, request *http.Request) {
	var setupRequest struct {
		Password string `json:"password"`
	}

	if decodeError := json.NewDecoder(request.Body).Decode(&setupRequest); decodeError != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	if len(setupRequest.Password) < 8 {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Password must be at least 8 characters", nil)
		return
	}

	// Check if password already set in settings
	var existingPassword sql.NullString
	databaseError := server.database.QueryRow("SELECT value FROM settings WHERE key = 'admin_password_hash'").Scan(&existingPassword)
	if databaseError == nil && existingPassword.Valid {
		server.writeError(responseWriter, http.StatusForbidden, "ALREADY_CONFIGURED", "Password has already been set", nil)
		return
	}

	passwordHash, hashError := bcrypt.GenerateFromPassword([]byte(setupRequest.Password), bcrypt.DefaultCost)
	if hashError != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "AUTH_ERROR", "Failed to hash password", nil)
		return
	}

	_, databaseError = server.database.Exec(`
		INSERT INTO settings (key, value, updated_at)
		VALUES ('admin_password_hash', ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at
	`, string(passwordHash), time.Now())

	if databaseError != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to save password", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusOK, map[string]string{"message": "Password set successfully"})
}

// handleAuthLogin authenticates user and creates a session
func (server *Server) handleAuthLogin(responseWriter http.ResponseWriter, request *http.Request) {
	// Rate Limiting
	clientIP := request.RemoteAddr
	server.loginAttemptsMutex.Lock()
	attempts := server.loginAttempts[clientIP]
	currentTime := time.Now()

	// Clean old attempts
	var validAttempts []time.Time
	for _, attemptTime := range attempts {
		if currentTime.Sub(attemptTime) < time.Hour {
			validAttempts = append(validAttempts, attemptTime)
		}
	}

	limit := server.configuration.Safety.MaximumLoginAttempts
	if limit <= 0 {
		limit = 1000 // Sane high default if not configured
	}

	if len(validAttempts) >= limit {
		server.loginAttemptsMutex.Unlock()
		server.writeError(responseWriter, http.StatusTooManyRequests, "RATE_LIMIT", "Too many login attempts. Please try again later.", nil)
		return
	}

	server.loginAttempts[clientIP] = append(validAttempts, currentTime)
	server.loginAttemptsMutex.Unlock()

	var loginRequest struct {
		Password string `json:"password"`
	}

	if decodeError := json.NewDecoder(request.Body).Decode(&loginRequest); decodeError != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	var storedHash string
	databaseError := server.database.QueryRow("SELECT value FROM settings WHERE key = 'admin_password_hash'").Scan(&storedHash)
	if databaseError == sql.ErrNoRows {
		server.writeError(responseWriter, http.StatusForbidden, "SETUP_REQUIRED", "Password has not been set yet", nil)
		return
	}

	if passwordMatchError := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(loginRequest.Password)); passwordMatchError != nil {
		server.writeError(responseWriter, http.StatusUnauthorized, "AUTH_ERROR", "Invalid password", nil)
		return
	}

	// Create session (without redundant password_hash)
	sessionID := uuid.New().String()
	expiresAt := time.Now().Add(time.Duration(server.configuration.Security.Auth.SessionTimeoutHours) * time.Hour)

	_, databaseError = server.database.Exec(`
		INSERT INTO auth_sessions (id, created_at, last_activity, expires_at)
		VALUES (?, ?, ?, ?)
	`, sessionID, time.Now(), time.Now(), expiresAt)

	if databaseError != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to create session", nil)
		return
	}

	// Reset attempts on success
	server.loginAttemptsMutex.Lock()
	delete(server.loginAttempts, clientIP)
	server.loginAttemptsMutex.Unlock()

	// Set cookie
	http.SetCookie(responseWriter, &http.Cookie{
		Name:     "session_token",
		Value:    sessionID,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   server.configuration.Security.Auth.RequireHTTPS,
		SameSite: http.SameSiteLaxMode,
	})

	server.writeJSON(responseWriter, http.StatusOK, map[string]any{
		"token":      sessionID,
		"expires_at": expiresAt.Format(time.RFC3339),
	})
}

// handleAuthLogout invalidates current session
func (server *Server) handleAuthLogout(responseWriter http.ResponseWriter, request *http.Request) {
	sessionToken := server.getSessionToken(request)
	if sessionToken != "" {
		server.database.Exec("DELETE FROM auth_sessions WHERE id = ?", sessionToken)
	}

	// Clear cookie
	http.SetCookie(responseWriter, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		Expires:  time.Now().Add(-1 * time.Hour),
		HttpOnly: true,
	})

	server.writeJSON(responseWriter, http.StatusOK, map[string]string{"message": "Logged out successfully"})
}

// handleAuthStatus checks if current request is authenticated
func (server *Server) handleAuthStatus(responseWriter http.ResponseWriter, request *http.Request) {
	sessionToken := server.getSessionToken(request)
	if sessionToken == "" {
		server.writeJSON(responseWriter, http.StatusOK, map[string]any{"authenticated": false})
		return
	}

	var expiresAt time.Time
	databaseError := server.database.QueryRow("SELECT expires_at FROM auth_sessions WHERE id = ?", sessionToken).Scan(&expiresAt)
	if databaseError != nil || time.Now().After(expiresAt) {
		server.writeJSON(responseWriter, http.StatusOK, map[string]any{"authenticated": false})
		return
	}

	server.writeJSON(responseWriter, http.StatusOK, map[string]any{"authenticated": true, "expires_at": expiresAt.Format(time.RFC3339)})
}
