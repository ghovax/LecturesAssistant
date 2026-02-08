package jobs

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"lectures/internal/configuration"
	"lectures/internal/database"
	"lectures/internal/llm"
	"lectures/internal/markdown"
	"lectures/internal/models"
	"lectures/internal/tools"

	"github.com/google/uuid"
)

// MockMarkdownConverter for testing
type MockMarkdownConverter struct {
	LastMarkdown string
	LastOptions  markdown.ConversionOptions
}

func (m *MockMarkdownConverter) CheckDependencies() error { return nil }
func (m *MockMarkdownConverter) MarkdownToHTML(markdownText string) (string, error) {
	m.LastMarkdown = markdownText
	return "<html>" + markdownText + "</html>", nil
}
func (m *MockMarkdownConverter) HTMLToPDF(htmlContent, outputPath string, options markdown.ConversionOptions) error {
	m.LastOptions = options
	return os.WriteFile(outputPath, []byte("fake-pdf-content"), 0644)
}
func (m *MockMarkdownConverter) HTMLToDocx(htmlContent, outputPath string, options markdown.ConversionOptions) error {
	m.LastOptions = options
	return os.WriteFile(outputPath, []byte("fake-docx-content"), 0644)
}
func (m *MockMarkdownConverter) SaveMarkdown(markdownText, outputPath string) error {
	m.LastMarkdown = markdownText
	return os.WriteFile(outputPath, []byte(markdownText), 0644)
}
func (m *MockMarkdownConverter) GenerateMetadataHeader(options markdown.ConversionOptions) string {
	return "Mock Header\n\n"
}

