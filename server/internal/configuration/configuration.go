package configuration

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Configuration struct {
	Server            ServerConfiguration        `yaml:"server" json:"server"`
	Storage           StorageConfiguration       `yaml:"storage" json:"storage"`
	Security          SecurityConfiguration      `yaml:"security" json:"security"`
	LLM               LLMConfiguration           `yaml:"llm" json:"llm"`
	Transcription     TranscriptionConfiguration `yaml:"transcription" json:"transcription"`
	Providers         ProvidersConfiguration     `yaml:"providers" json:"providers"`
	Documents         DocumentsConfiguration     `yaml:"documents" json:"documents"`
	Uploads           UploadsConfiguration       `yaml:"uploads" json:"uploads"`
	Safety            SafetyConfiguration        `yaml:"safety" json:"safety"`
	ConfigurationPath string                     `yaml:"-" json:"-"`
}

type SafetyConfiguration struct {
	MaximumCostPerJob    float64 `yaml:"maximum_cost_per_job" json:"maximum_cost_per_job"`
	MaximumLoginAttempts int     `yaml:"maximum_login_attempts_per_hour" json:"maximum_login_attempts_per_hour"`
	MaximumRetries       int     `yaml:"maximum_retries" json:"maximum_retries"`
}

type ServerConfiguration struct {
	Host string `yaml:"host" json:"host"`
	Port int    `yaml:"port" json:"port"`
}

type StorageConfiguration struct {
	DataDirectory string `yaml:"data_directory" json:"data_directory"`
	BinDirectory  string `yaml:"bin_directory,omitempty" json:"bin_directory,omitempty"`
	WebDirectory  string `yaml:"web_directory,omitempty" json:"web_directory,omitempty"`
}

type SecurityConfiguration struct {
	Auth AuthConfiguration `yaml:"auth" json:"auth"`
}

type AuthConfiguration struct {
	Type                string `yaml:"type" json:"type"`
	SessionTimeoutHours int    `yaml:"session_timeout_hours" json:"session_timeout_hours"`
	PasswordHash        string `yaml:"password_hash" json:"-"`
	RequireHTTPS        bool   `yaml:"require_https" json:"require_https"`
}

type LLMConfiguration struct {
	Provider                string              `yaml:"provider" json:"provider"`
	Language                string              `yaml:"language" json:"language"`
	EnableDocumentsMatching bool                `yaml:"enable_documents_matching" json:"enable_documents_matching"`
	Models                  ModelsConfiguration `yaml:"models" json:"models"`

	// Backwards compatibility (deprecated)
	Model        string `yaml:"model,omitempty" json:"model,omitempty"`
	DefaultModel string `yaml:"default_model,omitempty" json:"default_model,omitempty"`
}

type ModelConfiguration struct {
	Model    string `yaml:"model" json:"model"`
	Provider string `yaml:"provider,omitempty" json:"provider,omitempty"`
}

func (modelConfig *ModelConfiguration) UnmarshalYAML(value *yaml.Node) error {
	// Try string first
	var modelName string
	if err := value.Decode(&modelName); err == nil {
		modelConfig.Model = modelName
		return nil
	}

	// Try struct
	type alias ModelConfiguration
	var structuredModel alias
	if err := value.Decode(&structuredModel); err == nil {
		*modelConfig = ModelConfiguration(structuredModel)
		return nil
	}

	return nil
}

func (modelConfig ModelConfiguration) String() string {
	if modelConfig.Provider != "" {
		return modelConfig.Provider + ":" + modelConfig.Model
	}
	return modelConfig.Model
}

