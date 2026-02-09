package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"lectures/internal/configuration"
	"lectures/internal/database"
	"lectures/internal/jobs"
	"lectures/internal/tools"
)

func setupHTMLTestEnv(t *testing.T) (*Server, string, string, func()) {
	tempDir, err := os.MkdirTemp("", "html-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tempDir, "test.db")
	db, err := database.Initialize(dbPath)
	if err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}

	// Create user and session
	userID := "user-123"
	sessionID := "session-123"
	_, _ = db.Exec("INSERT INTO users (id, username, password_hash, role) VALUES (?, ?, ?, ?)", userID, "testuser", "hash", "user")
	_, _ = db.Exec("INSERT INTO auth_sessions (id, user_id, created_at, last_activity, expires_at) VALUES (?, ?, ?, ?, ?)", sessionID, userID, time.Now(), time.Now(), time.Now().Add(1*time.Hour))

	config := &configuration.Configuration{
		Storage: configuration.StorageConfiguration{DataDirectory: tempDir},
		Security: configuration.SecurityConfiguration{
			Auth: configuration.AuthConfiguration{Type: "session"},
		},
	}

	jobQueue := jobs.NewQueue(db, 1)
	mockLLM := &MockLLMProvider{}
	toolGenerator := tools.NewToolGenerator(config, mockLLM, nil)
	
	// Real converter might need pandoc/tectonic installed, so we use the mock
	mockConverter := &MockMarkdownConverter{}

	server := NewServer(config, db, jobQueue, mockLLM, nil, toolGenerator, mockConverter)

	cleanup := func() {
		db.Close()
		os.RemoveAll(tempDir)
	}

	return server, userID, sessionID, cleanup
}

func TestHandleGetTranscriptHTML(t *testing.T) {
	server, _, sessionID, cleanup := setupHTMLTestEnv(t)
	defer cleanup()

	// 1. Setup mock data
	examID := "exam-1"
	lectureID := "lecture-1"
	_, _ = server.database.Exec("INSERT INTO exams (id, user_id, title) VALUES (?, ?, ?)", examID, "user-123", "Test Exam")
	_, _ = server.database.Exec("INSERT INTO lectures (id, exam_id, title, status) VALUES (?, ?, ?, ?)", lectureID, examID, "Test Lecture", "ready")
	
	transcriptID := "trans-1"
	_, _ = server.database.Exec("INSERT INTO transcripts (id, lecture_id, status) VALUES (?, ?, ?)", transcriptID, lectureID, "completed")
	_, _ = server.database.Exec("INSERT INTO transcript_segments (transcript_id, text, start_millisecond, end_millisecond) VALUES (?, ?, ?, ?)", transcriptID, "Hello world.", 0, 1000)

	// 2. Make request
	req := httptest.NewRequest("GET", "/api/transcripts/html?lecture_id="+lectureID, nil)
	req.Header.Set("Authorization", "Bearer "+sessionID)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	
	rr := httptest.NewRecorder()
	server.Handler().ServeHTTP(rr, req)

	// 3. Verify
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}
	
	var apiResponse struct {
		Data struct {
			TranscriptID string `json:"transcript_id"`
			Status       string `json:"status"`
			Segments     []struct {
				ID               int    `json:"id"`
				StartMillisecond int64  `json:"start_millisecond"`
				EndMillisecond   int64  `json:"end_millisecond"`
				TextHTML         string `json:"text_html"`
			} `json:"segments"`
		} `json:"data"`
	}
	json.NewDecoder(rr.Body).Decode(&apiResponse)

	if apiResponse.Data.TranscriptID != transcriptID {
		t.Errorf("Expected transcript_id %s, got %s", transcriptID, apiResponse.Data.TranscriptID)
	}
	
	if len(apiResponse.Data.Segments) != 1 {
		t.Fatalf("Expected 1 segment, got %d", len(apiResponse.Data.Segments))
	}

	if !strings.Contains(apiResponse.Data.Segments[0].TextHTML, "Hello world") {
		t.Errorf("Segment TextHTML missing content: %s", apiResponse.Data.Segments[0].TextHTML)
	}
	
	if apiResponse.Data.Segments[0].StartMillisecond != 0 {
		t.Errorf("Expected start_millisecond 0, got %d", apiResponse.Data.Segments[0].StartMillisecond)
	}
}

