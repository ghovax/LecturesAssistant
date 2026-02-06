package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"lectures/internal/configuration"
	"lectures/internal/database"
	"lectures/internal/documents"
	"lectures/internal/jobs"
	"lectures/internal/llm"
	"lectures/internal/markdown"
	"lectures/internal/models"
	"lectures/internal/prompts"
	"lectures/internal/tools"
	"lectures/internal/transcription"

	"github.com/gorilla/websocket"
)

type MockLLMProvider struct {
	ResponseText string
	Delay        time.Duration
	Error        error
}

func (mock *MockLLMProvider) Chat(jobContext context.Context, request llm.ChatRequest) (<-chan llm.ChatResponseChunk, error) {
	if mock.Error != nil {
		return nil, mock.Error
	}

	responseChannel := make(chan llm.ChatResponseChunk, 1)

	go func() {
		if mock.Delay > 0 {
			select {
			case <-time.After(mock.Delay):
			case <-jobContext.Done():
				close(responseChannel)
				return
			}
		}

		text := mock.ResponseText
		if len(request.Messages) > 0 {
			lastMessage := request.Messages[len(request.Messages)-1].Content[0].Text
			if strings.Contains(lastMessage, "page_ranges") {
				text = `{"page_ranges": [{"start": 1, "end": 1}]}`
			} else if strings.Contains(lastMessage, "coverage_score") {
				text = `{"coverage_score": 95}`
			} else if strings.Contains(lastMessage, "analyze-lecture-structure") {
				text = "# Outline\n## Introduction\nCoverage: Basics\nIntroduces: \n- Concept 1 - Emphasis: High (Spent lots of time)\n"
			}
		}

		responseChannel <- llm.ChatResponseChunk{Text: text}
		close(responseChannel)
	}()

	return responseChannel, nil
}

func (mock *MockLLMProvider) Name() string { return "mock-llm" }

type MockTranscriptionProvider struct {
	Segments []transcription.Segment
}

func (mock *MockTranscriptionProvider) Transcribe(jobContext context.Context, audioPath string) ([]transcription.Segment, error) {
	return mock.Segments, nil
}

func (mock *MockTranscriptionProvider) SetPrompt(prompt string)  {}
func (mock *MockTranscriptionProvider) CheckDependencies() error { return nil }
func (mock *MockTranscriptionProvider) Name() string             { return "mock-transcription" }

type MockMediaProcessor struct{}

func (mediaProcessor *MockMediaProcessor) CheckDependencies() error { return nil }

func (mediaProcessor *MockMediaProcessor) ExtractAudio(inputPath, outputPath string) error {
	return os.WriteFile(outputPath, []byte("fake audio"), 0644)
}

func (mediaProcessor *MockMediaProcessor) SplitAudio(inputPath, outputDirectory string, segmentDuration int) ([]string, error) {
	if err := os.MkdirAll(outputDirectory, 0755); err != nil {
		return nil, err
	}

	segmentPath := filepath.Join(outputDirectory, "segment_001.mp3")
	if err := os.WriteFile(segmentPath, []byte("fake segment"), 0644); err != nil {
		return nil, err
	}

	return []string{segmentPath}, nil
}

func (mediaProcessor *MockMediaProcessor) GetDuration(inputPath string) (float64, error) {
	return 10.0, nil
}

type MockDocumentConverter struct{}

func (documentConverter *MockDocumentConverter) CheckDependencies() error { return nil }

func (documentConverter *MockDocumentConverter) ConvertToPDF(inputPath, outputPath string) error {
	return os.WriteFile(outputPath, []byte("fake pdf"), 0644)
}

func (documentConverter *MockDocumentConverter) ExtractPagesAsImages(pdfPath, outputDirectory string) ([]string, error) {
	if err := os.MkdirAll(outputDirectory, 0755); err != nil {
		return nil, err
	}

	imagePath := filepath.Join(outputDirectory, "page_001.png")
	if err := os.WriteFile(imagePath, []byte("fake image"), 0644); err != nil {
		return nil, err
	}

	return []string{imagePath}, nil
}