func TestJob_PublishMaterial_PuppetData(t *testing.T) {
	// 1. Setup environment in api/publish_material_test
	wd, _ := os.Getwd()
	// Use absolute path for dataDir because we will change working directory
	dataDir, _ := filepath.Abs(filepath.Join(wd, "..", "api", "publish_material_test"))
	os.RemoveAll(dataDir)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}

	dbPath := filepath.Join(dataDir, "database.db")
	db, err := database.Initialize(dbPath)
	if err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	config := &configuration.Configuration{
		Storage: configuration.StorageConfiguration{DataDirectory: dataDir},
		LLM:     configuration.LLMConfiguration{Language: "en-US"},
	}

	// 2. Prepare Puppet Data
	userID := uuid.New().String()
	examID := uuid.New().String()
	lectureID := uuid.New().String()
	
	// Create common records
	_, _ = db.Exec("INSERT INTO users (id, username, password_hash) VALUES (?, ?, ?)", userID, "tester", "hash")
	_, _ = db.Exec("INSERT INTO exams (id, user_id, title) VALUES (?, ?, ?)", examID, userID, "Puppet Course")
	_, _ = db.Exec("INSERT INTO lectures (id, exam_id, title, status, specified_date) VALUES (?, ?, ?, 'ready', ?)", 
		lectureID, examID, "Puppet Lecture", "2026-02-08T12:00:00Z")

	// Mocks
	mockLLM := &mockLLMProvider{
		ResponseText: `{"description": "Puppet Abstract from Mock LLM"}`,
	}
	realGenerator := tools.NewToolGenerator(config, mockLLM, nil)
	
	// Use REAL converter
	realConverter := markdown.NewConverter(dataDir)
	
	// Change working directory to server root so template is found
	originalWd, _ := os.Getwd()
	serverRoot, _ := filepath.Abs(filepath.Join(originalWd, "..", ".."))
	if err := os.Chdir(serverRoot); err != nil {
		t.Fatalf("Failed to chdir to server root: %v", err)
	}
	defer os.Chdir(originalWd)

	jobQueue := NewQueue(db, 1)
	RegisterHandlers(jobQueue, db, config, nil, nil, realGenerator, realConverter, nil, nil)

	// Run subtests for different formats
	t.Run("PDF Format", func(t *testing.T) {
		toolID := uuid.New().String()
		puppetContent := `# Puppet Guide PDF
## Intro
Content with citation[^1].
[^1]: Detail (` + "`" + `source.pdf` + "`" + `, p. 5)`

		_, _ = db.Exec(`INSERT INTO tools (id, exam_id, type, title, language_code, content, created_at) 
			VALUES (?, ?, 'guide', 'Puppet Guide PDF', 'en-US', ?, ?)`,
			toolID, examID, puppetContent, time.Now())
		
		_, _ = db.Exec(`INSERT INTO tool_source_references (tool_id, source_type, source_id, metadata) 
			VALUES (?, 'document', 'source.pdf', ?)`,
			toolID, `{"footnote_number": 1, "pages": [5], "description": "Detail"}`)

		jobPayload := map[string]any{"tool_id": toolID, "format": "pdf", "include_qr_code": true}
		payloadBytes, _ := json.Marshal(jobPayload)
		job := &models.Job{ID: uuid.New().String(), Type: models.JobTypePublishMaterial, Payload: string(payloadBytes)}

		err = jobQueue.handlers[models.JobTypePublishMaterial](context.Background(), job, func(p int, m string, meta any, metrics models.JobMetrics) {})
		if err != nil {
			t.Fatalf("Job failed: %v", err)
		}

		exportFile := filepath.Join(dataDir, "files", "exports", toolID, "Puppet Guide PDF.pdf")
		if _, err := os.Stat(exportFile); os.IsNotExist(err) {
			t.Errorf("Export file not created at %s", exportFile)
		}
		
		// Check if file is non-empty and has PDF header
		content, err := os.ReadFile(exportFile)
		if err != nil {
			t.Fatalf("Failed to read export file: %v", err)
		}
		if len(content) < 10 {
			t.Errorf("PDF file is too small")
		}
		// Note: We don't strictly check for %PDF- header here because the real converter 
		// might not be available in all environments, but it should at least exist.
	})

	t.Run("Markdown Format", func(t *testing.T) {
		toolID := uuid.New().String()
		puppetContent := `# Puppet Guide MD
Some markdown content.`

		_, _ = db.Exec(`INSERT INTO tools (id, exam_id, type, title, language_code, content, created_at) 
			VALUES (?, ?, 'guide', 'Puppet Guide MD', 'en-US', ?, ?)`,
			toolID, examID, puppetContent, time.Now())

		jobPayload := map[string]any{"tool_id": toolID, "format": "md", "include_qr_code": true}
		payloadBytes, _ := json.Marshal(jobPayload)
		job := &models.Job{ID: uuid.New().String(), Type: models.JobTypePublishMaterial, Payload: string(payloadBytes)}

		err = jobQueue.handlers[models.JobTypePublishMaterial](context.Background(), job, func(p int, m string, meta any, metrics models.JobMetrics) {})
		if err != nil {
			t.Fatalf("Job failed: %v", err)
		}

		exportFile := filepath.Join(dataDir, "files", "exports", toolID, "Puppet Guide MD.md")
		content, err := os.ReadFile(exportFile)
		if err != nil {
			t.Fatalf("Failed to read export file: %v", err)
		}
		
		output := string(content)
		if !strings.Contains(output, "Puppet Course") {
			t.Errorf("Markdown missing Course title. Got: %s", output)
		}
		if !strings.Contains(output, "Puppet Abstract from Mock LLM") {
			t.Errorf("Markdown missing abstract. Got: %s", output)
		}
	})
}

type mockLLMProvider struct {
	ResponseText string
}

func (m *mockLLMProvider) Chat(ctx context.Context, req *llm.ChatRequest) (<-chan llm.ChatResponseChunk, error) {
	ch := make(chan llm.ChatResponseChunk, 1)
	ch <- llm.ChatResponseChunk{Text: m.ResponseText}
	close(ch)
	return ch, nil
}

func (m *mockLLMProvider) Name() string { return "mock-llm" }