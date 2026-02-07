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
	Provider                string              `yaml:"provider"`
	Language                string              `yaml:"language"`
	EnableDocumentsMatching bool                `yaml:"enable_documents_matching"`
	Models                  ModelsConfiguration `yaml:"models"`

	// Backwards compatibility (deprecated)
	Model        string `yaml:"model,omitempty"`
	DefaultModel string `yaml:"default_model,omitempty"`
}

type ModelConfiguration struct {
	Model    string `yaml:"model"`
	Provider string `yaml:"provider,omitempty"`
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
	RecordingTranscription ModelConfiguration `yaml:"recording_transcription,omitempty"`
	DocumentsIngestion     ModelConfiguration `yaml:"documents_ingestion,omitempty"`
	DocumentsMatching      ModelConfiguration `yaml:"documents_matching,omitempty"`
	OutlineCreation        ModelConfiguration `yaml:"outline_creation,omitempty"`
	ContentGeneration      ModelConfiguration `yaml:"content_generation,omitempty"`
	ContentVerification    ModelConfiguration `yaml:"content_verification,omitempty"`
	ContentPolishing       ModelConfiguration `yaml:"content_polishing,omitempty"`

	// Backwards compatibility (deprecated)
	Ingestion     ModelConfiguration `yaml:"ingestion,omitempty"`
	Triangulation ModelConfiguration `yaml:"triangulation,omitempty"`
	Structure     ModelConfiguration `yaml:"structure,omitempty"`
	Generation    ModelConfiguration `yaml:"generation,omitempty"`
	Adherence     ModelConfiguration `yaml:"adherence,omitempty"`
	Polishing     ModelConfiguration `yaml:"polishing,omitempty"`
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
	Provider                string `yaml:"provider"`
	Model                   string `yaml:"model,omitempty"` // Optional: defaults to llm.models.recording_transcription
	AudioChunkLengthSeconds int    `yaml:"audio_chunk_length_seconds"`
	RefiningBatchSize       int    `yaml:"refining_batch_size"`
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
	OpenRouter OpenRouterConfig `yaml:"openrouter"`
	Ollama     OllamaConfig     `yaml:"ollama"`
}

type OpenRouterConfig struct {
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
func Load(configurationPath string) (*Configuration, error) {
	if configurationPath == "" {
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

	// Check if file exists
	if _, statError := os.Stat(configurationPath); os.IsNotExist(statError) {
		// Create default config
		newConfiguration := defaultConfiguration()
		if mkdirError := os.MkdirAll(filepath.Dir(configurationPath), 0755); mkdirError != nil {
			return nil, mkdirError
		}
		if saveError := Save(newConfiguration, configurationPath); saveError != nil {
			return nil, saveError
		}
		return newConfiguration, nil
	}

	// Read existing config
	configurationData, readingError := os.ReadFile(configurationPath)
	if readingError != nil {
		return nil, readingError
	}

	loadedConfiguration := &Configuration{}
	if unmarshalingError := yaml.Unmarshal(configurationData, loadedConfiguration); unmarshalingError != nil {
		return nil, unmarshalingError
	}

	return loadedConfiguration, nil
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
			Provider:                "openrouter",
			Model:                   "anthropic/claude-3.5-sonnet",
			Language:                "en-US",
			EnableDocumentsMatching: true,
		},
		Transcription: TranscriptionConfiguration{
			Provider:                "openrouter",
			Model:                   "",
			AudioChunkLengthSeconds: 300,
			RefiningBatchSize:       3,
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
