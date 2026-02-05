package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"lectures/internal/llm"
	"lectures/internal/models"
	"lectures/internal/prompts"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// handleCreateChatSession creates a new chat session for an exam
func (server *Server) handleCreateChatSession(responseWriter http.ResponseWriter, request *http.Request) {
	pathVariables := mux.Vars(request)
	examIdentifier := pathVariables["examId"]

	var createSessionRequest struct {
		Title string `json:"title"`
	}

	if err := json.NewDecoder(request.Body).Decode(&createSessionRequest); err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	// Verify exam exists
	var examExists bool
	err := server.database.QueryRow("SELECT EXISTS(SELECT 1 FROM exams WHERE id = ?)", examIdentifier).Scan(&examExists)
	if err != nil || !examExists {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Exam not found", nil)
		return
	}

	session := models.ChatSession{
		ID:        uuid.New().String(),
		ExamID:    examIdentifier,
		Title:     createSessionRequest.Title,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	transaction, err := server.database.Begin()
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to start transaction", nil)
		return
	}
	defer transaction.Rollback()

	_, err = transaction.Exec(`
		INSERT INTO chat_sessions (id, exam_id, title, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, session.ID, session.ExamID, session.Title, session.CreatedAt, session.UpdatedAt)

	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to create chat session", nil)
		return
	}

	// Initialize empty context config
	_, err = transaction.Exec(`
		INSERT INTO chat_context_configuration (session_id, included_lecture_ids, included_tool_ids)
		VALUES (?, ?, ?)
	`, session.ID, "[]", "[]")

	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to initialize chat context", nil)
		return
	}

	if err := transaction.Commit(); err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to commit transaction", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusCreated, session)
}

// handleListChatSessions lists all chat sessions for an exam
func (server *Server) handleListChatSessions(responseWriter http.ResponseWriter, request *http.Request) {
	pathVariables := mux.Vars(request)
	examIdentifier := pathVariables["examId"]

	rows, err := server.database.Query(`
		SELECT id, exam_id, title, created_at, updated_at
		FROM chat_sessions
		WHERE exam_id = ?
		ORDER BY updated_at DESC
	`, examIdentifier)
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to list chat sessions", nil)
		return
	}
	defer rows.Close()

	var sessions []models.ChatSession
	for rows.Next() {
		var session models.ChatSession
		if err := rows.Scan(&session.ID, &session.ExamID, &session.Title, &session.CreatedAt, &session.UpdatedAt); err != nil {
			continue
		}
		sessions = append(sessions, session)
	}

	server.writeJSON(responseWriter, http.StatusOK, sessions)
}

// handleGetChatSession retrieves a specific session and its messages
func (server *Server) handleGetChatSession(responseWriter http.ResponseWriter, request *http.Request) {
	pathVariables := mux.Vars(request)
	sessionIdentifier := pathVariables["sessionId"]

	var session models.ChatSession
	err := server.database.QueryRow(`
		SELECT id, exam_id, title, created_at, updated_at
		FROM chat_sessions
		WHERE id = ?
	`, sessionIdentifier).Scan(&session.ID, &session.ExamID, &session.Title, &session.CreatedAt, &session.UpdatedAt)

	if err == sql.ErrNoRows {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Chat session not found", nil)
		return
	}
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get chat session", nil)
		return
	}

	// Get context config
	var includedLectureIDsJSON, includedToolIDsJSON string
	err = server.database.QueryRow(`
		SELECT included_lecture_ids, included_tool_ids 
		FROM chat_context_configuration 
		WHERE session_id = ?
	`, sessionIdentifier).Scan(&includedLectureIDsJSON, &includedToolIDsJSON)

	var includedLectureIDs, includedToolIDs []string
	if err == nil {
		json.Unmarshal([]byte(includedLectureIDsJSON), &includedLectureIDs)
		json.Unmarshal([]byte(includedToolIDsJSON), &includedToolIDs)
	}

	// Get messages
	rows, err := server.database.Query(`
		SELECT id, session_id, role, content, model_used, created_at
		FROM chat_messages
		WHERE session_id = ?
		ORDER BY created_at ASC
	`, sessionIdentifier)
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get messages", nil)
		return
	}
	defer rows.Close()

	var messages []models.ChatMessage
	for rows.Next() {
		var message models.ChatMessage
		if err := rows.Scan(&message.ID, &message.SessionID, &message.Role, &message.Content, &message.ModelUsed, &message.CreatedAt); err != nil {
			continue
		}
		messages = append(messages, message)
	}

	server.writeJSON(responseWriter, http.StatusOK, map[string]any{
		"session": session,
		"context": map[string]any{
			"included_lecture_ids": includedLectureIDs,
			"included_tool_ids":    includedToolIDs,
		},
		"messages": messages,
	})
}

// handleDeleteChatSession deletes a chat session
func (server *Server) handleDeleteChatSession(responseWriter http.ResponseWriter, request *http.Request) {
	pathVariables := mux.Vars(request)
	sessionIdentifier := pathVariables["sessionId"]

	result, err := server.database.Exec("DELETE FROM chat_sessions WHERE id = ?", sessionIdentifier)
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to delete chat session", nil)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Chat session not found", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusOK, map[string]string{"message": "Chat session deleted successfully"})
}

// handleUpdateChatContext updates which materials are included in the chat session
func (server *Server) handleUpdateChatContext(responseWriter http.ResponseWriter, request *http.Request) {
	pathVariables := mux.Vars(request)
	sessionIdentifier := pathVariables["sessionId"]

	var updateContextRequest struct {
		IncludedLectureIDs []string `json:"included_lecture_ids"`
		IncludedToolIDs    []string `json:"included_tool_ids"`
	}

	if err := json.NewDecoder(request.Body).Decode(&updateContextRequest); err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	lectureIDsJSON, _ := json.Marshal(updateContextRequest.IncludedLectureIDs)
	toolIDsJSON, _ := json.Marshal(updateContextRequest.IncludedToolIDs)

	_, err := server.database.Exec(`
		UPDATE chat_context_config
		SET included_lecture_ids = ?, included_tool_ids = ?
		WHERE session_id = ?
	`, string(lectureIDsJSON), string(toolIDsJSON), sessionIdentifier)

	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to update chat context", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusOK, map[string]string{"message": "Chat context updated successfully"})
}

// handleSendMessage adds a user message and triggers the AI response
func (server *Server) handleSendMessage(responseWriter http.ResponseWriter, request *http.Request) {
	pathVariables := mux.Vars(request)
	sessionIdentifier := pathVariables["sessionId"]

	var sendMessageRequest struct {
		Content string `json:"content"`
	}

	if err := json.NewDecoder(request.Body).Decode(&sendMessageRequest); err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	if sendMessageRequest.Content == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Content is required", nil)
		return
	}

	// 1. Verify session and save user message
	var session models.ChatSession
	err := server.database.QueryRow("SELECT id, exam_id FROM chat_sessions WHERE id = ?", sessionIdentifier).Scan(&session.ID, &session.ExamID)
	if err != nil {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Chat session not found", nil)
		return
	}

	userMessage := models.ChatMessage{
		ID:        uuid.New().String(),
		SessionID: sessionIdentifier,
		Role:      "user",
		Content:   sendMessageRequest.Content,
		CreatedAt: time.Now(),
	}

	_, err = server.database.Exec(`
		INSERT INTO chat_messages (id, session_id, role, content, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, userMessage.ID, userMessage.SessionID, userMessage.Role, userMessage.Content, userMessage.CreatedAt)

	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to save user message", nil)
		return
	}

	// 2. Get history and context for LLM
	messages := server.getChatHistory(sessionIdentifier)
	lectureContext := server.getLectureContext(sessionIdentifier)

	// 3. Trigger async AI response
	go server.processAIResponse(sessionIdentifier, messages, lectureContext)

	server.writeJSON(responseWriter, http.StatusAccepted, userMessage)
}

