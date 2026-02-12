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
	"lectures/internal/markdown"
	"lectures/internal/models"
	"lectures/internal/prompts"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

// handleCreateChatSession creates a new chat session for an exam
func (server *Server) handleCreateChatSession(responseWriter http.ResponseWriter, request *http.Request) {
	var createSessionRequest struct {
		ExamID string `json:"exam_id"`
		Title  string `json:"title"`
	}

	if decodeError := json.NewDecoder(request.Body).Decode(&createSessionRequest); decodeError != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	if createSessionRequest.ExamID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "exam_id is required", nil)
		return
	}

	userID := server.getUserID(request)

	// Verify exam exists and belongs to user
	var examExists bool
	databaseError := server.database.QueryRow("SELECT EXISTS(SELECT 1 FROM exams WHERE id = ? AND user_id = ?)", createSessionRequest.ExamID, userID).Scan(&examExists)
	if databaseError != nil || !examExists {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Exam not found", nil)
		return
	}

	sessionID, _ := gonanoid.New()
	session := models.ChatSession{
		ID:        sessionID,
		ExamID:    createSessionRequest.ExamID,
		Title:     createSessionRequest.Title,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	databaseTransaction, databaseError := server.database.Begin()
	if databaseError != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to start transaction", nil)
		return
	}
	defer databaseTransaction.Rollback()

	_, databaseError = databaseTransaction.Exec(`
		INSERT INTO chat_sessions (id, exam_id, title, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, session.ID, session.ExamID, session.Title, session.CreatedAt, session.UpdatedAt)

	if databaseError != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to create chat session", nil)
		return
	}

	// Initialize empty context configuration
	_, databaseError = databaseTransaction.Exec(`
		INSERT INTO chat_context_configuration (session_id, included_lecture_ids, included_tool_ids)
		VALUES (?, ?, ?)
	`, session.ID, "[]", "[]")

	if databaseError != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to initialize chat context", nil)
		return
	}

	if commitError := databaseTransaction.Commit(); commitError != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to commit transaction", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusCreated, session)
}

// handleListChatSessions lists all chat sessions for an exam (must belong to the user)
func (server *Server) handleListChatSessions(responseWriter http.ResponseWriter, request *http.Request) {
	examID := request.URL.Query().Get("exam_id")
	if examID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "exam_id is required", nil)
		return
	}

	userID := server.getUserID(request)

	sessionRows, databaseError := server.database.Query(`
		SELECT chat_sessions.id, chat_sessions.exam_id, chat_sessions.title, chat_sessions.created_at, chat_sessions.updated_at
		FROM chat_sessions
		JOIN exams ON chat_sessions.exam_id = exams.id
		WHERE chat_sessions.exam_id = ? AND exams.user_id = ?
		ORDER BY chat_sessions.updated_at DESC
	`, examID, userID)
	if databaseError != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to list chat sessions", nil)
		return
	}
	defer sessionRows.Close()

	var sessions []models.ChatSession
	for sessionRows.Next() {
		var session models.ChatSession
		if scanError := sessionRows.Scan(&session.ID, &session.ExamID, &session.Title, &session.CreatedAt, &session.UpdatedAt); scanError != nil {
			continue
		}
		sessions = append(sessions, session)
	}

	server.writeJSON(responseWriter, http.StatusOK, sessions)
}

// handleGetChatSession retrieves a specific session and its messages
func (server *Server) handleGetChatSession(responseWriter http.ResponseWriter, request *http.Request) {
	sessionID := request.URL.Query().Get("session_id")
	examID := request.URL.Query().Get("exam_id")

	if sessionID == "" || examID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "session_id and exam_id are required", nil)
		return
	}

	userID := server.getUserID(request)

	var session models.ChatSession
	databaseError := server.database.QueryRow(`
		SELECT chat_sessions.id, chat_sessions.exam_id, chat_sessions.title, chat_sessions.created_at, chat_sessions.updated_at
		FROM chat_sessions
		JOIN exams ON chat_sessions.exam_id = exams.id
		WHERE chat_sessions.id = ? AND chat_sessions.exam_id = ? AND exams.user_id = ?
	`, sessionID, examID, userID).Scan(&session.ID, &session.ExamID, &session.Title, &session.CreatedAt, &session.UpdatedAt)

	if databaseError == sql.ErrNoRows {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Chat session not found in this exam", nil)
		return
	}
	if databaseError != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get chat session", nil)
		return
	}

	// Get context configuration
	var includedLectureIDsJSON, usedLectureIDsJSON, includedToolIDsJSON string
	databaseError = server.database.QueryRow(`
		SELECT included_lecture_ids, used_lecture_ids, included_tool_ids 
		FROM chat_context_configuration 
		WHERE session_id = ?
	`, sessionID).Scan(&includedLectureIDsJSON, &usedLectureIDsJSON, &includedToolIDsJSON)

	var contextIncludedLectureIDs, contextUsedLectureIDs, contextIncludedToolIDs []string
	if databaseError == nil {
		json.Unmarshal([]byte(includedLectureIDsJSON), &contextIncludedLectureIDs)
		json.Unmarshal([]byte(usedLectureIDsJSON), &contextUsedLectureIDs)
		json.Unmarshal([]byte(includedToolIDsJSON), &contextIncludedToolIDs)
	}

	// Get messages
	messageRows, databaseError := server.database.Query(`
		SELECT id, session_id, role, content, model_used, input_tokens, output_tokens, estimated_cost, created_at
		FROM chat_messages
		WHERE session_id = ?
		ORDER BY created_at ASC
	`, sessionID)
	if databaseError != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get messages", nil)
		return
	}
	defer messageRows.Close()

	var messages []models.ChatMessage
	for messageRows.Next() {
		var message models.ChatMessage
		var modelUsed sql.NullString
		if scanError := messageRows.Scan(&message.ID, &message.SessionID, &message.Role, &message.Content, &modelUsed, &message.InputTokens, &message.OutputTokens, &message.EstimatedCost, &message.CreatedAt); scanError != nil {
			slog.Error("Failed to scan chat message", "sessionID", sessionID, "error", scanError)
			continue
		}

		if modelUsed.Valid {
			message.ModelUsed = modelUsed.String
		}

		// Convert content to HTML
		if message.Content != "" {
			convertedHTML, err := server.markdownConverter.MarkdownToHTML(message.Content)
			if err == nil {
				message.ContentHTML = convertedHTML
			} else {
				message.ContentHTML = message.Content
			}
		}

		messages = append(messages, message)
	}

	slog.Info("Retrieved chat messages", "sessionID", sessionID, "count", len(messages))

	server.writeJSON(responseWriter, http.StatusOK, map[string]any{
		"session": session,
		"context": map[string]any{
			"included_lecture_ids": contextIncludedLectureIDs,
			"used_lecture_ids":     contextUsedLectureIDs,
			"included_tool_ids":    contextIncludedToolIDs,
		},
		"messages": messages,
	})
}

// handleDeleteChatSession deletes a chat session
func (server *Server) handleDeleteChatSession(responseWriter http.ResponseWriter, request *http.Request) {
	var deleteRequest struct {
		SessionID string `json:"session_id"`
		ExamID    string `json:"exam_id"`
	}
	if err := json.NewDecoder(request.Body).Decode(&deleteRequest); err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid body", nil)
		return
	}

	if deleteRequest.SessionID == "" || deleteRequest.ExamID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "session_id and exam_id are required", nil)
		return
	}

	userID := server.getUserID(request)

	result, databaseError := server.database.Exec(`
		DELETE FROM chat_sessions 
		WHERE id = ? AND exam_id = ? AND EXISTS (
			SELECT 1 FROM exams WHERE id = ? AND user_id = ?
		)
	`, deleteRequest.SessionID, deleteRequest.ExamID, deleteRequest.ExamID, userID)
	if databaseError != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to delete chat session", nil)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Chat session not found in this exam", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusOK, map[string]string{"message": "Chat session deleted successfully"})
}

// handleUpdateChatContext updates which materials are included in the chat session
func (server *Server) handleUpdateChatContext(responseWriter http.ResponseWriter, request *http.Request) {
	var updateContextRequest struct {
		SessionID          string   `json:"session_id"`
		IncludedLectureIDs []string `json:"included_lecture_ids"`
		IncludedToolIDs    []string `json:"included_tool_ids"`
	}

	if decodeError := json.NewDecoder(request.Body).Decode(&updateContextRequest); decodeError != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	if updateContextRequest.SessionID == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "session_id is required", nil)
		return
	}

	userID := server.getUserID(request)

	// Verify session and get its exam_id and verify ownership
	var examID string
	err := server.database.QueryRow(`
		SELECT chat_sessions.exam_id FROM chat_sessions 
		JOIN exams ON chat_sessions.exam_id = exams.id
		WHERE chat_sessions.id = ? AND exams.user_id = ?
	`, updateContextRequest.SessionID, userID).Scan(&examID)
	if err != nil {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Chat session not found", nil)
		return
	}

	// Verify all lectures belong to this exam
	for _, lectureID := range updateContextRequest.IncludedLectureIDs {
		var lectureExamID string
		err := server.database.QueryRow("SELECT exam_id FROM lectures WHERE id = ?", lectureID).Scan(&lectureExamID)
		if err != nil || lectureExamID != examID {
			server.writeError(responseWriter, http.StatusBadRequest, "RESOURCE_VIOLATION", "Lecture "+lectureID+" does not belong to this exam", nil)
			return
		}
	}

	// Verify all tools belong to this exam
	for _, toolID := range updateContextRequest.IncludedToolIDs {
		var toolExamID string
		err := server.database.QueryRow("SELECT exam_id FROM tools WHERE id = ?", toolID).Scan(&toolExamID)
		if err != nil || toolExamID != examID {
			server.writeError(responseWriter, http.StatusBadRequest, "RESOURCE_VIOLATION", "Tool "+toolID+" does not belong to this exam", nil)
			return
		}
	}

	lectureIDsJSON, _ := json.Marshal(updateContextRequest.IncludedLectureIDs)
	toolIDsJSON, _ := json.Marshal(updateContextRequest.IncludedToolIDs)

	_, databaseError := server.database.Exec(`
		UPDATE chat_context_configuration
		SET included_lecture_ids = ?, included_tool_ids = ?
		WHERE session_id = ?
	`, string(lectureIDsJSON), string(toolIDsJSON), updateContextRequest.SessionID)

	if databaseError != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to update chat context", nil)
		return
	}

	server.writeJSON(responseWriter, http.StatusOK, map[string]string{"message": "Chat context updated successfully"})
}

// handleSendMessage adds a user message and triggers the AI response
func (server *Server) handleSendMessage(responseWriter http.ResponseWriter, request *http.Request) {
	var sendMessageRequest struct {
		SessionID string `json:"session_id"`
		Content   string `json:"content"`
	}

	if decodeError := json.NewDecoder(request.Body).Decode(&sendMessageRequest); decodeError != nil {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body", nil)
		return
	}

	if sendMessageRequest.SessionID == "" || sendMessageRequest.Content == "" {
		server.writeError(responseWriter, http.StatusBadRequest, "VALIDATION_ERROR", "session_id and content are required", nil)
		return
	}

	userID := server.getUserID(request)

	// 1. Verify session and save user message
	var session models.ChatSession
	databaseError := server.database.QueryRow(`
		SELECT chat_sessions.id, chat_sessions.exam_id FROM chat_sessions 
		JOIN exams ON chat_sessions.exam_id = exams.id
		WHERE chat_sessions.id = ? AND exams.user_id = ?
	`, sendMessageRequest.SessionID, userID).Scan(&session.ID, &session.ExamID)
	if databaseError != nil {
		server.writeError(responseWriter, http.StatusNotFound, "NOT_FOUND", "Chat session not found", nil)
		return
	}

	userMsgID, _ := gonanoid.New()
	userMessage := models.ChatMessage{
		ID:        userMsgID,
		SessionID: sendMessageRequest.SessionID,
		Role:      "user",
		Content:   sendMessageRequest.Content,
		CreatedAt: time.Now(),
	}

	_, databaseError = server.database.Exec(`
		INSERT INTO chat_messages (id, session_id, role, content, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, userMessage.ID, userMessage.SessionID, userMessage.Role, userMessage.Content, userMessage.CreatedAt)

	if databaseError != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to save user message", nil)
		return
	}

	// 1.5. Lock the current context (move included to used)
	var currentIncludedJSON string
	var currentUsedJSON string
	server.database.QueryRow("SELECT included_lecture_ids, used_lecture_ids FROM chat_context_configuration WHERE session_id = ?", sendMessageRequest.SessionID).Scan(&currentIncludedJSON, &currentUsedJSON)

	var currentIncluded []string
	var currentUsed []string
	json.Unmarshal([]byte(currentIncludedJSON), &currentIncluded)
	json.Unmarshal([]byte(currentUsedJSON), &currentUsed)

	// Merge included into used, avoiding duplicates
	usedMap := make(map[string]bool)
	for _, id := range currentUsed {
		usedMap[id] = true
	}
	for _, id := range currentIncluded {
		usedMap[id] = true
	}

	newUsed := make([]string, 0, len(usedMap))
	for id := range usedMap {
		newUsed = append(newUsed, id)
	}
	newUsedJSON, _ := json.Marshal(newUsed)

	_, _ = server.database.Exec(`
		UPDATE chat_context_configuration 
		SET used_lecture_ids = ? 
		WHERE session_id = ?
	`, string(newUsedJSON), sendMessageRequest.SessionID)

	// 2. Get history and context for LLM
	messages := server.getChatHistory(sendMessageRequest.SessionID)

	// Fetch language code for the session
	var languageCode string
	err := server.database.QueryRow(`
		SELECT exams.language FROM exams
		JOIN chat_sessions ON exams.id = chat_sessions.exam_id
		WHERE chat_sessions.id = ?
	`, sendMessageRequest.SessionID).Scan(&languageCode)
	if err != nil || languageCode == "" {
		languageCode = server.configuration.LLM.Language
	}

	lectureContext := server.getLectureContext(sendMessageRequest.SessionID, languageCode)

	// 3. Trigger async AI response
	go server.processAIResponse(sendMessageRequest.SessionID, messages, lectureContext)

	server.writeJSON(responseWriter, http.StatusAccepted, userMessage)
}

