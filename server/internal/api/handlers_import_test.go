package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"lectures/internal/configuration"
	"lectures/internal/database"
	"lectures/internal/jobs"
	"lectures/internal/models"
	"lectures/internal/tools"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

// Helper to setup test environment
func setupImportTestEnv(t *testing.T) (*Server, *jobs.Queue, string, func()) {
	// 1. Temp Dir
	tempDir, err := os.MkdirTemp("", "import-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// 2. Database
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := database.Initialize(dbPath)
	if err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}

	// 3. User & Session
	userID := "test-user-id"
	_ = gonanoid.Must()
	_, _ = db.Exec("INSERT INTO users (id, username, password_hash, role) VALUES (?, ?, ?, ?)", userID, "testuser", "hash", "user")

	// 4. Job Queue
	jobQueue := jobs.NewQueue(db, 1) // 1 worker
	// We don't start the queue here because we just want to verify enqueuing, not execution.
	// But if we wanted execution, we'd call jobQueue.Start()

	// 5. Server
	config := &configuration.Configuration{
		Storage: configuration.StorageConfiguration{DataDirectory: tempDir},
		Security: configuration.SecurityConfiguration{
			Auth: configuration.AuthConfiguration{Type: "session"},
		},
	}

	// Mock dependencies for Server
	mockLLM := &MockLLMProvider{}
	toolGenerator := tools.NewToolGenerator(config, mockLLM, nil)

	server := NewServer(config, db, jobQueue, mockLLM, nil, toolGenerator, &MockMarkdownConverter{})

	cleanup := func() {
		db.Close()
		os.RemoveAll(tempDir)
	}

	return server, jobQueue, userID, cleanup
}

func TestHandleImport_GoogleDrive(t *testing.T) {
	server, _, userID, cleanup := setupImportTestEnv(t)
	defer cleanup()

	tests := []struct {
		name           string
		payload        map[string]any
		expectedStatus int
		expectedJob    bool
	}{
		{
			name: "Valid Google Drive Request",
			payload: map[string]any{
				"source":   "google_drive",
				"filename": "lecture.mp4",
				"data": map[string]string{
					"file_id":     "12345",
					"oauth_token": "valid-token",
				},
			},
			expectedStatus: http.StatusAccepted,
			expectedJob:    true,
		},
		{
			name: "Missing Source",
			payload: map[string]any{
				"filename": "lecture.mp4",
				"data": map[string]string{
					"file_id": "12345",
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectedJob:    false,
		},
		{
			name: "Unsupported Source",
			payload: map[string]any{
				"source": "dropbox",
				"data":   map[string]string{},
			},
			expectedStatus: http.StatusBadRequest,
			expectedJob:    false,
		},
		{
			name: "Google Drive Missing Data",
			payload: map[string]any{
				"source": "google_drive",
				"data": map[string]string{
					"file_id": "12345",
					// Missing oauth_token
				},
			},
			expectedStatus: http.StatusBadRequest,
			expectedJob:    false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			requestBody, _ := json.Marshal(testCase.payload)
			importRequest, _ := http.NewRequest("POST", "/api/uploads/import", bytes.NewBuffer(requestBody))

			// Manually inject user ID into context since we are bypassing auth middleware for this unit test
			// and calling the handler method directly would be tricky without the router.
			// However, since handleImport calls server.getUserID(request), and we are in package api,
			// we can access the private contextKey `userIDKey`.

			requestContext := context.WithValue(importRequest.Context(), userIDKey, userID)
			importRequest = importRequest.WithContext(requestContext)

			responseRecorder := httptest.NewRecorder()

			// Call the handler directly
			server.handleImport(responseRecorder, importRequest)

			if responseRecorder.Code != testCase.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", testCase.expectedStatus, responseRecorder.Code, responseRecorder.Body.String())
			}

			if testCase.expectedJob {
				var apiResponse struct {
					Data struct {
						JobID string `json:"job_id"`
					} `json:"data"`
				}
				if unmarshalError := json.Unmarshal(responseRecorder.Body.Bytes(), &apiResponse); unmarshalError != nil {
					t.Fatalf("Failed to unmarshal response: %v. Body: %s", unmarshalError, responseRecorder.Body.String())
				}
				jobID := apiResponse.Data.JobID

				if jobID == "" {
					t.Error("Expected job_id in response, got empty")
				} else {
					// Verify job exists in DB
					job, err := server.jobQueue.GetJob(jobID)
					if err != nil {
						t.Errorf("Failed to get job %s: %v", jobID, err)
					}
					if job.Type != models.JobTypeDownloadGoogleDrive {
						t.Errorf("Expected job type %s, got %s", models.JobTypeDownloadGoogleDrive, job.Type)
					}
				}
			}
		})
	}
}