func (server *Server) getChatHistory(sessionID string) []llm.Message {
	rows, err := server.database.Query(`
		SELECT role, content FROM chat_messages 
		WHERE session_id = ? 
		ORDER BY created_at ASC
	`, sessionID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var messages []llm.Message
	for rows.Next() {
		var role, content string
		if err := rows.Scan(&role, &content); err == nil {
			messages = append(messages, llm.Message{
				Role: role,
				Content: []llm.ContentPart{
					{Type: "text", Text: content},
				},
			})
		}
	}
	return messages
}

func (server *Server) getLectureContext(sessionID string) string {
	var includedLectureIDsJSON string
	err := server.database.QueryRow("SELECT included_lecture_ids FROM chat_context_configuration WHERE session_id = ?", sessionID).Scan(&includedLectureIDsJSON)
	if err != nil {
		return ""
	}

	var includedLectureIDs []string
	json.Unmarshal([]byte(includedLectureIDsJSON), &includedLectureIDs)

	if len(includedLectureIDs) == 0 {
		return ""
	}

	var contextBuilder strings.Builder
	for _, lectureID := range includedLectureIDs {
		var title string
		server.database.QueryRow("SELECT title FROM lectures WHERE id = ?", lectureID).Scan(&title)

		contextBuilder.WriteString(fmt.Sprintf("\n# Document: %s\n", title))

		// Add transcript
		rows, err := server.database.Query(`
			SELECT text FROM transcript_segments 
			WHERE transcript_id = (SELECT id FROM transcripts WHERE lecture_id = ?)
			ORDER BY start_millisecond ASC
		`, lectureID)
		if err == nil {
			for rows.Next() {
				var text string
				if err := rows.Scan(&text); err == nil {
					contextBuilder.WriteString(text + " ")
				}
			}
			rows.Close()
		}

		// Add reference documents
		docRows, err := server.database.Query(`
			SELECT rd.title, rp.page_number, rp.extracted_text
			FROM reference_documents rd
			JOIN reference_pages rp ON rd.id = rp.document_id
			WHERE rd.lecture_id = ?
			ORDER BY rd.id, rp.page_number ASC
		`, lectureID)
		if err == nil {
			currentDocTitle := ""
			for docRows.Next() {
				var title, text string
				var pageNumber int
				if err := docRows.Scan(&title, &pageNumber, &text); err == nil {
					if title != currentDocTitle {
						contextBuilder.WriteString(fmt.Sprintf("\n## Reference File: %s\n", title))
						currentDocTitle = title
					}
					contextBuilder.WriteString(fmt.Sprintf("\n### Page %d\n%s\n", pageNumber, text))
				}
			}
			docRows.Close()
		}
	}

	return contextBuilder.String()
}

func (server *Server) processAIResponse(sessionID string, history []llm.Message, lectureContext string) {
	// Prepare system prompt
	systemPrompt, err := server.promptManager.GetPrompt(prompts.PromptReadingAssistantMultiChat, map[string]string{
		"latex_instructions": "", // Optional
	})
	if err != nil {
		slog.Error("Failed to load system prompt", "error", err)
		return
	}

	// Add lecture context to system prompt or first message
	if lectureContext != "" {
		systemPrompt += "\n\nContext from lectures:\n" + lectureContext
	}

	fullMessages := append([]llm.Message{
		{Role: "system", Content: []llm.ContentPart{{Type: "text", Text: systemPrompt}}},
	}, history...)

	responseChannel, err := server.llmProvider.Chat(context.Background(), llm.ChatRequest{
		Model:    server.configuration.LLM.OpenRouter.DefaultModel,
		Messages: fullMessages,
		Stream:   true,
	})

	if err != nil {
		slog.Error("LLM chat failed", "error", err)
		return
	}

	var completeResponseBuilder strings.Builder
	for chunk := range responseChannel {
		if chunk.Error != nil {
			slog.Error("LLM stream error", "error", chunk.Error)
			break
		}

		completeResponseBuilder.WriteString(chunk.Text)

		// Broadcast token via WebSocket
		server.wsHub.Broadcast(WSMessage{
			Type:    "chat:token",
			Channel: "chat:" + sessionID,
			Payload: map[string]string{
				"token":            chunk.Text,
				"accumulated_text": completeResponseBuilder.String(),
			},
			Timestamp: time.Now().Format(time.RFC3339),
		})
	}

	// Save complete response
	assistantMessage := models.ChatMessage{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Role:      "assistant",
		Content:   completeResponseBuilder.String(),
		ModelUsed: server.configuration.LLM.OpenRouter.DefaultModel,
		CreatedAt: time.Now(),
	}

	_, err = server.database.Exec(`
		INSERT INTO chat_messages (id, session_id, role, content, model_used, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, assistantMessage.ID, assistantMessage.SessionID, assistantMessage.Role, assistantMessage.Content, assistantMessage.ModelUsed, assistantMessage.CreatedAt)

	if err != nil {
		slog.Error("Failed to save assistant message", "error", err)
	}

	// Broadcast complete message
	server.wsHub.Broadcast(WSMessage{
		Type:      "chat:complete",
		Channel:   "chat:" + sessionID,
		Payload:   assistantMessage,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}
