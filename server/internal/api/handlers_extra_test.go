package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"lectures/internal/configuration"
	"lectures/internal/database"
	"lectures/internal/jobs"
	"lectures/internal/tools"

	"golang.org/x/crypto/bcrypt"
)

func setupUniqueExtraTestEnv(t *testing.T, testName string) (*Server, string, string, func()) {
	tempDir, err := os.MkdirTemp("", "extra-test-"+testName+"-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tempDir, "test.db")
	db, err := database.Initialize(dbPath)
	if err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}

	// Create user and session
	userID := "user-" + testName
	sessionID := "session-" + testName
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	_, _ = db.Exec("INSERT INTO users (id, username, password_hash, role) VALUES (?, ?, ?, ?)", userID, "user"+testName, string(hash), "user")
	_, _ = db.Exec("INSERT INTO auth_sessions (id, user_id, created_at, last_activity, expires_at) VALUES (?, ?, ?, ?, ?)", sessionID, userID, time.Now(), time.Now(), time.Now().Add(1*time.Hour))

	config := &configuration.Configuration{
		Storage: configuration.StorageConfiguration{DataDirectory: tempDir},
		Security: configuration.SecurityConfiguration{
			Auth: configuration.AuthConfiguration{Type: "session"},
		},
	}

	jobQueue := jobs.NewQueue(db, 1)
	mockLLM := &MockLLMProvider{ResponseText: `{"title": "Suggested Title", "description": "Suggested Description"}`}
	toolGenerator := tools.NewToolGenerator(config, mockLLM, nil)

	// Register handlers for Suggest test
	jobs.RegisterHandlers(jobQueue, db, config, nil, nil, toolGenerator, &MockMarkdownConverter{}, nil, nil)

	jobQueue.Start()

	server := NewServer(config, db, jobQueue, mockLLM, nil, toolGenerator, &MockMarkdownConverter{})

	cleanup := func() {
		jobQueue.Stop()
		db.Close()
		os.RemoveAll(tempDir)
	}

	return server, userID, sessionID, cleanup
}

