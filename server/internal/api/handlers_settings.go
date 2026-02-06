package api

import (
	"encoding/json"
	"net/http"
	"time"
)

// handleGetSettings retrieves current application settings
func (server *Server) handleGetSettings(responseWriter http.ResponseWriter, request *http.Request) {
	server.writeJSON(responseWriter, http.StatusOK, map[string]any{
		"llm": map[string]string{
			"provider": server.configuration.LLM.Provider,
			"model":    server.configuration.LLM.OpenRouter.DefaultModel,
			"language": server.configuration.LLM.Language,
		},
		"transcription": map[string]string{
			"provider": server.configuration.Transcription.Provider,
		},
	})
}

// handleUpdateSettings updates user preferences and persists them
func (server *Server) handleUpdateSettings(responseWriter http.ResponseWriter, request *http.Request) {
	var updateSettingsRequest map[string]any
	if err := json.NewDecoder(request.Body).Decode(&updateSettingsRequest); err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	for key, value := range updateSettingsRequest {
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
	if value, ok := updateSettingsRequest["llm"]; ok {
		if llmMap, ok := value.(map[string]any); ok {
			if provider, ok := llmMap["provider"].(string); ok {
				server.configuration.LLM.Provider = provider
			}
			if language, ok := llmMap["language"].(string); ok {
				server.configuration.LLM.Language = language
			}
		}
	}
	if value, ok := updateSettingsRequest["transcription"]; ok {
		if transMap, ok := value.(map[string]any); ok {
			if provider, ok := transMap["provider"].(string); ok {
				server.configuration.Transcription.Provider = provider
			}
		}
	}

	server.writeJSON(responseWriter, http.StatusOK, updateSettingsRequest)
}