func TestFullPipeline(tester *testing.T) {
	temporaryDirectory, err := os.MkdirTemp("", "lectures-test-*")
	if err != nil {
		tester.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(temporaryDirectory)

	databasePath := filepath.Join(temporaryDirectory, "test.db")
	initializedDatabase, err := database.Initialize(databasePath)
	if err != nil {
		tester.Fatalf("Failed to initialize database: %v", err)
	}
	defer initializedDatabase.Close()

	config := &configuration.Configuration{
		Storage: configuration.StorageConfiguration{DataDirectory: temporaryDirectory},
		Server:  configuration.ServerConfiguration{Host: "127.0.0.1", Port: 0},
		Security: configuration.SecurityConfiguration{
			Auth: configuration.AuthConfiguration{Type: "session", SessionTimeoutHours: 24},
		},
		LLM: configuration.LLMConfiguration{Language: "en-US"},
	}

	promptManager := prompts.NewManager("../../prompts")
	mockLLM := &MockLLMProvider{ResponseText: "Mocked AI Response"}

	transcriptionService := transcription.NewService(config, &MockTranscriptionProvider{
		Segments: []transcription.Segment{{Start: 0, End: 5, Text: "Hello, test lecture."}},
	}, mockLLM, promptManager)
	transcriptionService.SetMediaProcessor(&MockMediaProcessor{})

	documentProcessor := documents.NewProcessor(mockLLM, "mock-model", promptManager)
	documentProcessor.SetConverter(&MockDocumentConverter{})

	markdownConverter := markdown.NewConverter(temporaryDirectory)
	toolGenerator := tools.NewToolGenerator(config, mockLLM, promptManager)

	jobQueue := jobs.NewQueue(initializedDatabase, 1)
	jobs.RegisterHandlers(jobQueue, initializedDatabase, config, transcriptionService, documentProcessor, toolGenerator, markdownConverter, database.CheckLectureReadiness)
	jobQueue.Start()
	defer jobQueue.Stop()

	apiServer := NewServer(config, initializedDatabase, jobQueue, mockLLM, promptManager)
	testServer := httptest.NewServer(apiServer.Handler())
	defer testServer.Close()

	httpClient := testServer.Client()

	// Setup initial password
	setupPayload, err := json.Marshal(map[string]string{"password": "password123"})
	if err != nil {
		tester.Fatalf("Failed to marshal setup payload: %v", err)
	}

	setupResponse, err := httpClient.Post(testServer.URL+"/api/auth/setup", "application/json", bytes.NewBuffer(setupPayload))
	if err != nil {
		tester.Fatalf("Auth setup request failed: %v", err)
	}
	setupResponse.Body.Close()

	// Login to get session token
	loginPayload, err := json.Marshal(map[string]string{"password": "password123"})
	if err != nil {
		tester.Fatalf("Failed to marshal login payload: %v", err)
	}

	loginResponse, err := httpClient.Post(testServer.URL+"/api/auth/login", "application/json", bytes.NewBuffer(loginPayload))
	if err != nil {
		tester.Fatalf("Login request failed: %v", err)
	}
	defer loginResponse.Body.Close()

	var loginData struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(loginResponse.Body).Decode(&loginData); err != nil {
		tester.Fatalf("Failed to decode login response: %v", err)
	}
	sessionToken := loginData.Data.Token

	// Helper to create authenticated requests
	createAuthenticatedRequest := func(method, url string, body io.Reader) *http.Request {
		httpRequest, err := http.NewRequest(method, url, body)
		if err != nil {
			tester.Fatalf("Failed to create request: %v", err)
		}
		httpRequest.Header.Set("Authorization", "Bearer "+sessionToken)
		return httpRequest
	}

	// Create Exam
	examPayload, err := json.Marshal(map[string]string{
		"title":       "Test Course",
		"description": "Integration testing",
	})
	if err != nil {
		tester.Fatalf("Failed to marshal exam payload: %v", err)
	}

	examRequest := createAuthenticatedRequest("POST", testServer.URL+"/api/exams", bytes.NewBuffer(examPayload))
	examRequest.Header.Set("Content-Type", "application/json")

	examResponse, err := httpClient.Do(examRequest)
	if err != nil {
		tester.Fatalf("Exam creation request failed: %v", err)
	}
	defer examResponse.Body.Close()

	var examResponseData struct {
		Data models.Exam `json:"data"`
	}
	if err := json.NewDecoder(examResponse.Body).Decode(&examResponseData); err != nil {
		tester.Fatalf("Failed to decode exam response: %v", err)
	}
	examID := examResponseData.Data.ID

	// Create Lecture with media and documents
	requestBody := &bytes.Buffer{}
	multipartWriter := multipart.NewWriter(requestBody)
	_ = multipartWriter.WriteField("title", "Lecture 1")
	_ = multipartWriter.WriteField("description", "First lecture")

	mediaPart, err := multipartWriter.CreateFormFile("media", "test-audio.mp3")
	if err != nil {
		tester.Fatalf("Failed to create media form file: %v", err)
	}
	_, _ = mediaPart.Write([]byte("fake audio content"))

	documentPart, err := multipartWriter.CreateFormFile("documents", "test-slides.pdf")
	if err != nil {
		tester.Fatalf("Failed to create document form file: %v", err)
	}
	_, _ = documentPart.Write([]byte("fake pdf content"))
	multipartWriter.Close()

	lectureRequest := createAuthenticatedRequest("POST", fmt.Sprintf("%s/api/exams/%s/lectures", testServer.URL, examID), requestBody)
	lectureRequest.Header.Set("Content-Type", multipartWriter.FormDataContentType())

	lectureResponse, err := httpClient.Do(lectureRequest)
	if err != nil {
		tester.Fatalf("Lecture creation request failed: %v", err)
	}
	defer lectureResponse.Body.Close()

	var lectureResponseData struct {
		Data models.Lecture `json:"data"`
	}
	if err := json.NewDecoder(lectureResponse.Body).Decode(&lectureResponseData); err != nil {
		tester.Fatalf("Failed to decode lecture response: %v", err)
	}
	lectureID := lectureResponseData.Data.ID

	// Wait for background jobs to complete (Lecture status -> 'ready')
	deadline := time.Now().Add(10 * time.Second)
	var lectureStatus string
	for time.Now().Before(deadline) {
		_ = initializedDatabase.QueryRow("SELECT status FROM lectures WHERE id = ?", lectureID).Scan(&lectureStatus)
		if lectureStatus == "ready" {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	if lectureStatus != "ready" {
		tester.Fatalf("Lecture failed to reach 'ready' status, got %q", lectureStatus)
	}

	// Create Chat Session
	chatPayload, err := json.Marshal(map[string]string{"title": "Study Session"})
	if err != nil {
		tester.Fatalf("Failed to marshal chat payload: %v", err)
	}

	chatRequest := createAuthenticatedRequest("POST", fmt.Sprintf("%s/api/exams/%s/chat/sessions", testServer.URL, examID), bytes.NewBuffer(chatPayload))
	chatRequest.Header.Set("Content-Type", "application/json")

	chatResponse, err := httpClient.Do(chatRequest)
	if err != nil {
		tester.Fatalf("Chat session creation request failed: %v", err)
	}
	defer chatResponse.Body.Close()

	var chatResponseData struct {
		Data models.ChatSession `json:"data"`
	}
	if err := json.NewDecoder(chatResponse.Body).Decode(&chatResponseData); err != nil {
		tester.Fatalf("Failed to decode chat session response: %v", err)
	}
	sessionID := chatResponseData.Data.ID

	// Send user message
	messagePayload, err := json.Marshal(map[string]string{"content": "Tell me about the lecture"})
	if err != nil {
		tester.Fatalf("Failed to marshal message payload: %v", err)
	}

	messageRequest := createAuthenticatedRequest("POST", fmt.Sprintf("%s/api/exams/%s/chat/sessions/%s/messages", testServer.URL, examID, sessionID), bytes.NewBuffer(messagePayload))
	messageRequest.Header.Set("Content-Type", "application/json")

	messageResponse, err := httpClient.Do(messageRequest)
	if err != nil {
		tester.Fatalf("Message sending failed: %v", err)
	}
	messageResponse.Body.Close()

	// Wait for async AI response
	var assistantMessageCount int
	messageDeadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(messageDeadline) {
		_ = initializedDatabase.QueryRow("SELECT COUNT(*) FROM chat_messages WHERE session_id = ? AND role = 'assistant'", sessionID).Scan(&assistantMessageCount)
		if assistantMessageCount == 1 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if assistantMessageCount != 1 {
		tester.Errorf("Expected 1 assistant message, found %d", assistantMessageCount)
	}
}

func TestWebSocketUpdates(tester *testing.T) {
	temporaryDirectory, err := os.MkdirTemp("", "lectures-ws-test-*")
	if err != nil {
		tester.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(temporaryDirectory)

	databasePath := filepath.Join(temporaryDirectory, "test.db")
	initializedDatabase, err := database.Initialize(databasePath)
	if err != nil {
		tester.Fatalf("Failed to initialize database: %v", err)
	}
	defer initializedDatabase.Close()

	config := &configuration.Configuration{
		Security: configuration.SecurityConfiguration{
			Auth: configuration.AuthConfiguration{Type: "session"},
		},
	}

	_, _ = initializedDatabase.Exec("INSERT INTO settings (key, value) VALUES ('admin_password_hash', ?)", "dummy_hash")

	sessionID := "test-session-id"
	_, _ = initializedDatabase.Exec("INSERT INTO auth_sessions (id, password_hash, expires_at) VALUES (?, ?, ?)", sessionID, "dummy_hash", time.Now().Add(1*time.Hour))

	jobQueue := jobs.NewQueue(initializedDatabase, 1)
	apiServer := NewServer(config, initializedDatabase, jobQueue, nil, nil)

	testServer := httptest.NewServer(apiServer.Handler())
	defer testServer.Close()

	websocketURL := "ws" + strings.TrimPrefix(testServer.URL, "http") + "/api/socket"
	headers := http.Header{}
	headers.Add("Authorization", "Bearer "+sessionID)

	dialer := websocket.Dialer{}
	websocketConnection, _, err := dialer.Dial(websocketURL, headers)
	if err != nil {
		tester.Fatalf("WebSocket dial failed: %v", err)
	}
	defer websocketConnection.Close()

	var handshake map[string]any
	if err := websocketConnection.ReadJSON(&handshake); err != nil {
		tester.Fatalf("Failed to read handshake: %v", err)
	}

	if handshake["type"] != "connected" {
		tester.Errorf("Expected 'connected', got %v", handshake["type"])
	}

	jobID := "test-job-123"
	subscribeRequest := map[string]string{
		"type":    "subscribe",
		"channel": "job:" + jobID,
	}
	if err := websocketConnection.WriteJSON(subscribeRequest); err != nil {
		tester.Fatalf("Failed to send subscribe request: %v", err)
	}

	var subscriptionConfirmation map[string]any
	if err := websocketConnection.ReadJSON(&subscriptionConfirmation); err != nil {
		tester.Fatalf("Failed to read subscription confirmation: %v", err)
	}

	// Broadcast update and verify receipt
	apiServer.wsHub.Broadcast(WSMessage{
		Type:    "job:progress",
		Channel: "job:" + jobID,
		Payload: map[string]any{
			"status":   "RUNNING",
			"progress": 50,
		},
		Timestamp: time.Now().Format(time.RFC3339),
	})

	var progressUpdate WSMessage
	if err := websocketConnection.ReadJSON(&progressUpdate); err != nil {
		tester.Fatalf("Failed to read update: %v", err)
	}

	if progressUpdate.Type != "job:progress" {
		tester.Errorf("Expected 'job:progress', got %s", progressUpdate.Type)
	}
}

func TestAIFailureModes(tester *testing.T) {
	temporaryDirectory, err := os.MkdirTemp("", "lectures-fail-test-*")
	if err != nil {
		tester.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(temporaryDirectory)

	databasePath := filepath.Join(temporaryDirectory, "test.db")
	initializedDatabase, err := database.Initialize(databasePath)
	if err != nil {
		tester.Fatalf("Failed to initialize database: %v", err)
	}
	defer initializedDatabase.Close()

	config := &configuration.Configuration{
		Storage: configuration.StorageConfiguration{DataDirectory: temporaryDirectory},
		LLM:     configuration.LLMConfiguration{Language: "en-US"},
	}

	promptManager := prompts.NewManager("../../prompts")
	mockLLM := &MockLLMProvider{}

	jobQueue := jobs.NewQueue(initializedDatabase, 1)
	transcriptionService := transcription.NewService(config, &MockTranscriptionProvider{}, mockLLM, promptManager)
	documentProcessor := documents.NewProcessor(mockLLM, "mock-model", promptManager)
	toolGenerator := tools.NewToolGenerator(config, mockLLM, promptManager)
	markdownConverter := markdown.NewConverter(temporaryDirectory)

	jobs.RegisterHandlers(jobQueue, initializedDatabase, config, transcriptionService, documentProcessor, toolGenerator, markdownConverter, database.CheckLectureReadiness)
	jobQueue.Start()
	defer jobQueue.Stop()

	lectureID, examID := "l1", "e1"
	_, _ = initializedDatabase.Exec("INSERT INTO exams (id, title, description) VALUES (?, ?, ?)", examID, "Exam", "Desc")
	_, _ = initializedDatabase.Exec("INSERT INTO lectures (id, exam_id, title, description, status) VALUES (?, ?, ?, ?, ?)", lectureID, examID, "Lecture", "Desc", "ready")
	_, _ = initializedDatabase.Exec("INSERT INTO transcripts (id, lecture_id, status) VALUES (?, ?, ?)", "t1", lectureID, "completed")
	_, _ = initializedDatabase.Exec("INSERT INTO transcript_segments (transcript_id, text, start_millisecond, end_millisecond) VALUES (?, ?, ?, ?)", "t1", "Hi", 0, 1000)

	tester.Run("Faulty JSON Response from AI", func(subTester *testing.T) {
		mockLLM.ResponseText = "Not JSON"
		mockLLM.Error = nil
		mockLLM.Delay = 0

		jobID, _ := jobQueue.Enqueue(models.JobTypeBuildMaterial, map[string]string{
			"lecture_id": lectureID,
			"type":       "guide",
		})

		// Poll for failure
		deadline := time.Now().Add(5 * time.Second)
		var status, jobError string
		for time.Now().Before(deadline) {
			_ = initializedDatabase.QueryRow("SELECT status, error FROM jobs WHERE id = ?", jobID).Scan(&status, &jobError)
			if status == models.JobStatusFailed || status == models.JobStatusCompleted {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}

		if status != models.JobStatusFailed || !strings.Contains(jobError, "failed to parse sections from LLM response") {
			subTester.Errorf("Job did not fail as expected: %s (%s)", status, jobError)
		}
	})

	tester.Run("AI Provider Error", func(subTester *testing.T) {
		mockLLM.Error = errors.New("connection refused")

		jobID, _ := jobQueue.Enqueue(models.JobTypeBuildMaterial, map[string]string{
			"lecture_id": lectureID,
			"type":       "guide",
		})

		// Poll for failure
		deadline := time.Now().Add(5 * time.Second)
		var status, jobError string
		for time.Now().Before(deadline) {
			_ = initializedDatabase.QueryRow("SELECT status, error FROM jobs WHERE id = ?", jobID).Scan(&status, &jobError)
			if status == models.JobStatusFailed || status == models.JobStatusCompleted {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}

		if status != models.JobStatusFailed || !strings.Contains(jobError, "connection refused") {
			subTester.Errorf("Job did not fail as expected: %s (%s)", status, jobError)
		}
	})

	tester.Run("AI Hang (Cancel Job)", func(subTester *testing.T) {
		mockLLM.Error = nil
		mockLLM.Delay = 2 * time.Second

		jobID, _ := jobQueue.Enqueue(models.JobTypeBuildMaterial, map[string]string{
			"lecture_id": lectureID,
			"type":       "guide",
		})

		// Wait for job to start running
		time.Sleep(200 * time.Millisecond)
		_ = jobQueue.CancelJob(jobID)

		// Poll for cancellation
		deadline := time.Now().Add(2 * time.Second)
		var status string
		for time.Now().Before(deadline) {
			_ = initializedDatabase.QueryRow("SELECT status FROM jobs WHERE id = ?", jobID).Scan(&status)
			if status == models.JobStatusCancelled {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}

		if status != models.JobStatusCancelled {
			subTester.Errorf("Expected cancelled, got %s", status)
		}
	})
}

func TestStudyTools(tester *testing.T) {
	temporaryDirectory, err := os.MkdirTemp("", "lectures-tools-test-*")
	if err != nil {
		tester.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(temporaryDirectory)

	databasePath := filepath.Join(temporaryDirectory, "test.db")
	initializedDatabase, err := database.Initialize(databasePath)
	if err != nil {
		tester.Fatalf("Failed to initialize database: %v", err)
	}
	defer initializedDatabase.Close()

	config := &configuration.Configuration{
		Storage: configuration.StorageConfiguration{DataDirectory: temporaryDirectory},
	}

	promptManager := prompts.NewManager("../../prompts")
	mockLLM := &MockLLMProvider{ResponseText: `[{"front": "Q", "back": "A"}]`}

	jobQueue := jobs.NewQueue(initializedDatabase, 1)
	toolGenerator := tools.NewToolGenerator(config, mockLLM, promptManager)

	jobs.RegisterHandlers(jobQueue, initializedDatabase, config, nil, nil, toolGenerator, nil, nil)
	jobQueue.Start()
	defer jobQueue.Stop()

	examID, lectureID := "exam-1", "lecture-1"
	_, _ = initializedDatabase.Exec("INSERT INTO exams (id, title, description) VALUES (?, ?, ?)", examID, "Exam", "Desc")
	_, _ = initializedDatabase.Exec("INSERT INTO lectures (id, exam_id, title, description, status) VALUES (?, ?, ?, ?, ?)", lectureID, examID, "Lecture", "Desc", "ready")
	_, _ = initializedDatabase.Exec("INSERT INTO transcripts (id, lecture_id, status) VALUES (?, ?, ?)", "t1", lectureID, "completed")
	_, _ = initializedDatabase.Exec("INSERT INTO transcript_segments (transcript_id, text, start_millisecond, end_millisecond) VALUES (?, ?, ?, ?)", "t1", "Content", 0, 1000)

	tester.Run("Flashcards", func(subTester *testing.T) {
		jobID, _ := jobQueue.Enqueue(models.JobTypeBuildMaterial, map[string]string{
			"lecture_id": lectureID,
			"exam_id":    examID,
			"type":       "flashcard",
		})

		// Poll for completion
		deadline := time.Now().Add(5 * time.Second)
		var status string
		for time.Now().Before(deadline) {
			_ = initializedDatabase.QueryRow("SELECT status FROM jobs WHERE id = ?", jobID).Scan(&status)
			if status == models.JobStatusCompleted || status == models.JobStatusFailed {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}

		if status != models.JobStatusCompleted {
			subTester.Errorf("Expected completed, got %s", status)
		}
	})

	tester.Run("Quiz", func(subTester *testing.T) {
		mockLLM.ResponseText = `[{"question": "Q", "options": ["A", "B", "C", "D"], "correct_answer": "A", "explanation": "E"}]`

		jobID, _ := jobQueue.Enqueue(models.JobTypeBuildMaterial, map[string]string{
			"lecture_id": lectureID,
			"exam_id":    examID,
			"type":       "quiz",
		})

		// Poll for completion
		deadline := time.Now().Add(5 * time.Second)
		var status string
		for time.Now().Before(deadline) {
			_ = initializedDatabase.QueryRow("SELECT status FROM jobs WHERE id = ?", jobID).Scan(&status)
			if status == models.JobStatusCompleted || status == models.JobStatusFailed {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}

		if status != models.JobStatusCompleted {
			subTester.Errorf("Expected completed, got %s", status)
		}
	})
}

type MockMarkdownConverter struct{}

func (markdownConverter *MockMarkdownConverter) CheckDependencies() error { return nil }

func (markdownConverter *MockMarkdownConverter) MarkdownToHTML(markdownText string) (string, error) {
	return "<html></html>", nil
}

func (markdownConverter *MockMarkdownConverter) HTMLToPDF(htmlContent, outputPath string, options markdown.ConversionOptions) error {
	return os.WriteFile(outputPath, []byte("fake pdf"), 0644)
}

func TestPDFExport(tester *testing.T) {
	temporaryDirectory, err := os.MkdirTemp("", "lectures-export-test-*")
	if err != nil {
		tester.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(temporaryDirectory)

	databasePath := filepath.Join(temporaryDirectory, "test.db")
	initializedDatabase, err := database.Initialize(databasePath)
	if err != nil {
		tester.Fatalf("Failed to initialize database: %v", err)
	}
	defer initializedDatabase.Close()

	config := &configuration.Configuration{
		Storage: configuration.StorageConfiguration{DataDirectory: temporaryDirectory},
	}

	jobQueue := jobs.NewQueue(initializedDatabase, 1)
	jobs.RegisterHandlers(jobQueue, initializedDatabase, config, nil, nil, nil, &MockMarkdownConverter{}, nil)
	jobQueue.Start()
	defer jobQueue.Stop()

	toolID := "tool-1"
	_, _ = initializedDatabase.Exec("INSERT INTO exams (id, title) VALUES ('e1', 'E')")
	_, _ = initializedDatabase.Exec("INSERT INTO tools (id, exam_id, type, title, content) VALUES (?, 'e1', 'guide', 'Title', 'Content')", toolID)

	jobID, err := jobQueue.Enqueue(models.JobTypePublishMaterial, map[string]string{
		"tool_id": toolID,
	})
	if err != nil {
		tester.Fatalf("Failed to enqueue job: %v", err)
	}

	// Poll for completion
	deadline := time.Now().Add(5 * time.Second)
	var status, result string
	for time.Now().Before(deadline) {
		_ = initializedDatabase.QueryRow("SELECT status, result FROM jobs WHERE id = ?", jobID).Scan(&status, &result)
		if status == models.JobStatusCompleted || status == models.JobStatusFailed {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if status != models.JobStatusCompleted || !strings.Contains(result, "pdf_path") {
		tester.Errorf("Export failed: %s (%s)", status, result)
	}
}

func TestUnauthorizedAccess(tester *testing.T) {
	temporaryDirectory, err := os.MkdirTemp("", "lectures-auth-test-*")
	if err != nil {
		tester.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(temporaryDirectory)

	databasePath := filepath.Join(temporaryDirectory, "test.db")
	initializedDatabase, err := database.Initialize(databasePath)
	if err != nil {
		tester.Fatalf("Failed to initialize database: %v", err)
	}
	defer initializedDatabase.Close()

	config := &configuration.Configuration{}
	apiServer := NewServer(config, initializedDatabase, nil, nil, nil)
	testServer := httptest.NewServer(apiServer.Handler())
	defer testServer.Close()

	endpoints := []string{"/api/exams", "/api/jobs", "/api/settings"}

	for _, endpoint := range endpoints {
		tester.Run(endpoint, func(subTester *testing.T) {
			httpResponse, err := http.Get(testServer.URL + endpoint)
			if err != nil {
				subTester.Fatalf("Request failed: %v", err)
			}
			defer httpResponse.Body.Close()

			if httpResponse.StatusCode != http.StatusUnauthorized {
				subTester.Errorf("Expected 401 for %s, got %d", endpoint, httpResponse.StatusCode)
			}
		})
	}
}

func TestUserDailyUsageScenarios(tester *testing.T) {
	// Setup environment
	temporaryDirectory, err := os.MkdirTemp("", "user-usage-test-*")
	if err != nil {
		tester.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(temporaryDirectory)

	databasePath := filepath.Join(temporaryDirectory, "user_usage.db")
	initializedDatabase, err := database.Initialize(databasePath)
	if err != nil {
		tester.Fatalf("Failed to init DB: %v", err)
	}
	defer initializedDatabase.Close()

	config := &configuration.Configuration{
		Storage: configuration.StorageConfiguration{DataDirectory: temporaryDirectory},
		Server:  configuration.ServerConfiguration{Host: "127.0.0.1", Port: 0},
		Security: configuration.SecurityConfiguration{
			Auth: configuration.AuthConfiguration{Type: "session", SessionTimeoutHours: 1},
		},
	}

	promptManager := prompts.NewManager("../../prompts")
	jobQueue := jobs.NewQueue(initializedDatabase, 1)
	jobQueue.Start()
	defer jobQueue.Stop()

	apiServer := NewServer(config, initializedDatabase, jobQueue, nil, promptManager)
	testServer := httptest.NewServer(apiServer.Handler())
	defer testServer.Close()

	httpClient := testServer.Client()
	var sessionToken string

	tester.Run("Initial Setup and Misusage", func(subTester *testing.T) {
		// 1. Try to login before setup
		loginPayload, _ := json.Marshal(map[string]string{"password": "password123"})
		httpResponse, _ := httpClient.Post(testServer.URL+"/api/auth/login", "application/json", bytes.NewBuffer(loginPayload))
		if httpResponse.StatusCode != http.StatusForbidden {
			subTester.Errorf("Expected 403 Forbidden for login before setup, got %d", httpResponse.StatusCode)
		}

		// 2. Setup with too short password
		setupPayload, _ := json.Marshal(map[string]string{"password": "short"})
		httpResponse, _ = httpClient.Post(testServer.URL+"/api/auth/setup", "application/json", bytes.NewBuffer(setupPayload))
		if httpResponse.StatusCode != http.StatusBadRequest {
			subTester.Errorf("Expected 400 Bad Request for short password, got %d", httpResponse.StatusCode)
		}

		// 3. Valid setup
		setupPayload, _ = json.Marshal(map[string]string{"password": "valid_password"})
		httpResponse, _ = httpClient.Post(testServer.URL+"/api/auth/setup", "application/json", bytes.NewBuffer(setupPayload))
		if httpResponse.StatusCode != http.StatusOK {
			subTester.Errorf("Expected 200 OK for valid setup, got %d", httpResponse.StatusCode)
		}

		// 4. Try setup again (should fail)
		httpResponse, _ = httpClient.Post(testServer.URL+"/api/auth/setup", "application/json", bytes.NewBuffer(setupPayload))
		if httpResponse.StatusCode != http.StatusForbidden {
			subTester.Errorf("Expected 403 Forbidden for duplicate setup, got %d", httpResponse.StatusCode)
		}

		// 5. Valid login
		loginPayload, _ = json.Marshal(map[string]string{"password": "valid_password"})
		httpResponse, _ = httpClient.Post(testServer.URL+"/api/auth/login", "application/json", bytes.NewBuffer(loginPayload))
		
		var loginResponseData struct {
			Data struct {
				Token string `json:"token"`
			} `json:"data"`
		}
		json.NewDecoder(httpResponse.Body).Decode(&loginResponseData)
		sessionToken = loginResponseData.Data.Token
		if sessionToken == "" {
			subTester.Fatal("Failed to get session token")
		}
	})

	authenticatedDo := func(httpRequest *http.Request) *http.Response {
		httpRequest.Header.Set("Authorization", "Bearer "+sessionToken)
		httpResponse, err := httpClient.Do(httpRequest)
		if err != nil {
			tester.Fatalf("Request failed: %v", err)
		}
		return httpResponse
	}

	var examID string
	tester.Run("Exam Management & Validations", func(subTester *testing.T) {
		// 1. Create exam with empty title
		payload, _ := json.Marshal(map[string]string{"title": ""})
		httpRequest, _ := http.NewRequest("POST", testServer.URL+"/api/exams", bytes.NewBuffer(payload))
		httpResponse := authenticatedDo(httpRequest)
		if httpResponse.StatusCode != http.StatusBadRequest {
			subTester.Errorf("Expected 400 for empty exam title, got %d", httpResponse.StatusCode)
		}

		// 2. Create valid exam
		payload, _ = json.Marshal(map[string]string{"title": "Biology 101", "description": "Intro to Bio"})
		httpRequest, _ = http.NewRequest("POST", testServer.URL+"/api/exams", bytes.NewBuffer(payload))
		httpResponse = authenticatedDo(httpRequest)
		var examResponseData struct {
			Data models.Exam `json:"data"`
		}
		json.NewDecoder(httpResponse.Body).Decode(&examResponseData)
		examID = examResponseData.Data.ID

		// 3. Update exam
		updatePayload, _ := json.Marshal(map[string]string{"title": "Advanced Biology"})
		httpRequest, _ = http.NewRequest("PATCH", testServer.URL+"/api/exams/"+examID, bytes.NewBuffer(updatePayload))
		httpResponse = authenticatedDo(httpRequest)
		json.NewDecoder(httpResponse.Body).Decode(&examResponseData)
		if examResponseData.Data.Title != "Advanced Biology" {
			subTester.Errorf("Expected title update, got %s", examResponseData.Data.Title)
		}

		// 4. Get non-existent exam
		httpRequest, _ = http.NewRequest("GET", testServer.URL+"/api/exams/invalid-id", nil)
		httpResponse = authenticatedDo(httpRequest)
		if httpResponse.StatusCode != http.StatusNotFound {
			subTester.Errorf("Expected 404 for non-existent exam, got %d", httpResponse.StatusCode)
		}
	})

	var lectureID string
	tester.Run("Lecture Management & Cascade", func(subTester *testing.T) {
		// 1. Create lecture for invalid exam
		requestBody := &bytes.Buffer{}
		multipartWriter := multipart.NewWriter(requestBody)
		_ = multipartWriter.WriteField("title", "Lecture 1")
		multipartWriter.Close()
		httpRequest, _ := http.NewRequest("POST", testServer.URL+"/api/exams/wrong-exam/lectures", requestBody)
		httpRequest.Header.Set("Content-Type", multipartWriter.FormDataContentType())
		httpResponse := authenticatedDo(httpRequest)
		if httpResponse.StatusCode != http.StatusNotFound {
			subTester.Errorf("Expected 404 when creating lecture for invalid exam, got %d", httpResponse.StatusCode)
		}

		// 2. Create valid lecture
		requestBody = &bytes.Buffer{}
		multipartWriter = multipart.NewWriter(requestBody)
		_ = multipartWriter.WriteField("title", "Cell Structure")
		mediaPart, _ := multipartWriter.CreateFormFile("media", "test.mp3")
		_, _ = mediaPart.Write([]byte("audio data"))
		multipartWriter.Close()
		httpRequest, _ = http.NewRequest("POST", testServer.URL+"/api/exams/"+examID+"/lectures", requestBody)
		httpRequest.Header.Set("Content-Type", multipartWriter.FormDataContentType())
		httpResponse = authenticatedDo(httpRequest)
		var lectureResponseData struct {
			Data models.Lecture `json:"data"`
		}
		json.NewDecoder(httpResponse.Body).Decode(&lectureResponseData)
		lectureID = lectureResponseData.Data.ID

		// 3. Try to delete lecture while it is processing
		// (The status is 'processing' immediately after creation)
		httpRequest, _ = http.NewRequest("DELETE", testServer.URL+fmt.Sprintf("/api/exams/%s/lectures/%s", examID, lectureID), nil)
		httpResponse = authenticatedDo(httpRequest)
		if httpResponse.StatusCode != http.StatusConflict {
			subTester.Errorf("Expected 409 Conflict when deleting processing lecture, got %d", httpResponse.StatusCode)
		}

		// 4. Update lecture status to 'ready' manually in DB to allow deletion
		_, _ = initializedDatabase.Exec("UPDATE lectures SET status = 'ready' WHERE id = ?", lectureID)

		// 5. Delete Exam and verify cascade
		httpRequest, _ = http.NewRequest("DELETE", testServer.URL+"/api/exams/"+examID, nil)
		httpResponse = authenticatedDo(httpRequest)
		if httpResponse.StatusCode != http.StatusOK {
			subTester.Errorf("Expected 200 OK for exam deletion, got %d", httpResponse.StatusCode)
		}

		// Verify lecture is gone
		var count int
		_ = initializedDatabase.QueryRow("SELECT COUNT(*) FROM lectures WHERE id = ?", lectureID).Scan(&count)
		if count != 0 {
			subTester.Error("Lecture was not deleted via cascade from Exam")
		}

		// Verify files are gone
		lectureDirectory := filepath.Join(temporaryDirectory, "files", "lectures", lectureID)
		if _, err := os.Stat(lectureDirectory); !os.IsNotExist(err) {
			subTester.Error("Lecture directory was not cleaned up after exam deletion")
		}
	})

	tester.Run("Session Lifecycle", func(subTester *testing.T) {
		// 1. Check status (authenticated)
		httpRequest, _ := http.NewRequest("GET", testServer.URL+"/api/auth/status", nil)
		httpRequest.Header.Set("Authorization", "Bearer "+sessionToken)
		httpResponse, _ := httpClient.Do(httpRequest)
		var authStatusResponse struct {
			Data struct {
				Authenticated bool `json:"authenticated"`
			} `json:"data"`
		}
		json.NewDecoder(httpResponse.Body).Decode(&authStatusResponse)
		if !authStatusResponse.Data.Authenticated {
			subTester.Error("Expected authenticated status to be true")
		}

		// 2. Logout
		httpRequest, _ = http.NewRequest("POST", testServer.URL+"/api/auth/logout", nil)
		httpResponse = authenticatedDo(httpRequest)
		if httpResponse.StatusCode != http.StatusOK {
			subTester.Errorf("Logout failed with status %d", httpResponse.StatusCode)
		}

		// 3. Check status again (should be false)
		httpRequest, _ = http.NewRequest("GET", testServer.URL+"/api/auth/status", nil)
		httpRequest.Header.Set("Authorization", "Bearer "+sessionToken)
		httpResponse, _ = httpClient.Do(httpRequest)
		_ = json.NewDecoder(httpResponse.Body).Decode(&authStatusResponse)
		if authStatusResponse.Data.Authenticated {
			subTester.Error("Expected authenticated status to be false after logout")
		}

		// 4. Try to access protected endpoint (should fail)
		httpRequest, _ = http.NewRequest("GET", testServer.URL+"/api/exams", nil)
		httpRequest.Header.Set("Authorization", "Bearer "+sessionToken)
		httpResponse, _ = httpClient.Do(httpRequest)
		if httpResponse.StatusCode != http.StatusUnauthorized {
			subTester.Errorf("Expected 401 Unauthorized after logout, got %d", httpResponse.StatusCode)
		}
	})
}