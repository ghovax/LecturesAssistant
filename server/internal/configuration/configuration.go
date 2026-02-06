package configuration

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Configuration struct {
	Server        ServerConfiguration        `yaml:"server"`
	Storage       StorageConfiguration       `yaml:"storage"`
	Security      SecurityConfiguration      `yaml:"security"`
	LLM           LLMConfiguration           `yaml:"llm"`
	Transcription TranscriptionConfiguration `yaml:"transcription"`
	Providers     ProvidersConfiguration     `yaml:"providers"`
	Documents     DocumentsConfiguration     `yaml:"documents"`
	Uploads       UploadsConfiguration       `yaml:"uploads"`
	Safety        SafetyConfiguration        `yaml:"safety"`
}

type SafetyConfiguration struct {
	MaximumCostPerJob    float64 `yaml:"maximum_cost_per_job"`
	MaximumLoginAttempts int     `yaml:"maximum_login_attempts_per_hour"`
	MaximumRetries       int     `yaml:"maximum_retries"`
}

type ServerConfiguration struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type StorageConfiguration struct {
	DataDirectory string `yaml:"data_directory"`
}

type SecurityConfiguration struct {
	Auth AuthConfiguration `yaml:"auth"`
}

type AuthConfiguration struct {
	Type                string `yaml:"type"`
	SessionTimeoutHours int    `yaml:"session_timeout_hours"`
	PasswordHash        string `yaml:"password_hash"`
	RequireHTTPS        bool   `yaml:"require_https"`
}

type LLMConfiguration struct {
	Provider string              `yaml:"provider"`
	Model    string              `yaml:"model"` // Global fallback
	Language string              `yaml:"language"`
	Models   ModelsConfiguration `yaml:"models"`
}

type ModelsConfiguration struct {
	Ingestion     string `yaml:"ingestion"`
	Triangulation string `yaml:"triangulation"`
	Structure     string `yaml:"structure"`
	Generation    string `yaml:"generation"`
	Adherence     string `yaml:"adherence"`
	Polishing     string `yaml:"polishing"`
}

type TranscriptionConfiguration struct {
	Provider                string `yaml:"provider"`
	Model                   string `yaml:"model"`
	AudioChunkLengthSeconds int    `yaml:"audio_chunk_length_seconds"`
	RefiningBatchSize       int    `yaml:"refining_batch_size"`
	WhisperDevice           string `yaml:"whisper_device"`
}

type ProvidersConfiguration struct {
	OpenRouter OpenRouterConfig `yaml:"openrouter"`
	OpenAI     OpenAIConfig     `yaml:"openai"`
	Ollama     OllamaConfig     `yaml:"ollama"`
}

type OpenRouterConfig struct {
	APIKey string `yaml:"api_key"`
}

type OpenAIConfig struct {
	APIKey string `yaml:"api_key"`
}

type OllamaConfig struct {
	BaseURL string `yaml:"base_url"`
}

type DocumentsConfiguration struct {
	RenderDPI        int      `yaml:"render_dots_per_inch"`
	MaximumPages     int      `yaml:"maximum_pages"`
	SupportedFormats []string `yaml:"supported_formats"`
}

type UploadsConfiguration struct {
	Media     MediaUploadConfiguration    `yaml:"media"`
	Documents DocumentUploadConfiguration `yaml:"documents"`
}

type MediaUploadConfiguration struct {
	MaximumFileSizeMB        int          `yaml:"maximum_file_size_megabytes"`
	MaximumFilesPerLecture   int          `yaml:"maximum_files_per_lecture"`
	SupportedFormats         MediaFormats `yaml:"supported_formats"`
	ChunkedUploadThresholdMB int          `yaml:"chunked_upload_threshold_megabytes"`
}

type MediaFormats struct {
	Video []string `yaml:"video"`
	Audio []string `yaml:"audio"`
}

type DocumentUploadConfiguration struct {
	MaximumFileSizeMB       int      `yaml:"maximum_file_size_megabytes"`
	MaximumFilesPerLecture  int      `yaml:"maximum_files_per_lecture"`
	MaximumPagesPerDocument int      `yaml:"maximum_pages_per_document"`
	SupportedFormats        []string `yaml:"supported_formats"`
}

// Load reads the configuration from a file or creates a default one
func Load(path string) (*Configuration, error) {
	if path == "" {
		dataDir := os.Getenv("STORAGE_DATA_DIRECTORY")
		if dataDir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, err
			}
			dataDir = filepath.Join(home, ".lectures")
		}
		path = filepath.Join(dataDir, "configuration.yaml")
	}

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Create default config
		configuration := defaultConfiguration()
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return nil, err
		}
		if err := Save(configuration, path); err != nil {
			return nil, err
		}
		return configuration, nil
	}

	// Read existing config
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	configuration := &Configuration{}
	if err := yaml.Unmarshal(data, configuration); err != nil {
		return nil, err
	}

	return configuration, nil
}

// Save writes the configuration to a file
func Save(configuration *Configuration, path string) error {
	data, err := yaml.Marshal(configuration)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// defaultConfiguration returns a configuration with sensible defaults
func defaultConfiguration() *Configuration {
	home, _ := os.UserHomeDir()
	return &Configuration{
		Server: ServerConfiguration{
			Host: "127.0.0.1",
			Port: 3000,
		},
		Storage: StorageConfiguration{
			DataDirectory: filepath.Join(home, ".lectures"),
		},
		Security: SecurityConfiguration{
			Auth: AuthConfiguration{
				Type:                "session",
				SessionTimeoutHours: 24,
				RequireHTTPS:        false,
			},
		},
		LLM: LLMConfiguration{
			Provider: "openrouter",
			Model:    "anthropic/claude-3.5-sonnet",
			Language: "en-US",
		},
		Transcription: TranscriptionConfiguration{
			Provider:                "whisper-local",
			Model:                   "base",
			AudioChunkLengthSeconds: 300,
			RefiningBatchSize:       3,
			WhisperDevice:           "auto",
		},
		Providers: ProvidersConfiguration{
			OpenRouter: OpenRouterConfig{
				APIKey: "",
			},
			Ollama: OllamaConfig{
				BaseURL: "http://localhost:11434",
			},
		},
		Documents: DocumentsConfiguration{
			RenderDPI:        150,
			MaximumPages:     500,
			SupportedFormats: []string{"pdf", "pptx", "docx"},
		},
		Uploads: UploadsConfiguration{
			Media: MediaUploadConfiguration{
				MaximumFileSizeMB:      2048,
				MaximumFilesPerLecture: 50,
				SupportedFormats: MediaFormats{
					Video: []string{"mp4", "webm", "mov", "mkv"},
					Audio: []string{"mp3", "wav", "m4a", "ogg", "flac"},
				},
				ChunkedUploadThresholdMB: 100,
			},
			Documents: DocumentUploadConfiguration{
				MaximumFileSizeMB:       500,
				MaximumFilesPerLecture:  100,
				MaximumPagesPerDocument: 500,
				SupportedFormats:        []string{"pdf", "pptx", "docx"},
			},
		},
		Safety: SafetyConfiguration{
			MaximumCostPerJob:    10.0, // $10 safety threshold
			MaximumLoginAttempts: 100,  // High limit as requested
			MaximumRetries:       3,
		},
	}
}
