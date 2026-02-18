package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"lectures/internal/configuration"
	"lectures/internal/llm"
	"lectures/internal/models"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"golang.org/x/crypto/bcrypt"
)

// handleAuthSetup allows creating the first user (admin) if no users exist
func (server *Server) handleAuthSetup(responseWriter http.ResponseWriter, request *http.Request) {
	var setupRequest struct {
		Username         string `json:"username"`
		Password         string `json:"password"`
		OpenRouterAPIKey string `json:"openrouter_api_key"`
	}

	if decodeError := json.NewDecoder(request.Body).Decode(&setupRequest); decodeError != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	if len(setupRequest.Password) < 8 {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Password must be at least 8 characters", nil)
		return
	}

	if setupRequest.Username == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Username is required", nil)
		return
	}

	if setupRequest.OpenRouterAPIKey == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "OpenRouter API Key is required", nil)
		return
	}

	// Check if any users already exist
	var userCount int
	databaseError := server.database.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	if databaseError != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to check user existence", nil)
		return
	}

	if userCount > 0 {
		server.writeError(responseWriter, http.StatusForbidden, "ALREADY_INITIALIZED", "Initial setup has already been completed", nil)
		return
	}

	// Update configuration with the provided API key
	server.configuration.Providers.OpenRouter.APIKey = setupRequest.OpenRouterAPIKey
	if server.configuration.ConfigurationPath != "" {
		if err := configuration.Save(server.configuration, server.configuration.ConfigurationPath); err != nil {
			server.writeError(responseWriter, http.StatusInternalServerError, "CONFIGURATION_ERROR", "Failed to save configuration", nil)
			return
		}
	}

	// Update the running LLM provider with the new API key
	if routingProvider, ok := server.llmProvider.(*llm.RoutingProvider); ok {
		if openRouterProvider, ok := routingProvider.GetProvider("openrouter").(*llm.OpenRouterProvider); ok {
			openRouterProvider.SetAPIKey(setupRequest.OpenRouterAPIKey)
		}
	}

	// Persist providers configuration to database so it can be recovered even if YAML is lost
	providersJSON, _ := json.Marshal(server.configuration.Providers)
	_, _ = server.database.Exec(`
		INSERT INTO settings (key, value, updated_at)
		VALUES ('providers', ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at
	`, string(providersJSON), time.Now())

	passwordHash, passwordHashingError := bcrypt.GenerateFromPassword([]byte(setupRequest.Password), bcrypt.DefaultCost)
	if passwordHashingError != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "AUTHENTICATION_ERROR", "Failed to hash password", nil)
		return
	}

	userID, _ := gonanoid.New()
	_, databaseError = server.database.Exec(`
		INSERT INTO users (id, username, password_hash, role, created_at, updated_at)
		VALUES (?, ?, ?, 'admin', ?, ?)
	`, userID, setupRequest.Username, string(passwordHash), time.Now(), time.Now())

	if databaseError != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to create initial user", nil)
		return
	}

	// Create session for auto-login
	sessionID, _ := gonanoid.New()
	expiresAt := time.Now().Add(time.Duration(server.configuration.Security.Auth.SessionTimeoutHours) * time.Hour)

	_, databaseError = server.database.Exec(`
		INSERT INTO auth_sessions (id, user_id, created_at, last_activity, expires_at)
		VALUES (?, ?, ?, ?, ?)
	`, sessionID, userID, time.Now(), time.Now(), expiresAt)

	if databaseError != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to create session", nil)
		return
	}

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
		"user": map[string]string{
			"id":       userID,
			"username": setupRequest.Username,
			"role":     "admin",
		},
	})
}