type ModelsConfiguration struct {
	// New naming convention
	RecordingTranscription ModelConfiguration `yaml:"recording_transcription,omitempty" json:"recording_transcription,omitempty"`
	DocumentsIngestion     ModelConfiguration `yaml:"documents_ingestion,omitempty" json:"documents_ingestion,omitempty"`
	DocumentsMatching      ModelConfiguration `yaml:"documents_matching,omitempty" json:"documents_matching,omitempty"`
	OutlineCreation        ModelConfiguration `yaml:"outline_creation,omitempty" json:"outline_creation,omitempty"`
	ContentGeneration      ModelConfiguration `yaml:"content_generation,omitempty" json:"content_generation,omitempty"`
	ContentVerification    ModelConfiguration `yaml:"content_verification,omitempty" json:"content_verification,omitempty"`
	ContentPolishing       ModelConfiguration `yaml:"content_polishing,omitempty" json:"content_polishing,omitempty"`

	// Backwards compatibility (deprecated)
	Ingestion     ModelConfiguration `yaml:"ingestion,omitempty" json:"ingestion,omitempty"`
	Triangulation ModelConfiguration `yaml:"triangulation,omitempty" json:"triangulation,omitempty"`
	Structure     ModelConfiguration `yaml:"structure,omitempty" json:"structure,omitempty"`
	Generation    ModelConfiguration `yaml:"generation,omitempty" json:"generation,omitempty"`
	Adherence     ModelConfiguration `yaml:"adherence,omitempty" json:"adherence,omitempty"`
	Polishing     ModelConfiguration `yaml:"polishing,omitempty" json:"polishing,omitempty"`
}

// GetModelForTask returns the model to use for a specific task
func (llmConfig *LLMConfiguration) GetModelForTask(task string) string {
	var modelConfig ModelConfiguration

	// Try new naming convention first
	switch task {
	case "recording_transcription":
		modelConfig = llmConfig.Models.RecordingTranscription
	case "documents_ingestion":
		modelConfig = llmConfig.Models.DocumentsIngestion
	case "documents_matching":
		modelConfig = llmConfig.Models.DocumentsMatching
	case "outline_creation":
		modelConfig = llmConfig.Models.OutlineCreation
	case "content_generation":
		modelConfig = llmConfig.Models.ContentGeneration
	case "content_verification":
		modelConfig = llmConfig.Models.ContentVerification
	case "content_polishing":
		modelConfig = llmConfig.Models.ContentPolishing
	}

	// Fallback to old naming (backwards compatibility)
	if modelConfig.Model == "" {
		switch task {
		case "recording_transcription":
			// No old equivalent, this is new
		case "documents_ingestion":
			modelConfig = llmConfig.Models.Ingestion
		case "documents_matching":
			modelConfig = llmConfig.Models.Triangulation
		case "outline_creation":
			modelConfig = llmConfig.Models.Structure
		case "content_generation":
			modelConfig = llmConfig.Models.Generation
		case "content_verification":
			modelConfig = llmConfig.Models.Adherence
		case "content_polishing":
			modelConfig = llmConfig.Models.Polishing
		}
	}

	if modelConfig.Model != "" {
		return modelConfig.String()
	}

	// Final fallback to deprecated fields (for old configs)
	if llmConfig.DefaultModel != "" {
		return llmConfig.DefaultModel
	}
	if llmConfig.Model != "" {
		return llmConfig.Model
	}

	return ""
}

type TranscriptionConfiguration struct {
	Provider                string `yaml:"provider" json:"provider"`
	Model                   string `yaml:"model,omitempty" json:"model,omitempty"` // Optional: defaults to llm.models.recording_transcription
	AudioChunkLengthSeconds int    `yaml:"audio_chunk_length_seconds" json:"audio_chunk_length_seconds"`
	RefiningBatchSize       int    `yaml:"refining_batch_size" json:"refining_batch_size"`
}

// GetModel returns the model to use for transcription
// Falls back to LLM configuration if not explicitly set
func (transcriptionConfig *TranscriptionConfiguration) GetModel(llmConfig *LLMConfiguration) string {
	if transcriptionConfig.Model != "" {
		return transcriptionConfig.Model
	}
	// Use recording_transcription model from LLM config
	return llmConfig.GetModelForTask("recording_transcription")
}

type ProvidersConfiguration struct {
	OpenRouter OpenRouterConfiguration `yaml:"openrouter" json:"openrouter"`
	Ollama     OllamaConfiguration     `yaml:"ollama" json:"ollama"`
	Google     GoogleConfiguration     `yaml:"google" json:"google"`
}

type OpenRouterConfiguration struct {
	APIKey string `yaml:"api_key" json:"api_key"`
}

type OllamaConfiguration struct {
	BaseURL string `yaml:"base_url" json:"base_url"`
}

