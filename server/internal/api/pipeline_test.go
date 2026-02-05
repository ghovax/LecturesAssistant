package api

import (
	"bytes"
	"context"
	"encoding/json"
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
)

// MockLLMProvider simulates LLM responses
type MockLLMProvider struct {
	ResponseText string
}

func (mock *MockLLMProvider) Chat(ctx context.Context, request llm.ChatRequest) (<-chan llm.ChatResponseChunk, error) {
	responseChannel := make(chan llm.ChatResponseChunk, 1)
	
	text := mock.ResponseText
	
	// Intelligent mocking based on prompt content
	if len(request.Messages) > 0 {
		lastMessage := request.Messages[len(request.Messages)-1].Content[0].Text
		
		if strings.Contains(lastMessage, "page_ranges") {
			text = `{"page_ranges": [{"start": 1, "end": 1}]}`
		} else if strings.Contains(lastMessage, "coverage_score") {
			text = `{"coverage_score": 95}`
		} else if strings.Contains(lastMessage, "analyze-lecture-structure") {
			text = `# Outline
## Introduction
Coverage: Basics
Introduces: 
- Concept 1 - Emphasis: High (Spent lots of time)
`
		}
	}
	
	responseChannel <- llm.ChatResponseChunk{Text: text}
	close(responseChannel)
	return responseChannel, nil
}

func (mock *MockLLMProvider) Name() string { return "mock-llm" }

// MockTranscriptionProvider simulates transcription
type MockTranscriptionProvider struct {
	Segments []transcription.Segment
}

func (mock *MockTranscriptionProvider) Transcribe(ctx context.Context, audioPath string) ([]transcription.Segment, error) {
	return mock.Segments, nil
}

func (mock *MockTranscriptionProvider) SetPrompt(prompt string) {}
func (mock *MockTranscriptionProvider) CheckDependencies() error { return nil }
func (mock *MockTranscriptionProvider) Name() string            { return "mock-transcription" }

// MockMediaProcessor mocks FFmpeg
type MockMediaProcessor struct{}

func (m *MockMediaProcessor) CheckDependencies() error { return nil }
func (m *MockMediaProcessor) ExtractAudio(inputPath string, outputPath string) error {
	return os.WriteFile(outputPath, []byte("fake audio"), 0644)
}
func (m *MockMediaProcessor) SplitAudio(inputPath string, outputDirectory string, duration int) ([]string, error) {
	os.MkdirAll(outputDirectory, 0755)
	segmentPath := filepath.Join(outputDirectory, "segment_001.mp3")
	os.WriteFile(segmentPath, []byte("fake segment"), 0644)
	return []string{segmentPath}, nil
}
func (m *MockMediaProcessor) GetDuration(inputPath string) (float64, error) { return 10.0, nil }

// MockDocumentConverter mocks GS and LibreOffice
type MockDocumentConverter struct{}

func (m *MockDocumentConverter) CheckDependencies() error { return nil }
func (m *MockDocumentConverter) ConvertToPDF(inputPath string, outputPath string) error {
	return os.WriteFile(outputPath, []byte("fake pdf"), 0644)
}
func (m *MockDocumentConverter) ExtractPagesAsImages(pdfPath string, outputDirectory string) ([]string, error) {
	os.MkdirAll(outputDirectory, 0755)
	imagePath := filepath.Join(outputDirectory, "page_001.png")
	os.WriteFile(imagePath, []byte("fake image"), 0644)
	return []string{imagePath}, nil
}

