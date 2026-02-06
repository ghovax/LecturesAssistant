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
			"model":    server.configuration.LLM.Model,
			"language": server.configuration.LLM.Language,
		},
		"transcription": map[string]string{
			"provider": server.configuration.Transcription.Provider,
			"model":    server.configuration.Transcription.Model,
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

	// Whitelist of allowed settings keys
	allowedKeys := map[string]bool{
		"llm":           true,
		"transcription": true,
		"documents":     true,
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
	if settingValue, exists := updateSettingsRequest["llm"]; exists {
		if llmConfigurationMap, isMap := settingValue.(map[string]any); isMap {
			if llmProvider, isString := llmConfigurationMap["provider"].(string); isString {
				server.configuration.LLM.Provider = llmProvider
			}
			if llmModel, isString := llmConfigurationMap["model"].(string); isString {
				server.configuration.LLM.Model = llmModel
			}
			if languageCode, isString := llmConfigurationMap["language"].(string); isString {
				server.configuration.LLM.Language = languageCode
			}
		}
	}
	if settingValue, exists := updateSettingsRequest["transcription"]; exists {
		if transcriptionConfigurationMap, isMap := settingValue.(map[string]any); isMap {
			if transcriptionProvider, isString := transcriptionConfigurationMap["provider"].(string); isString {
				server.configuration.Transcription.Provider = transcriptionProvider
			}
			if transcriptionModel, isString := transcriptionConfigurationMap["model"].(string); isString {
				server.configuration.Transcription.Model = transcriptionModel
			}
		}
	}

	server.writeJSON(responseWriter, http.StatusOK, updateSettingsRequest)
}
