package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Configuration struct {
	Server        ServerConfig        `yaml:"server"`
	Storage       StorageConfig       `yaml:"storage"`
	Security      SecurityConfig      `yaml:"security"`
	LLM           LLMConfig           `yaml:"llm"`
	Transcription TranscriptionConfig `yaml:"transcription"`
	Documents     DocumentsConfig     `yaml:"documents"`
	Uploads       UploadsConfig       `yaml:"uploads"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type StorageConfig struct {
	DataDirectory string `yaml:"data_directory"`
}

type SecurityConfig struct {
	Auth AuthConfig `yaml:"auth"`
}

type AuthConfig struct {
	Type                 string `yaml:"type"`
	SessionTimeoutHours  int    `yaml:"session_timeout_hours"`
	PasswordHash         string `yaml:"password_hash"`
	RequireHTTPS         bool   `yaml:"require_https"`
}

type LLMConfig struct {
	Provider   string            `yaml:"provider"`
	OpenRouter OpenRouterConfig  `yaml:"openrouter"`
	Ollama     OllamaConfig      `yaml:"ollama"`
}

type OpenRouterConfig struct {
	APIKey       string `yaml:"api_key"`
	DefaultModel string `yaml:"default_model"`
}

type OllamaConfig struct {
	BaseURL      string `yaml:"base_url"`
	DefaultModel string `yaml:"default_model"`
}

type TranscriptionConfig struct {
	Provider string        `yaml:"provider"`
	Whisper  WhisperConfig `yaml:"whisper"`
	OpenAI   OpenAIConfig  `yaml:"openai"`
}

type WhisperConfig struct {
	Model  string `yaml:"model"`
	Device string `yaml:"device"`
}

type OpenAIConfig struct {
	APIKey string `yaml:"api_key"`
}

type DocumentsConfig struct {
	RenderDPI        int      `yaml:"render_dots_per_inch"`
	MaximumPages     int      `yaml:"maximum_pages"`
	SupportedFormats []string `yaml:"supported_formats"`
}

type UploadsConfig struct {
	Media     MediaUploadConfig    `yaml:"media"`
	Documents DocumentUploadConfig `yaml:"documents"`
}

type MediaUploadConfig struct {
	MaxFileSizeMB              int                `yaml:"maximum_file_size_megabytes"`
	MaxFilesPerLecture         int                `yaml:"maximum_files_per_lecture"`
	SupportedFormats           MediaFormats       `yaml:"supported_formats"`
	ChunkedUploadThresholdMB   int                `yaml:"chunked_upload_threshold_megabytes"`
}

type MediaFormats struct {
	Video []string `yaml:"video"`
	Audio []string `yaml:"audio"`
}

type DocumentUploadConfig struct {
	MaxFileSizeMB       int      `yaml:"maximum_file_size_megabytes"`
	MaxFilesPerLecture  int      `yaml:"maximum_files_per_lecture"`
	MaxPagesPerDocument int      `yaml:"maximum_pages_per_document"`
	SupportedFormats    []string `yaml:"supported_formats"`
}

// Load reads the configuration from a file or creates a default one
func Load(path string) (*Configuration, error) {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(home, ".lectures", "configuration.yaml")
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
		Server: ServerConfig{
			Host: "127.0.0.1",
			Port: 3000,
		},
		Storage: StorageConfig{
			DataDirectory: filepath.Join(home, ".lectures"),
		},
		Security: SecurityConfig{
			Auth: AuthConfig{
				Type:                "session",
				SessionTimeoutHours: 24,
				RequireHTTPS:        false,
			},
		},
		LLM: LLMConfig{
			Provider: "openrouter",
			OpenRouter: OpenRouterConfig{
				DefaultModel: "anthropic/claude-3.5-sonnet",
			},
			Ollama: OllamaConfig{
				BaseURL:      "http://localhost:11434",
				DefaultModel: "llama3.2",
			},
		},
		Transcription: TranscriptionConfig{
			Provider: "whisper-local",
			Whisper: WhisperConfig{
				Model:  "base",
				Device: "auto",
			},
		},
		Documents: DocumentsConfig{
			RenderDPI:        150,
			MaximumPages:     500,
			SupportedFormats: []string{"pdf", "pptx", "docx"},
		},
		Uploads: UploadsConfig{
			Media: MediaUploadConfig{
				MaxFileSizeMB:      2048,
				MaxFilesPerLecture: 50,
				SupportedFormats: MediaFormats{
					Video: []string{"mp4", "webm", "mov", "mkv"},
					Audio: []string{"mp3", "wav", "m4a", "ogg", "flac"},
				},
				ChunkedUploadThresholdMB: 100,
			},
			Documents: DocumentUploadConfig{
				MaxFileSizeMB:       500,
				MaxFilesPerLecture:  100,
				MaxPagesPerDocument: 500,
				SupportedFormats:    []string{"pdf", "pptx", "docx"},
			},
		},
	}
}