type GoogleConfiguration struct {
	ClientID     string `yaml:"client_id" json:"client_id"`
	ClientSecret string `yaml:"client_secret" json:"client_secret"`
}

type DocumentsConfiguration struct {
	RenderDPI        int      `yaml:"render_dots_per_inch" json:"render_dots_per_inch"`
	MaximumPages     int      `yaml:"maximum_pages" json:"maximum_pages"`
	SupportedFormats []string `yaml:"supported_formats" json:"supported_formats"`
}

type UploadsConfiguration struct {
	Media     MediaUploadConfiguration    `yaml:"media" json:"media"`
	Documents DocumentUploadConfiguration `yaml:"documents" json:"documents"`
}

type MediaUploadConfiguration struct {
	MaximumFileSizeMB        int          `yaml:"maximum_file_size_megabytes" json:"maximum_file_size_megabytes"`
	MaximumFilesPerLecture   int          `yaml:"maximum_files_per_lecture" json:"maximum_files_per_lecture"`
	SupportedFormats         MediaFormats `yaml:"supported_formats" json:"supported_formats"`
	ChunkedUploadThresholdMB int          `yaml:"chunked_upload_threshold_megabytes" json:"chunked_upload_threshold_megabytes"`
}

type MediaFormats struct {
	Video []string `yaml:"video" json:"video"`
	Audio []string `yaml:"audio" json:"audio"`
}

type DocumentUploadConfiguration struct {
	MaximumFileSizeMB       int      `yaml:"maximum_file_size_megabytes" json:"maximum_file_size_megabytes"`
	MaximumFilesPerLecture  int      `yaml:"maximum_files_per_lecture" json:"maximum_files_per_lecture"`
	MaximumPagesPerDocument int      `yaml:"maximum_pages_per_document" json:"maximum_pages_per_document"`
	SupportedFormats        []string `yaml:"supported_formats" json:"supported_formats"`
}

// Load reads the configuration from a file or creates a default one
func Load(configurationPath string) (*Configuration, error) {
	useLocalDefaults := false

	if configurationPath == "" {
		// 1. Check for existing local config
		if _, err := os.Stat("configuration.yaml"); err == nil {
			configurationPath = "configuration.yaml"
			useLocalDefaults = true
		} else {
			// 2. Check if we appear to be in a bundled/portable environment
			// If 'bin' or 'prompts' exist, we should default to local config/data
			_, errBin := os.Stat("bin")
			_, errPrompts := os.Stat("prompts")
			if errBin == nil || errPrompts == nil {
				configurationPath = "configuration.yaml"
				useLocalDefaults = true
			} else {
				// 3. Fallback to standard home directory location
				dataDirectory := os.Getenv("STORAGE_DATA_DIRECTORY")
				if dataDirectory == "" {
					homeDirectory, homeDirError := os.UserHomeDir()
					if homeDirError != nil {
						return nil, homeDirError
					}
					dataDirectory = filepath.Join(homeDirectory, ".lectures")
				}
				configurationPath = filepath.Join(dataDirectory, "configuration.yaml")
			}
		}
	} else if !filepath.IsAbs(configurationPath) && (configurationPath == "configuration.yaml" || strings.HasPrefix(configurationPath, "./")) {
		// If explicitly told to use a local path, assume local defaults for creation
		useLocalDefaults = true
	}

	var loadedConfiguration *Configuration

	// Check if file exists
	if _, statError := os.Stat(configurationPath); os.IsNotExist(statError) {
		// Create default config
		loadedConfiguration = defaultConfiguration(useLocalDefaults)
		loadedConfiguration.ConfigurationPath = configurationPath
		if mkdirError := os.MkdirAll(filepath.Dir(configurationPath), 0755); mkdirError != nil {
			return nil, mkdirError
		}
		if saveError := Save(loadedConfiguration, configurationPath); saveError != nil {
			return nil, saveError
		}
	} else {
		// Read existing config
		configurationData, readingError := os.ReadFile(configurationPath)
		if readingError != nil {
			return nil, readingError
		}

		loadedConfiguration = &Configuration{}
		if unmarshalingError := yaml.Unmarshal(configurationData, loadedConfiguration); unmarshalingError != nil {
			return nil, unmarshalingError
		}
		loadedConfiguration.ConfigurationPath = configurationPath
	}

	loadedConfiguration.Storage.DataDirectory = expandTilde(loadedConfiguration.Storage.DataDirectory)

	// Environment variable overrides (Absolute Priority)
	if envDataDir := os.Getenv("STORAGE_DATA_DIRECTORY"); envDataDir != "" {
		loadedConfiguration.Storage.DataDirectory = expandTilde(envDataDir)
	}
	if envWebDir := os.Getenv("STORAGE_WEB_DIRECTORY"); envWebDir != "" {
		loadedConfiguration.Storage.WebDirectory = expandTilde(envWebDir)
	}

	return loadedConfiguration, nil
}