func TestHandleAuthChangePassword(t *testing.T) {
	server, _, sessionID, cleanup := setupUniqueExtraTestEnv(t, "password")
	defer cleanup()

	payload := map[string]string{
		"current_password": "password123",
		"new_password":     "newpassword123",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("PATCH", "/api/auth/password", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+sessionID)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	rr := httptest.NewRecorder()
	server.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleUpdateTool(t *testing.T) {
	server, userID, sessionID, cleanup := setupUniqueExtraTestEnv(t, "updatetool")
	defer cleanup()

	examID := "exam-1"
	toolID := "tool-1"
	_, _ = server.database.Exec("INSERT INTO exams (id, user_id, title) VALUES (?, ?, ?)", examID, userID, "Test Exam")
	_, _ = server.database.Exec("INSERT INTO tools (id, exam_id, type, title, content) VALUES (?, ?, 'guide', 'Old Title', 'Old Content')", toolID, examID)

	newTitle := "New Polished Title"
	payload := map[string]any{
		"tool_id": toolID,
		"exam_id": examID,
		"title":   newTitle,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("PATCH", "/api/tools/details", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+sessionID)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	rr := httptest.NewRecorder()
	server.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var title string
	server.database.QueryRow("SELECT title FROM tools WHERE id = ?", toolID).Scan(&title)
	if title != newTitle {
		t.Errorf("Expected title %s, got %s", newTitle, title)
	}
}

func TestHandleUpdateTranscript(t *testing.T) {
	server, userID, sessionID, cleanup := setupUniqueExtraTestEnv(t, "updatetrans")
	defer cleanup()

	examID := "exam-1"
	lectureID := "lecture-1"
	transcriptID := "trans-1"
	_, _ = server.database.Exec("INSERT INTO exams (id, user_id, title) VALUES (?, ?, ?)", examID, userID, "Test Exam")
	_, _ = server.database.Exec("INSERT INTO lectures (id, exam_id, title, status) VALUES (?, ?, ?, ?)", lectureID, examID, "Test Lecture", "ready")
	_, _ = server.database.Exec("INSERT INTO transcripts (id, lecture_id, status) VALUES (?, ?, ?)", transcriptID, lectureID, "completed")
	_, _ = server.database.Exec("INSERT INTO transcript_segments (id, transcript_id, text, start_millisecond, end_millisecond) VALUES (?, ?, ?, ?, ?)", 1, transcriptID, "Old Text", 0, 1000)

	payload := map[string]any{
		"transcript_id": transcriptID,
		"lecture_id":    lectureID,
		"segments": []map[string]any{
			{"id": 1, "text": "New Corrected Text"},
		},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("PATCH", "/api/transcripts", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+sessionID)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	rr := httptest.NewRecorder()
	server.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var text string
	server.database.QueryRow("SELECT text FROM transcript_segments WHERE id = 1").Scan(&text)
	if text != "New Corrected Text" {
		t.Errorf("Expected text 'New Corrected Text', got %s", text)
	}
}

func TestHandleDeleteMedia(t *testing.T) {
	server, userID, sessionID, cleanup := setupUniqueExtraTestEnv(t, "delmedia")
	defer cleanup()

	examID := "exam-1"
	lectureID := "lecture-1"
	mediaID := "media-1"
	_, _ = server.database.Exec("INSERT INTO exams (id, user_id, title) VALUES (?, ?, ?)", examID, userID, "Test Exam")
	_, _ = server.database.Exec("INSERT INTO lectures (id, exam_id, title, status) VALUES (?, ?, ?, ?)", lectureID, examID, "Test Lecture", "ready")
	_, _ = server.database.Exec("INSERT INTO lecture_media (id, lecture_id, media_type, sequence_order, file_path) VALUES (?, ?, 'audio', 0, 'path')", mediaID, lectureID)

	payload := map[string]string{
		"media_id":   mediaID,
		"lecture_id": lectureID,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("DELETE", "/api/media", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+sessionID)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	rr := httptest.NewRecorder()
	server.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var count int
	server.database.QueryRow("SELECT COUNT(*) FROM lecture_media WHERE id = ?", mediaID).Scan(&count)
	if count != 0 {
		t.Errorf("Media was not deleted from database")
	}
}

func TestHandleExamSearch(t *testing.T) {
	server, userID, sessionID, cleanup := setupUniqueExtraTestEnv(t, "search")
	defer cleanup()

	examID := "exam-1"
	lectureID := "lecture-1"
	transcriptID := "trans-1"
	_, _ = server.database.Exec("INSERT INTO exams (id, user_id, title) VALUES (?, ?, ?)", examID, userID, "Test Exam")
	_, _ = server.database.Exec("INSERT INTO lectures (id, exam_id, title, status) VALUES (?, ?, ?, ?)", lectureID, examID, "Test Lecture", "ready")
	_, _ = server.database.Exec("INSERT INTO transcripts (id, lecture_id, status) VALUES (?, ?, ?)", transcriptID, lectureID, "completed")
	_, _ = server.database.Exec("INSERT INTO transcript_segments (transcript_id, text, start_millisecond, end_millisecond) VALUES (?, 'This is a unique-search-keyword test.', 0, 1000)", transcriptID)

	req := httptest.NewRequest("GET", "/api/exams/search?exam_id="+examID+"&query=unique-search-keyword", nil)
	req.Header.Set("Authorization", "Bearer "+sessionID)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	rr := httptest.NewRecorder()
	server.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var apiResponse struct {
		Data []map[string]any `json:"data"`
	}
	json.NewDecoder(rr.Body).Decode(&apiResponse)
	if len(apiResponse.Data) == 0 {
		t.Errorf("Expected search results, got none. Body: %s", rr.Body.String())
	}
}

func TestHandleExamSuggest(t *testing.T) {
	server, userID, sessionID, cleanup := setupUniqueExtraTestEnv(t, "suggest")
	defer cleanup()

	examID := "exam-1"
	_, _ = server.database.Exec("INSERT INTO exams (id, user_id, title) VALUES (?, ?, ?)", examID, userID, "Test Exam")

	payload := map[string]string{"exam_id": examID}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/exams/suggest", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+sessionID)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	rr := httptest.NewRecorder()
	server.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Errorf("Expected status 202, got %d. Body: %s", rr.Code, rr.Body.String())
	}
}