// handleAuthRegister allows new users to create an account
func (server *Server) handleAuthRegister(responseWriter http.ResponseWriter, request *http.Request) {
	var registerRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if decodeError := json.NewDecoder(request.Body).Decode(&registerRequest); decodeError != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	if len(registerRequest.Password) < 8 {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Password must be at least 8 characters", nil)
		return
	}

	if registerRequest.Username == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Username is required", nil)
		return
	}

	// Check if setup has been completed
	var userCount int
	databaseError := server.database.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	if databaseError != nil || userCount == 0 {
		server.writeError(responseWriter, http.StatusForbidden, "NOT_INITIALIZED", "Initial setup must be completed first", nil)
		return
	}

	// Check if username is taken
	var exists bool
	server.database.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)", registerRequest.Username).Scan(&exists)
	if exists {
		server.writeError(responseWriter, http.StatusConflict, "USERNAME_TAKEN", "Username is already registered", nil)
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(registerRequest.Password), bcrypt.DefaultCost)
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "AUTHENTICATION_ERROR", "Failed to hash password", nil)
		return
	}

	userID, _ := gonanoid.New()
	_, err = server.database.Exec(`
		INSERT INTO users (id, username, password_hash, role, created_at, updated_at)
		VALUES (?, ?, ?, 'user', ?, ?)
	`, userID, registerRequest.Username, string(passwordHash), time.Now(), time.Now())

	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to create user", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusCreated, map[string]string{"message": "Account created successfully. You can now log in."})
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
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if decodeError := json.NewDecoder(request.Body).Decode(&loginRequest); decodeError != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	var user models.User
	databaseError := server.database.QueryRow("SELECT id, username, password_hash, role FROM users WHERE username = ?", loginRequest.Username).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role)
	if databaseError == sql.ErrNoRows {
		server.writeError(responseWriter, http.StatusUnauthorized, "AUTHENTICATION_ERROR", "Invalid username or password", nil)
		return
	}

	if passwordMatchError := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(loginRequest.Password)); passwordMatchError != nil {
		server.writeError(responseWriter, http.StatusUnauthorized, "AUTHENTICATION_ERROR", "Invalid username or password", nil)
		return
	}

	// Create session
	sessionID, _ := gonanoid.New()
	expiresAt := time.Now().Add(time.Duration(server.configuration.Security.Auth.SessionTimeoutHours) * time.Hour)

	_, databaseError = server.database.Exec(`
		INSERT INTO auth_sessions (id, user_id, created_at, last_activity, expires_at)
		VALUES (?, ?, ?, ?, ?)
	`, sessionID, user.ID, time.Now(), time.Now(), expiresAt)

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
		"user": map[string]string{
			"id":       user.ID,
			"username": user.Username,
			"role":     user.Role,
		},
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
	// Check if any users exist to determine if system is initialized
	var userCount int
	server.database.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	initialized := userCount > 0

	sessionToken := server.getSessionToken(request)
	if sessionToken == "" {
		server.writeJSON(responseWriter, http.StatusOK, map[string]any{
			"authenticated": false,
			"initialized":   initialized,
		})
		return
	}

	var userID, username, role string
	var expiresAt time.Time
	databaseError := server.database.QueryRow(`
		SELECT auth_sessions.expires_at, users.id, users.username, users.role
		FROM auth_sessions
		JOIN users ON auth_sessions.user_id = users.id
		WHERE auth_sessions.id = ?
	`, sessionToken).Scan(&expiresAt, &userID, &username, &role)

	if databaseError != nil || time.Now().After(expiresAt) {
		server.writeJSON(responseWriter, http.StatusOK, map[string]any{
			"authenticated": false,
			"initialized":   initialized,
		})
		return
	}

	server.writeJSON(responseWriter, http.StatusOK, map[string]any{
		"authenticated": true,
		"initialized":   initialized,
		"expires_at":    expiresAt.Format(time.RFC3339),
		"user": map[string]string{
			"id":       userID,
			"username": username,
			"role":     role,
		},
	})
}

// handleAuthChangePassword allows a user to change their password
func (server *Server) handleAuthChangePassword(responseWriter http.ResponseWriter, request *http.Request) {
	var passwordRequest struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}

	if decodeError := json.NewDecoder(request.Body).Decode(&passwordRequest); decodeError != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	if len(passwordRequest.NewPassword) < 8 {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "New password must be at least 8 characters", nil)
		return
	}

	userID := server.getUserID(request)

	var passwordHash string
	err := server.database.QueryRow("SELECT password_hash FROM users WHERE id = ?", userID).Scan(&passwordHash)
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get user details", nil)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(passwordRequest.CurrentPassword)); err != nil {
		server.writeError(responseWriter, http.StatusUnauthorized, "AUTHENTICATION_ERROR", "Invalid current password", nil)
		return
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(passwordRequest.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "AUTHENTICATION_ERROR", "Failed to hash new password", nil)
		return
	}

	_, err = server.database.Exec("UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?", string(newHash), time.Now(), userID)
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to update password", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusOK, map[string]string{"message": "Password updated successfully"})
}