func expandTilde(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[1:])
		}
	}
	return path
}

// Save writes the configuration to a file
func Save(configuration *Configuration, configurationPath string) error {
	marshaledData, marshalingError := yaml.Marshal(configuration)
	if marshalingError != nil {
		return marshalingError
	}
	return os.WriteFile(configurationPath, marshaledData, 0600)
}

// defaultConfiguration returns a configuration with sensible defaults
func defaultConfiguration(isLocal bool) *Configuration {
	var dataDir string
	if isLocal {
		dataDir = "./data"
	} else if os.Getenv("IN_DOCKER_ENV") == "true" {
		dataDir = "/data"
	} else {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, ".lectures", "data")
	}

	return &Configuration{
		Server: ServerConfiguration{
			Host: "0.0.0.0",
			Port: 3000,
		},
		Storage: StorageConfiguration{
			DataDirectory: dataDir,
		},
		Security: SecurityConfiguration{
			Auth: AuthConfiguration{
				Type:                "session",
				SessionTimeoutHours: 72,
				RequireHTTPS:        false,
			},
		},
		LLM: LLMConfiguration{
			Provider:                "openrouter",
			Model:                   "google/gemini-3-flash-preview",
			Language:                "en-US",
			EnableDocumentsMatching: false,
			Models: ModelsConfiguration{
				RecordingTranscription: ModelConfiguration{Model: "google/gemini-2.5-flash-lite"},
				DocumentsIngestion:     ModelConfiguration{Model: "google/gemini-2.5-flash-lite"},
				DocumentsMatching:      ModelConfiguration{Model: "google/gemini-2.5-flash-lite"},
				OutlineCreation:        ModelConfiguration{Model: "google/gemini-3-flash-preview"},
				ContentGeneration:      ModelConfiguration{Model: "google/gemini-3-flash-preview"},
				ContentVerification:    ModelConfiguration{Model: "google/gemini-3-flash-preview"},
				ContentPolishing:       ModelConfiguration{Model: "google/gemini-2.5-flash-lite"},
			},
		},
		Transcription: TranscriptionConfiguration{
			Provider:                "openrouter",
			Model:                   "",
			AudioChunkLengthSeconds: 300,
			RefiningBatchSize:       3,
		},
		Providers: ProvidersConfiguration{
			OpenRouter: OpenRouterConfiguration{
				APIKey: "",
			},
			Ollama: OllamaConfiguration{
				BaseURL: "http://localhost:11434",
			},
			Google: GoogleConfiguration{
				ClientID:     "",
				ClientSecret: "",
			},
		},
		Documents: DocumentsConfiguration{
			RenderDPI:        200,
			MaximumPages:     1000,
			SupportedFormats: []string{"pdf", "pptx", "docx"},
		},
		Uploads: UploadsConfiguration{
			Media: MediaUploadConfiguration{
				MaximumFileSizeMB:      5120,
				MaximumFilesPerLecture: 10,
				SupportedFormats: MediaFormats{
					Video: []string{"mp4", "mkv", "mov", "webm"},
					Audio: []string{"mp3", "wav", "m4a", "flac"},
				},
				ChunkedUploadThresholdMB: 100,
			},
			Documents: DocumentUploadConfiguration{
				MaximumFileSizeMB:       500,
				MaximumFilesPerLecture:  50,
				MaximumPagesPerDocument: 500,
				SupportedFormats:        []string{"pdf", "pptx", "docx"},
			},
		},
		Safety: SafetyConfiguration{
			MaximumCostPerJob:    15.0,
			MaximumLoginAttempts: 10,
			MaximumRetries:       3,
		},
	}
}