func TestHandleGetPageHTML(t *testing.T) {
	server, _, sessionID, cleanup := setupHTMLTestEnv(t)
	defer cleanup()

	// 1. Setup mock data
	examID := "exam-1"
	lectureID := "lecture-1"
	docID := "doc-1"
	_, _ = server.database.Exec("INSERT INTO exams (id, user_id, title) VALUES (?, ?, ?)", examID, "user-123", "Test Exam")
	_, _ = server.database.Exec("INSERT INTO lectures (id, exam_id, title, status) VALUES (?, ?, ?, ?)", lectureID, examID, "Test Lecture", "ready")
	_, _ = server.database.Exec("INSERT INTO reference_documents (id, lecture_id, document_type, title, file_path, page_count, extraction_status) VALUES (?, ?, 'pdf', 'Test Doc', 'path', 1, 'completed')", docID, lectureID)
	_, _ = server.database.Exec("INSERT INTO reference_pages (document_id, page_number, image_path, extracted_text) VALUES (?, ?, ?, ?)", docID, 1, "img-path", "This is page content.")

	// 2. Make request
	url := fmt.Sprintf("/api/documents/pages/html?document_id=%s&lecture_id=%s&page_number=1", docID, lectureID)
	req := httptest.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+sessionID)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	
	rr := httptest.NewRecorder()
	server.Handler().ServeHTTP(rr, req)

	// 3. Verify
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}
	
	if rr.Header().Get("Content-Type") != "text/html" {
		t.Errorf("Expected content-type text/html, got %s", rr.Header().Get("Content-Type"))
	}

	if !strings.Contains(rr.Body.String(), "This is page content") {
		t.Errorf("Response body missing page content: %s", rr.Body.String())
	}
}

func TestHandleGetToolHTML_Guide(t *testing.T) {
	server, _, sessionID, cleanup := setupHTMLTestEnv(t)
	defer cleanup()

	// 1. Setup mock data
	examID := "exam-1"
	toolID := "tool-1"
	content := `# Top Title

## Section 1

Content of section 1.`
	_, _ = server.database.Exec("INSERT INTO exams (id, user_id, title) VALUES (?, ?, ?)", examID, "user-123", "Test Exam")
	_, _ = server.database.Exec("INSERT INTO tools (id, exam_id, type, title, content) VALUES (?, ?, 'guide', 'Test Guide', ?)", toolID, examID, content)

	// 2. Make request
	url := fmt.Sprintf("/api/tools/html?tool_id=%s&exam_id=%s", toolID, examID)
	req := httptest.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+sessionID)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	
	rr := httptest.NewRecorder()
	server.Handler().ServeHTTP(rr, req)

	// 3. Verify
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
	
	var apiResponse struct {
		Data struct {
			ToolID      string `json:"tool_id"`
			Type        string `json:"type"`
			ContentHTML string `json:"content_html"`
		} `json:"data"`
	}
	json.NewDecoder(rr.Body).Decode(&apiResponse)

	if apiResponse.Data.ToolID != toolID {
		t.Errorf("Expected tool_id %s, got %s", toolID, apiResponse.Data.ToolID)
	}

	// Verify title was stripped from HTML content
	if strings.Contains(apiResponse.Data.ContentHTML, "Top Title") {
		t.Errorf("ContentHTML should not contain the top-level title. Got: %s", apiResponse.Data.ContentHTML)
	}
	
	if !strings.Contains(apiResponse.Data.ContentHTML, "Section 1") {
		t.Errorf("ContentHTML missing section content. Got: %s", apiResponse.Data.ContentHTML)
	}
}

