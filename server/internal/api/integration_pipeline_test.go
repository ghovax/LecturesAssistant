//go:build integration

package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

// To run this test:
// 1. Copy server/configuration.yaml.example to server/configuration.yaml and add your API keys
// 2. Place 'test_audio.mp3' and 'test_document.pdf' in server/internal/api/test_input/
// 3. Run: make test-integration
//    OR: go test -v -tags=integration ./internal/api/...

func TestFullPipeline_RealProviders(tester *testing.T) {
	if testing.Short() {
		tester.Skip("Skipping real provider test in short mode")
	}

	// Change to server root directory for test execution
	// This ensures relative paths like "xelatex-template.tex" work correctly
	originalDir, _ := os.Getwd()
	serverRoot := filepath.Join(originalDir, "..", "..")
	os.Chdir(serverRoot)
	defer os.Chdir(originalDir)

	// 1. Setup Environment
	inputDirectory := filepath.Join(originalDir, "test_input")
	audioPath := filepath.Join(inputDirectory, "Where Are You From Part 1 - Dialogue ( lingoneo.org ).mp3")
	documentPath := filepath.Join(inputDirectory, "Where Are You From Part 1 - Dialogue ( lingoneo.org ).pdf")

	// Make paths absolute since we changed directory
	audioPath, _ = filepath.Abs(audioPath)
	documentPath, _ = filepath.Abs(documentPath)

	if _, statError := os.Stat(audioPath); os.IsNotExist(statError) {
		tester.Fatalf("Missing test audio at %s. Please provide a real audio file.", audioPath)
	}
	if _, statError := os.Stat(documentPath); os.IsNotExist(statError) {
		tester.Fatalf("Missing test document at %s. Please provide a real PDF file.", documentPath)
	}

	// Use real configuration (now in current directory since we changed to server root)
	config, loadError := configuration.Load("configuration.yaml")
	if loadError != nil {
		tester.Fatalf("Failed to load configuration: %v", loadError)
	}

	// Override storage to a local 'test_run_data' folder for inspection
	// Use absolute path based on original directory
	testRunDataDir := filepath.Join(originalDir, "test_integration_pipeline_results")
	testRunDataDir, _ = filepath.Abs(testRunDataDir)
	os.RemoveAll(testRunDataDir)
	config.Storage.DataDirectory = testRunDataDir
	os.MkdirAll(filepath.Join(testRunDataDir, "files", "lectures"), 0755)
	os.MkdirAll(filepath.Join(os.TempDir(), "lectures-uploads"), 0755)

	// Setup detailed logging to file for debugging
	logFile, _ := os.Create(filepath.Join(testRunDataDir, "test_debug.log"))
	defer logFile.Close()

	logger := slog.New(slog.NewJSONHandler(logFile, nil))
	slog.SetDefault(logger)

	// 2. Initialize Real Components
	initializedDatabase, _ := database.Initialize(filepath.Join(testRunDataDir, "test_database.db"))
	defer initializedDatabase.Close()

	promptManager := prompts.NewManager("prompts")
	llmProvider := llm.NewOpenRouterProvider(config.Providers.OpenRouter.APIKey)

	// Use OpenRouter's chat API for transcription instead of Whisper endpoint
	transcriptionModel := config.Transcription.GetModel(&config.LLM)
	transcriptionProvider := transcription.NewOpenRouterTranscriptionProvider(
		llmProvider,
		transcriptionModel,
	)
	transcriptionService := transcription.NewService(config, transcriptionProvider, llmProvider, promptManager)

	documentProcessor := documents.NewProcessor(llmProvider, config.LLM.GetModelForTask("documents_ingestion"), promptManager)
	markdownConverter := markdown.NewConverter(testRunDataDir)
	toolGenerator := tools.NewToolGenerator(config, llmProvider, promptManager)

	jobQueue := jobs.NewQueue(initializedDatabase, 1)
	jobs.RegisterHandlers(jobQueue, initializedDatabase, config, transcriptionService, documentProcessor, toolGenerator, markdownConverter, database.CheckLectureReadiness)
	jobQueue.Start()
	defer jobQueue.Stop()

	apiServer := NewServer(config, initializedDatabase, jobQueue, llmProvider, promptManager, toolGenerator)
	testServer := httptest.NewServer(apiServer.Handler())
	defer testServer.Close()

	httpClient := testServer.Client()

	// 3. Execution Flow

	// A. Auth Setup & Login
	tester.Log("Setting up user...")
	setupPayload, _ := json.Marshal(map[string]string{"username": "tester", "password": "password123"})
	setupResp, _ := httpClient.Post(testServer.URL+"/api/auth/setup", "application/json", bytes.NewBuffer(setupPayload))
	if setupResp.StatusCode != http.StatusOK && setupResp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(setupResp.Body)
		tester.Fatalf("Setup failed with status %d: %s", setupResp.StatusCode, string(bodyBytes))
	}

	loginPayload, _ := json.Marshal(map[string]string{"username": "tester", "password": "password123"})
	loginResponse, _ := httpClient.Post(testServer.URL+"/api/auth/login", "application/json", bytes.NewBuffer(loginPayload))
	if loginResponse.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(loginResponse.Body)
		tester.Fatalf("Login failed with status %d: %s", loginResponse.StatusCode, string(bodyBytes))
	}
	var loginData struct{ Data struct{ Token string } }
	if err := json.NewDecoder(loginResponse.Body).Decode(&loginData); err != nil {
		tester.Fatalf("Failed to decode login response: %v", err)
	}
	sessionToken := loginData.Data.Token
	if sessionToken == "" {
		tester.Fatalf("Session token is empty after login")
	}
	tester.Logf("Logged in with token: %s...", sessionToken[:20])

	authenticatedRequest := func(method, url string, body io.Reader) *http.Request {
		httpRequest, _ := http.NewRequest(method, url, body)
		httpRequest.Header.Set("Authorization", "Bearer "+sessionToken)
		httpRequest.Header.Set("X-Requested-With", "XMLHttpRequest")
		return httpRequest
	}

	// B. Create Exam
	tester.Log("Creating exam...")
	examPayload, _ := json.Marshal(map[string]string{"title": "Full Run Test Course"})
	examReq := authenticatedRequest("POST", testServer.URL+"/api/exams", bytes.NewBuffer(examPayload))
	examReq.Header.Set("Content-Type", "application/json")
	examResp, _ := httpClient.Do(examReq)
	var examRes struct{ Data models.Exam }
	json.NewDecoder(examResp.Body).Decode(&examRes)
	examID := examRes.Data.ID

	// C. Upload Files (Direct)
	tester.Log("Uploading audio and document...")
	requestBody := &bytes.Buffer{}
	multipartWriter := multipart.NewWriter(requestBody)
	multipartWriter.WriteField("title", "Real World Test Lecture")
	multipartWriter.WriteField("exam_id", examID)
	multipartWriter.WriteField("specified_date", "2026-01-03")

	audioFile, _ := os.Open(audioPath)
	mediaPart, _ := multipartWriter.CreateFormFile("media", filepath.Base(audioPath))
	io.Copy(mediaPart, audioFile)

	documentFile, _ := os.Open(documentPath)
	documentPart, _ := multipartWriter.CreateFormFile("documents", filepath.Base(documentPath))
	io.Copy(documentPart, documentFile)
	multipartWriter.Close()

	lectureReq := authenticatedRequest("POST", testServer.URL+"/api/lectures", requestBody)
	lectureReq.Header.Set("Content-Type", multipartWriter.FormDataContentType())
	lectureResp, uploadError := httpClient.Do(lectureReq)
	if uploadError != nil {
		tester.Fatalf("Lecture upload failed: %v", uploadError)
	}
	if lectureResp.StatusCode != http.StatusOK && lectureResp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(lectureResp.Body)
		tester.Fatalf("Lecture creation failed with status %d: %s", lectureResp.StatusCode, string(bodyBytes))
	}
	var lectureRes struct{ Data models.Lecture }
	if err := json.NewDecoder(lectureResp.Body).Decode(&lectureRes); err != nil {
		tester.Fatalf("Failed to decode lecture response: %v", err)
	}
	lectureID := lectureRes.Data.ID
	if lectureID == "" {
		tester.Fatalf("Lecture ID is empty after creation")
	}
	tester.Logf("Created lecture with ID: %s", lectureID)

	// D. Wait for Processing (Transcription + Ingestion)
	tester.Log("Waiting for AI ingestion of media and documents...")
	deadline := time.Now().Add(10 * time.Minute)
	var lectureStatus string
	for time.Now().Before(deadline) {
		_ = initializedDatabase.QueryRow("SELECT status FROM lectures WHERE id = ?", lectureID).Scan(&lectureStatus)
		if lectureStatus == "ready" {
			break
		}
		if lectureStatus == "failed" {
			tester.Fatal("Lecture processing failed")
		}
		time.Sleep(5 * time.Second)
	}

	// E. Trigger Study Guide Generation
	tester.Log("Triggering study guide generation...")
	toolPayload, _ := json.Marshal(map[string]string{
		"exam_id":       examID,
		"lecture_id":    lectureID,
		"type":          "guide",
		"length":        "short",
		"language_code": "de",
	})
	toolReq := authenticatedRequest("POST", testServer.URL+"/api/tools", bytes.NewBuffer(toolPayload))
	toolReq.Header.Set("Content-Type", "application/json")
	toolResp, _ := httpClient.Do(toolReq)
	var toolJobRes struct {
		Data struct {
			JobID string `json:"job_id"`
		}
	}
	json.NewDecoder(toolResp.Body).Decode(&toolJobRes)
	jobID := toolJobRes.Data.JobID

	// Wait for Generation
	tester.Log("Polling generation job status...")
	var toolID string
	for time.Now().Before(deadline) {
		var status, result string
		_ = initializedDatabase.QueryRow("SELECT status, result FROM jobs WHERE id = ?", jobID).Scan(&status, &result)
		if status == "COMPLETED" {
			var resultData map[string]string
			json.Unmarshal([]byte(result), &resultData)
			toolID = resultData["tool_id"]
			break
		}
		if status == "FAILED" {
			var errorString string
			_ = initializedDatabase.QueryRow("SELECT error FROM jobs WHERE id = ?", jobID).Scan(&errorString)
			tester.Fatalf("Generation job failed: %s", errorString)
		}
		time.Sleep(2 * time.Second)
	}

	// F. Export to PDF
	tester.Log("Exporting to PDF...")
	exportPayload, _ := json.Marshal(map[string]string{
		"tool_id": toolID,
		"exam_id": examID,
	})
	exportReq := authenticatedRequest("POST", testServer.URL+"/api/tools/export", bytes.NewBuffer(exportPayload))
	exportReq.Header.Set("Content-Type", "application/json")
	exportResp, _ := httpClient.Do(exportReq)
	var exportJobRes struct {
		Data struct {
			JobID string `json:"job_id"`
		}
	}
	json.NewDecoder(exportResp.Body).Decode(&exportJobRes)
	publishJobID := exportJobRes.Data.JobID

	var pdfPath string
	for time.Now().Before(deadline) {
		var status, result string
		_ = initializedDatabase.QueryRow("SELECT status, result FROM jobs WHERE id = ?", publishJobID).Scan(&status, &result)
		if status == "COMPLETED" {
			var resultData map[string]string
			json.Unmarshal([]byte(result), &resultData)
			pdfPath = resultData["pdf_path"]
			break
		}
		time.Sleep(2 * time.Second)
	}

	// G. Verify Results
	tester.Log("Verifying output files...")
	if _, statError := os.Stat(pdfPath); statError != nil {
		tester.Errorf("Final PDF not found at %s", pdfPath)
	} else {
		tester.Logf("Success! Final PDF generated at: %s", pdfPath)
	}
}
