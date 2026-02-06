//go:build integration

package api

import (
	"bytes"
	"encoding/json"
	"io"
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
// 1. Place 'test_audio.mp3' and 'test_doc.pdf' in server/internal/api/test_input/
// 2. Ensure OpenRouter API key is set in server/configuration.yaml
// 3. Run: go test -v -tags=integration ./internal/api/integration_pipeline_test.go [other api files...]
//    OR more simply: go test -v -tags=integration ./internal/api/...

func TestFullPipeline_RealProviders(tester *testing.T) {
	if testing.Short() {
		tester.Skip("Skipping real provider test in short mode")
	}

	// 1. Setup Environment
	inputDirectory := filepath.Join("test_input")
	audioPath := filepath.Join(inputDirectory, "test_audio.mp3")
	documentPath := filepath.Join(inputDirectory, "test_document.pdf")

	if _, statError := os.Stat(audioPath); os.IsNotExist(statError) {
		tester.Fatalf("Missing test audio at %s. Please provide a real audio file.", audioPath)
	}
	if _, statError := os.Stat(documentPath); os.IsNotExist(statError) {
		tester.Fatalf("Missing test document at %s. Please provide a real PDF file.", documentPath)
	}

	// Use real configuration
	config, loadError := configuration.Load("../../configuration.yaml")
	if loadError != nil {
		tester.Fatalf("Failed to load configuration: %v", loadError)
	}

	// Override storage to a local 'test_run_data' folder for inspection
	testRunDataDir, _ := filepath.Abs("test_integration_pipeline_results")
	os.RemoveAll(testRunDataDir)
	config.Storage.DataDirectory = testRunDataDir
	os.MkdirAll(filepath.Join(testRunDataDir, "files", "lectures"), 0755)
	os.MkdirAll(filepath.Join(testRunDataDir, "tmp", "uploads"), 0755)

	// 2. Initialize Real Components
	initializedDatabase, _ := database.Initialize(filepath.Join(testRunDataDir, "test_database.db"))
	defer initializedDatabase.Close()

	promptManager := prompts.NewManager("../../prompts")
	llmProvider := llm.NewOpenRouterProvider(config.Providers.OpenRouter.APIKey)

	// Use OpenRouter's chat API for transcription instead of Whisper endpoint
	transcriptionProvider := transcription.NewOpenRouterTranscriptionProvider(
		llmProvider,
		config.Transcription.Model,
	)
	transcriptionService := transcription.NewService(config, transcriptionProvider, llmProvider, promptManager)

	documentProcessor := documents.NewProcessor(llmProvider, config.LLM.Models.Ingestion, promptManager)
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
	httpClient.Post(testServer.URL+"/api/auth/setup", "application/json", bytes.NewBuffer(setupPayload))

	loginPayload, _ := json.Marshal(map[string]string{"username": "tester", "password": "password123"})
	loginResponse, _ := httpClient.Post(testServer.URL+"/api/auth/login", "application/json", bytes.NewBuffer(loginPayload))
	var loginData struct{ Data struct{ Token string } }
	json.NewDecoder(loginResponse.Body).Decode(&loginData)
	sessionToken := loginData.Data.Token

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
	var lectureRes struct{ Data models.Lecture }
	json.NewDecoder(lectureResp.Body).Decode(&lectureRes)
	lectureID := lectureRes.Data.ID

	// D. Wait for Processing (Transcription + Ingestion)
	tester.Log("Waiting for AI ingestion (transcription and OCR)...")
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
		"exam_id":    examID,
		"lecture_id": lectureID,
		"type":       "guide",
		"length":     "short",
	})
	toolReq := authenticatedRequest("POST", testServer.URL+"/api/tools", bytes.NewBuffer(toolPayload))
	toolReq.Header.Set("Content-Type", "application/json")
	toolResp, _ := httpClient.Do(toolReq)
	var toolJobRes struct {
		Data struct {
			JobID string `json:"job_id"`
		}
	}
	json.NewDecoder(toolJobRes.Body).Decode(&toolJobRes)
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
	// Actually, the API doesn't have a direct 'export' endpoint yet, but the handler is in internal/jobs.
	// Let's trigger it via the queue directly for the test.
	publishJobID, _ := jobQueue.Enqueue("tester", models.JobTypePublishMaterial, map[string]string{"tool_id": toolID})

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
		// Copy result to project root for easy access
		finalPDF := "test_output_result.pdf"
		inputData, _ := os.ReadFile(pdfPath)
		os.WriteFile(finalPDF, inputData, 0644)
		tester.Logf("Final PDF copied to project root as %s", finalPDF)
	}
}