func TestFullPipeline(tester *testing.T) {
	// 1. Setup temporary environment
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
		Storage: configuration.StorageConfiguration{
			DataDirectory: temporaryDirectory,
		},
		Server: configuration.ServerConfiguration{
			Host: "127.0.0.1",
			Port: 0,
		},
		Security: configuration.SecurityConfiguration{
			Auth: configuration.AuthConfiguration{
				Type:                "session",
				SessionTimeoutHours: 24,
			},
		},
		LLM: configuration.LLMConfiguration{
			Language: "en-US",
		},
	}

	promptManager := prompts.NewManager("../../prompts")
	mockLLM := &MockLLMProvider{ResponseText: "Mocked AI Response"}
	
	// Mock transcription service
	mockTranscriptionProvider := &MockTranscriptionProvider{
		Segments: []transcription.Segment{
			{Start: 0, End: 5, Text: "Hello, this is a test lecture."},
		},
	}
	transcriptionService := transcription.NewService(config, mockTranscriptionProvider, mockLLM, promptManager)
	transcriptionService.SetMediaProcessor(&MockMediaProcessor{})
	
	// Mock document processor
	documentProcessor := documents.NewProcessor(mockLLM, "mock-model", promptManager)
	documentProcessor.SetConverter(&MockDocumentConverter{})
	
	markdownConverter := markdown.NewConverter(temporaryDirectory)
	toolGenerator := tools.NewToolGenerator(config, mockLLM, promptManager)

	jobQueue := jobs.NewQueue(initializedDatabase, 1)
	
	// Register real handlers using mocks
	jobs.RegisterHandlers(
		jobQueue,
		initializedDatabase,
		config,
		transcriptionService,
		documentProcessor,
		toolGenerator,
		markdownConverter,
		database.CheckLectureReadiness,
	)
	
	jobQueue.Start()
	defer jobQueue.Stop()

	apiServer := NewServer(config, initializedDatabase, jobQueue, mockLLM, promptManager)
	testServer := httptest.NewServer(apiServer.Handler())
	defer testServer.Close()

	httpClient := testServer.Client()

	// 2. Authentication Setup
	setupPayload, _ := json.Marshal(map[string]string{"password": "password123"})
	setupResponse, err := httpClient.Post(testServer.URL+"/api/auth/setup", "application/json", bytes.NewBuffer(setupPayload))
	if err != nil || setupResponse.StatusCode != http.StatusOK {
		tester.Fatalf("Auth setup failed: %v, status: %d", err, setupResponse.StatusCode)
	}

	// 3. Login
	loginPayload, _ := json.Marshal(map[string]string{"password": "password123"})
	loginResponse, err := httpClient.Post(testServer.URL+"/api/auth/login", "application/json", bytes.NewBuffer(loginPayload))
	if err != nil || loginResponse.StatusCode != http.StatusOK {
		tester.Fatalf("Login failed: %v", err)
	}
	
	var loginData struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	json.NewDecoder(loginResponse.Body).Decode(&loginData)
	sessionToken := loginData.Data.Token

	// Helper to add auth header
	createAuthenticatedRequest := func(method, url string, body io.Reader) *http.Request {
		request, _ := http.NewRequest(method, url, body)
		request.Header.Set("Authorization", "Bearer "+sessionToken)
		return request
	}

	// 4. Create Exam
	examPayload, _ := json.Marshal(map[string]string{
		"title":       "Test Course",
		"description": "Integration testing",
	})
	examRequest := createAuthenticatedRequest("POST", testServer.URL+"/api/exams", bytes.NewBuffer(examPayload))
	examRequest.Header.Set("Content-Type", "application/json")
	examResponse, err := httpClient.Do(examRequest)
	if err != nil || examResponse.StatusCode != http.StatusCreated {
		tester.Fatalf("Exam creation failed: %v, status: %d", err, examResponse.StatusCode)
	}
	
	var examResult struct {
		Data models.Exam `json:"data"`
	}
	json.NewDecoder(examResponse.Body).Decode(&examResult)
	examID := examResult.Data.ID

	// 5. Create Lecture with Uploads
	requestBody := &bytes.Buffer{}
	multipartWriter := multipart.NewWriter(requestBody)
	multipartWriter.WriteField("title", "Lecture 1")
	multipartWriter.WriteField("description", "First lecture")
	
	mediaPart, _ := multipartWriter.CreateFormFile("media", "test-audio.mp3")
	mediaPart.Write([]byte("fake audio content"))
	
	documentPart, _ := multipartWriter.CreateFormFile("documents", "test-slides.pdf")
	documentPart.Write([]byte("fake pdf content"))
	multipartWriter.Close()

	lectureRequest := createAuthenticatedRequest("POST", fmt.Sprintf("%s/api/exams/%s/lectures", testServer.URL, examID), requestBody)
	lectureRequest.Header.Set("Content-Type", multipartWriter.FormDataContentType())
	lectureResponse, err := httpClient.Do(lectureRequest)
	if err != nil || lectureResponse.StatusCode != http.StatusCreated {
		tester.Fatalf("Lecture creation failed: %v, status: %d", err, lectureResponse.StatusCode)
	}

	var lectureResult struct {
		Data models.Lecture `json:"data"`
	}
	json.NewDecoder(lectureResponse.Body).Decode(&lectureResult)
	lectureID := lectureResult.Data.ID

	// 6. Wait for background processing
	// We poll until status is 'ready' or timeout
	deadline := time.Now().Add(10 * time.Second)
	var lectureStatus string
	for time.Now().Before(deadline) {
		initializedDatabase.QueryRow("SELECT status FROM lectures WHERE id = ?", lectureID).Scan(&lectureStatus)
		if lectureStatus == "ready" {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	if lectureStatus != "ready" {
		tester.Fatalf("Lecture failed to reach 'ready' status, got %q", lectureStatus)
	}

	// 7. Verify Results in Database

	var segmentCount int
	initializedDatabase.QueryRow("SELECT COUNT(*) FROM transcript_segments WHERE transcript_id = (SELECT id FROM transcripts WHERE lecture_id = ?)", lectureID).Scan(&segmentCount)
	if segmentCount == 0 {
		tester.Errorf("Expected transcript segments, found 0")
	}

	var pageCount int
	initializedDatabase.QueryRow("SELECT COUNT(*) FROM reference_pages WHERE document_id = (SELECT id FROM reference_documents WHERE lecture_id = ?)", lectureID).Scan(&pageCount)
	if pageCount == 0 {
		tester.Errorf("Expected reference pages, found 0")
	}

	// 8. Test Tool Generation (Study Guide)
	// We need to mock more specific LLM responses for the sequential generator
	// Analysis, then segments...
	
	mockLLM.ResponseText = `# Outline
## Introduction
Coverage: Basics
Introduces: 
- Concept 1 - Emphasis: High (Spent lots of time)
`
	
	toolPayload, _ := json.Marshal(map[string]string{
		"lecture_id": lectureID,
		"type":       "guide",
	})
	toolRequest := createAuthenticatedRequest("POST", fmt.Sprintf("%s/api/exams/%s/tools", testServer.URL, examID), bytes.NewBuffer(toolPayload))
	toolRequest.Header.Set("Content-Type", "application/json")
	toolResponse, err := httpClient.Do(toolRequest)
	if err != nil || toolResponse.StatusCode != http.StatusAccepted {
		tester.Fatalf("Tool generation failed: %v, status: %d", err, toolResponse.StatusCode)
	}

	// Wait for tool generation job
	time.Sleep(1 * time.Second)

	var toolCount int
	initializedDatabase.QueryRow("SELECT COUNT(*) FROM tools WHERE exam_id = ?", examID).Scan(&toolCount)
	if toolCount != 1 {
		tester.Errorf("Expected 1 tool in database, found %d", toolCount)
	}
}