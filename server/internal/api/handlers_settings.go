package api

import (
	"encoding/json"
	"net/http"
	"time"

	"lectures/internal/llm"
)

// handleGetSettings retrieves current application settings
func (server *Server) handleGetSettings(responseWriter http.ResponseWriter, request *http.Request) {
	// Prepare resolved models for display
	resolved := map[string]string{
		"recording_transcription": server.configuration.LLM.GetModelForTask("recording_transcription"),
		"documents_ingestion":     server.configuration.LLM.GetModelForTask("documents_ingestion"),
		"documents_matching":      server.configuration.LLM.GetModelForTask("documents_matching"),
		"outline_creation":        server.configuration.LLM.GetModelForTask("outline_creation"),
		"content_generation":      server.configuration.LLM.GetModelForTask("content_generation"),
		"content_verification":    server.configuration.LLM.GetModelForTask("content_verification"),
		"content_polishing":       server.configuration.LLM.GetModelForTask("content_polishing"),
	}

	server.writeJSON(responseWriter, http.StatusOK, map[string]any{
		"llm":             server.configuration.LLM,
		"transcription":   server.configuration.Transcription,
		"documents":       server.configuration.Documents,
		"safety":          server.configuration.Safety,
		"providers":       server.configuration.Providers,
		"resolved_models": resolved,
	})
}

// handleUpdateSettings updates user preferences and persists them
func (server *Server) handleUpdateSettings(responseWriter http.ResponseWriter, request *http.Request) {
	var updateSettingsRequest map[string]any
	if err := json.NewDecoder(request.Body).Decode(&updateSettingsRequest); err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	// Whitelist of allowed settings keys
	allowedKeys := map[string]bool{
		"llm":           true,
		"transcription": true,
		"documents":     true,
		"safety":        true,
		"theme":         true,
	}

	for key, value := range updateSettingsRequest {
		if !allowedKeys[key] {
			server.writeError(responseWriter, http.StatusForbidden, "FORBIDDEN_SETTING", "Setting key '"+key+"' is protected or invalid", nil)
			return
		}

		valueJSON, err := json.Marshal(value)
		if err != nil {
			continue
		}

		_, err = server.database.Exec(`
			INSERT INTO settings (key, value, updated_at)
			VALUES (?, ?, ?)
			ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at
		`, key, string(valueJSON), time.Now())

		if err != nil {
			server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to persist setting: "+key, nil)
			return
		}
	}

	// Update server.configuration in-memory to ensure immediate effect
	for key, value := range updateSettingsRequest {
		valueBytes, err := json.Marshal(value)
		if err != nil {
			continue
		}

		switch key {
		case "llm":
			json.Unmarshal(valueBytes, &server.configuration.LLM)
		case "transcription":
			json.Unmarshal(valueBytes, &server.configuration.Transcription)
		case "documents":
			json.Unmarshal(valueBytes, &server.configuration.Documents)
		case "safety":
			json.Unmarshal(valueBytes, &server.configuration.Safety)
		}
	}

	// If providers configuration was updated, reflect it in the running providers
	if providersValue, exists := updateSettingsRequest["providers"]; exists {
		if providersBytes, err := json.Marshal(providersValue); err == nil {
			json.Unmarshal(providersBytes, &server.configuration.Providers)

			// Update OpenRouter API Key if it was changed
			if routingProvider, ok := server.llmProvider.(*llm.RoutingProvider); ok {
				if openRouterProvider, ok := routingProvider.GetProvider("openrouter").(*llm.OpenRouterProvider); ok {
					openRouterProvider.SetAPIKey(server.configuration.Providers.OpenRouter.APIKey)
				}
			}
		}
	}

	server.writeJSON(responseWriter, http.StatusOK, updateSettingsRequest)
}