func (server *Server) getChatHistory(sessionID string) []llm.Message {
	messageRows, databaseError := server.database.Query(`
		SELECT role, content FROM chat_messages 
		WHERE session_id = ? 
		ORDER BY created_at ASC
	`, sessionID)
	if databaseError != nil {
		slog.Error("Failed to query chat history", "sessionID", sessionID, "error", databaseError)
		return nil
	}
	defer messageRows.Close()

	var messages []llm.Message
	for messageRows.Next() {
		var role, content string
		if scanError := messageRows.Scan(&role, &content); scanError == nil {
			messages = append(messages, llm.Message{
				Role: role,
				Content: []llm.ContentPart{
					{Type: "text", Text: content},
				},
			})
		}
	}
	slog.Debug("Retrieved chat history", "sessionID", sessionID, "count", len(messages))
	return messages
}

func (server *Server) getLectureContext(sessionID string, languageCode string) string {
	var includedLectureIDsJSON string
	var usedLectureIDsJSON string
	databaseError := server.database.QueryRow(`
		SELECT included_lecture_ids, used_lecture_ids 
		FROM chat_context_configuration 
		WHERE session_id = ?
	`, sessionID).Scan(&includedLectureIDsJSON, &usedLectureIDsJSON)
	if databaseError != nil {
		return ""
	}

	var includedIDs, usedIDs []string
	json.Unmarshal([]byte(includedLectureIDsJSON), &includedIDs)
	json.Unmarshal([]byte(usedLectureIDsJSON), &usedIDs)

	// Combine both sets into a unique list
	allIDsMap := make(map[string]bool)
	for _, id := range includedIDs {
		allIDsMap[id] = true
	}
	for _, id := range usedIDs {
		allIDsMap[id] = true
	}

	if len(allIDsMap) == 0 {
		return ""
	}

	markdownReconstructor := markdown.NewReconstructor()
	markdownReconstructor.Language = languageCode
	rootNode := &markdown.Node{Type: markdown.NodeDocument}

	// Iterate through the combined unique IDs
	for lectureID := range allIDsMap {
		var title string
		server.database.QueryRow("SELECT title FROM lectures WHERE id = ?", lectureID).Scan(&title)

		rootNode.Children = append(rootNode.Children, &markdown.Node{
			Type:    markdown.NodeHeading,
			Level:   1,
			Content: title,
		})

		// Add transcript
		transcriptRows, databaseError := server.database.Query(`
			SELECT text FROM transcript_segments 
			WHERE transcript_id = (SELECT id FROM transcripts WHERE lecture_id = ?)
			ORDER BY start_millisecond ASC
		`, lectureID)
		if databaseError == nil {
			var transcriptBuilder strings.Builder
			for transcriptRows.Next() {
				var text string
				if scanError := transcriptRows.Scan(&text); scanError == nil {
					transcriptBuilder.WriteString(text + " ")
				}
			}
			transcriptRows.Close()
			if transcriptBuilder.Len() > 0 {
				rootNode.Children = append(rootNode.Children, &markdown.Node{
					Type:    markdown.NodeParagraph,
					Content: strings.TrimSpace(transcriptBuilder.String()),
				})
			}
		}

		// Add reference documents
		documentRows, databaseError := server.database.Query(`
			SELECT reference_documents.title, reference_pages.page_number, reference_pages.extracted_text
			FROM reference_documents
			JOIN reference_pages ON reference_documents.id = reference_pages.document_id
			WHERE reference_documents.lecture_id = ?
			ORDER BY reference_documents.id, reference_pages.page_number ASC
		`, lectureID)
		if databaseError == nil {
			currentDocTitle := ""
			for documentRows.Next() {
				var title, text string
				var pageNumber int
				if scanError := documentRows.Scan(&title, &pageNumber, &text); scanError == nil {
					if title != currentDocTitle {
						rootNode.Children = append(rootNode.Children, &markdown.Node{
							Type:    markdown.NodeHeading,
							Level:   2,
							Content: "Reference File: " + title,
						})
						currentDocTitle = title
					}
					rootNode.Children = append(rootNode.Children, &markdown.Node{
						Type:    markdown.NodeHeading,
						Level:   3,
						Content: fmt.Sprintf("Page %d", pageNumber),
					})
					rootNode.Children = append(rootNode.Children, &markdown.Node{
						Type:    markdown.NodeParagraph,
						Content: strings.TrimSpace(text),
					})
				}
			}
			documentRows.Close()
		}
	}

	return markdownReconstructor.Reconstruct(rootNode)
}