func TestHandleGetToolHTML_Flashcards(t *testing.T) {
	server, _, sessionID, cleanup := setupHTMLTestEnv(t)
	defer cleanup()

	// 1. Setup mock data
	examID := "exam-1"
	toolID := "tool-1"
	flashcards := []map[string]string{
		{"front": "Front 1", "back": "Back 1"},
	}
	fcJSON, _ := json.Marshal(flashcards)
	_, _ = server.database.Exec("INSERT INTO exams (id, user_id, title) VALUES (?, ?, ?)", examID, "user-123", "Test Exam")
	_, _ = server.database.Exec("INSERT INTO tools (id, exam_id, type, title, content) VALUES (?, ?, 'flashcard', 'Test FC', ?)", toolID, examID, string(fcJSON))

	// 2. Make request
	url := fmt.Sprintf("/api/tools/html?tool_id=%s&exam_id=%s", toolID, examID)
	req := httptest.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+sessionID)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	
	rr := httptest.NewRecorder()
	server.Handler().ServeHTTP(rr, req)

	// 3. Verify
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
	
	var apiResponse struct {
		Data struct {
			Type    string `json:"type"`
			Content []struct {
				FrontHTML string `json:"front_html"`
				BackHTML  string `json:"back_html"`
			} `json:"content"`
		} `json:"data"`
	}
	json.NewDecoder(rr.Body).Decode(&apiResponse)

	if len(apiResponse.Data.Content) != 1 {
		t.Fatalf("Expected 1 flashcard, got %d", len(apiResponse.Data.Content))
	}

	if !strings.Contains(apiResponse.Data.Content[0].FrontHTML, "Front 1") {
		t.Errorf("FrontHTML incorrect: %s", apiResponse.Data.Content[0].FrontHTML)
	}
}

func TestHandleGetToolHTML_Quiz(t *testing.T) {
	server, _, sessionID, cleanup := setupHTMLTestEnv(t)
	defer cleanup()

	// 1. Setup mock data
	examID := "exam-1"
	toolID := "tool-1"
	quiz := []map[string]any{
		{
			"question": "Q1",
			"options": []string{"O1", "O2"},
			"correct_answer": "O1",
			"explanation": "Exp 1",
		},
	}
	quizJSON, _ := json.Marshal(quiz)
	_, _ = server.database.Exec("INSERT INTO exams (id, user_id, title) VALUES (?, ?, ?)", examID, "user-123", "Test Exam")
	_, _ = server.database.Exec("INSERT INTO tools (id, exam_id, type, title, content) VALUES (?, ?, 'quiz', 'Test Quiz', ?)", toolID, examID, string(quizJSON))

	// 2. Make request
	url := fmt.Sprintf("/api/tools/html?tool_id=%s&exam_id=%s", toolID, examID)
	req := httptest.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+sessionID)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	
	rr := httptest.NewRecorder()
	server.Handler().ServeHTTP(rr, req)

	// 3. Verify
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
	
	var apiResponse struct {
		Data struct {
			Type    string `json:"type"`
			Content []struct {
				QuestionHTML      string   `json:"question_html"`
				OptionsHTML       []string `json:"options_html"`
				CorrectAnswerHTML string   `json:"correct_answer_html"`
				ExplanationHTML   string   `json:"explanation_html"`
			} `json:"content"`
		} `json:"data"`
	}
	json.NewDecoder(rr.Body).Decode(&apiResponse)

	if len(apiResponse.Data.Content) != 1 {
		t.Fatalf("Expected 1 quiz item, got %d", len(apiResponse.Data.Content))
	}

	if !strings.Contains(apiResponse.Data.Content[0].QuestionHTML, "Q1") {
		t.Errorf("QuestionHTML incorrect: %s", apiResponse.Data.Content[0].QuestionHTML)
	}
	
	if len(apiResponse.Data.Content[0].OptionsHTML) != 2 {
		t.Errorf("Expected 2 options, got %d", len(apiResponse.Data.Content[0].OptionsHTML))
	}
}