func (server *Server) processAIResponse(sessionID string, history []llm.Message, lectureContext string) {
	// Fetch language code for the session
	var languageCode string
	err := server.database.QueryRow(`
		SELECT exams.language FROM exams
		JOIN chat_sessions ON chat_sessions.exam_id = exams.id
		WHERE chat_sessions.id = ?
	`, sessionID).Scan(&languageCode)
	if err != nil || languageCode == "" {
		languageCode = server.configuration.LLM.Language
	}

	// Prepare system prompt
	var systemPrompt string
	if server.promptManager != nil {
		latexInstructions, _ := server.promptManager.GetPrompt(prompts.PromptLatexInstructions, nil)
		languageRequirement, _ := server.promptManager.GetPrompt(prompts.PromptLanguageRequirement, map[string]string{
			"language":      languageCode,
			"language_code": languageCode,
		})

		var promptError error
		systemPrompt, promptError = server.promptManager.GetPrompt(prompts.PromptReadingAssistantMultiChat, map[string]string{
			"latex_instructions":   latexInstructions,
			"language_requirement": languageRequirement,
		})
		if promptError != nil {
			slog.Error("Failed to load system prompt", "error", promptError)
			return
		}
	} else {
		// Fallback prompt when promptManager is nil (e.g., in tests)
		systemPrompt = "You are a helpful reading assistant. Help the user understand their lecture materials."
	}

	markdownReconstructor := markdown.NewReconstructor()
	markdownReconstructor.Language = languageCode
	rootNode := &markdown.Node{Type: markdown.NodeDocument}

	// Add the base system prompt as a paragraph
	rootNode.Children = append(rootNode.Children, &markdown.Node{
		Type:    markdown.NodeParagraph,
		Content: systemPrompt,
	})

	// Add lecture context if available
	if lectureContext != "" {
		// Parse and append the lectureContext (which is already markdown reconstructed)
		contextParser := markdown.NewParser()
		contextAST := contextParser.Parse(lectureContext)
		rootNode.Children = append(rootNode.Children, contextAST.Children...)
	}

	finalSystemPrompt := markdownReconstructor.Reconstruct(rootNode)

	fullMessages := append([]llm.Message{
		{Role: "system", Content: []llm.ContentPart{{Type: "text", Text: finalSystemPrompt}}},
	}, history...)

	model := server.configuration.LLM.Model

	responseChannel, chatError := server.llmProvider.Chat(context.Background(), &llm.ChatRequest{
		Model:    model,
		Messages: fullMessages,
		Stream:   true,
	})

	if chatError != nil {
		slog.Error("LLM chat failed", "error", chatError)
		server.wsHub.Broadcast(WSMessage{
			Type:      "chat:error",
			Channel:   "chat:" + sessionID,
			Payload:   map[string]string{"error": "Failed to generate response"},
			Timestamp: time.Now().Format(time.RFC3339),
		})
		return
	}

	var totalMetrics models.JobMetrics
	var completeResponseBuilder strings.Builder
	for chunk := range responseChannel {
		if chunk.Error != nil {
			slog.Error("LLM stream error", "error", chunk.Error)
			server.wsHub.Broadcast(WSMessage{
				Type:      "chat:error",
				Channel:   "chat:" + sessionID,
				Payload:   map[string]string{"error": "Stream interrupted"},
				Timestamp: time.Now().Format(time.RFC3339),
			})
			break
		}

		completeResponseBuilder.WriteString(chunk.Text)
		totalMetrics.InputTokens += chunk.InputTokens
		totalMetrics.OutputTokens += chunk.OutputTokens
		totalMetrics.EstimatedCost += chunk.Cost

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

	// Post-process response: Parse citations and convert to standard footnotes
	markdownReconstructor = markdown.NewReconstructor()
	markdownReconstructor.Language = languageCode
	finalContent, citations := markdownReconstructor.ParseCitations(completeResponseBuilder.String())

	// Improve footnotes using AI if we have citations
	if len(citations) > 0 {
		updatedCitations, footnoteMetrics, err := server.toolGenerator.ProcessFootnotesAI(context.Background(), citations, languageCode, models.GenerationOptions{})
		totalMetrics.InputTokens += footnoteMetrics.InputTokens
		totalMetrics.OutputTokens += footnoteMetrics.OutputTokens
		totalMetrics.EstimatedCost += footnoteMetrics.EstimatedCost
		if err == nil {
			citations = updatedCitations
		}
	}

	finalContent = markdownReconstructor.AppendCitations(finalContent, citations)

	// Convert complete response to HTML
	contentHTML, _ := server.markdownConverter.MarkdownToHTML(finalContent)

	slog.Info("Chat AI response completed",
		"sessionID", sessionID,
		"input_tokens", totalMetrics.InputTokens,
		"output_tokens", totalMetrics.OutputTokens,
		"estimated_cost_usd", totalMetrics.EstimatedCost)

	// Save complete response
	assistantMsgID, _ := gonanoid.New()
	assistantMessage := models.ChatMessage{
		ID:            assistantMsgID,
		SessionID:     sessionID,
		Role:          "assistant",
		Content:       finalContent,
		ContentHTML:   contentHTML,
		ModelUsed:     model,
		InputTokens:   totalMetrics.InputTokens,
		OutputTokens:  totalMetrics.OutputTokens,
		EstimatedCost: totalMetrics.EstimatedCost,
		CreatedAt:     time.Now(),
	}

	_, databaseError := server.database.Exec(`
		INSERT INTO chat_messages (id, session_id, role, content, model_used, input_tokens, output_tokens, estimated_cost, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, assistantMessage.ID, assistantMessage.SessionID, assistantMessage.Role, assistantMessage.Content, assistantMessage.ModelUsed, assistantMessage.InputTokens, assistantMessage.OutputTokens, assistantMessage.EstimatedCost, assistantMessage.CreatedAt)

	if databaseError != nil {
		slog.Error("Failed to save assistant message", "error", databaseError)
	}

	// Broadcast complete message
	server.wsHub.Broadcast(WSMessage{
		Type:      "chat:complete",
		Channel:   "chat:" + sessionID,
		Payload:   assistantMessage,
		Timestamp: time.Now().Format(time.RFC3339),
	})
}
