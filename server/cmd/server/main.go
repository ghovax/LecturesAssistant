package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"lectures/internal/api"
	"lectures/internal/configuration"
	"lectures/internal/database"
	"lectures/internal/documents"
	"lectures/internal/jobs"
	"lectures/internal/llm"
	"lectures/internal/markdown"
	"lectures/internal/prompts"
	"lectures/internal/tools"
	"lectures/internal/transcription"
)

func main() {
	// Parse command-line flags
	configurationPath := flag.String("configuration", "", "Path to configuration file")
	flag.Parse()

	// Load configuration
	loadedConfiguration, err := configuration.Load(*configurationPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Ensure data directory exists
	if err := ensureDataDirectory(loadedConfiguration.Storage.DataDirectory); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Initialize JSON logging to a file
	logFilePath := filepath.Join(loadedConfiguration.Storage.DataDirectory, "server.log")
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()

	// MultiWriter to log to both file and stdout
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	logger := slog.New(slog.NewJSONHandler(multiWriter, nil))
	slog.SetDefault(logger)

	// Initialize database
	databasePath := filepath.Join(loadedConfiguration.Storage.DataDirectory, "database.db")
	initializedDatabase, err := database.Initialize(databasePath)
	if err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer initializedDatabase.Close()

	// Initialize prompt manager
	promptManager := prompts.NewManager("prompts")

	// Initialize LLM provider
	var llmProvider llm.Provider
	var defaultLLMModel string

	switch loadedConfiguration.LLM.Provider {
	case "openrouter":
		llmProvider = llm.NewOpenRouterProvider(loadedConfiguration.LLM.OpenRouter.APIKey)
		defaultLLMModel = loadedConfiguration.LLM.OpenRouter.DefaultModel
	case "ollama":
		llmProvider = llm.NewOllamaProvider(loadedConfiguration.LLM.Ollama.BaseURL)
		defaultLLMModel = loadedConfiguration.LLM.Ollama.DefaultModel
	default:
		slog.Warn("Unknown LLM provider, falling back to openrouter with empty key", "provider", loadedConfiguration.LLM.Provider)
		llmProvider = llm.NewOpenRouterProvider("")
		defaultLLMModel = loadedConfiguration.LLM.OpenRouter.DefaultModel
	}

	// Initialize transcription provider and service
	var transcriptionProvider transcription.Provider
	switch loadedConfiguration.Transcription.Provider {
	case "whisper-local":
		transcriptionProvider = transcription.NewWhisperProvider(
			loadedConfiguration.Transcription.Whisper.Model,
			loadedConfiguration.Transcription.Whisper.Device,
		)
	default:
		slog.Warn("Unknown transcription provider, falling back to whisper-local", "provider", loadedConfiguration.Transcription.Provider)
		transcriptionProvider = transcription.NewWhisperProvider("base", "auto")
	}

	transcriptionService := transcription.NewService(loadedConfiguration, transcriptionProvider, llmProvider, promptManager)

	// Initialize document processor
	documentProcessor := documents.NewProcessor(llmProvider, defaultLLMModel, promptManager)

	// Initialize markdown converter
	markdownConverter := markdown.NewConverter(loadedConfiguration.Storage.DataDirectory)

	// Check dependencies
	if err := transcriptionService.CheckDependencies(); err != nil {
		slog.Error("Transcription dependencies check failed", "error", err)
		os.Exit(1)
	}
	if err := documentProcessor.CheckDependencies(); err != nil {
		slog.Error("Document processor dependencies check failed", "error", err)
		os.Exit(1)
	}
	if err := markdownConverter.CheckDependencies(); err != nil {
		slog.Warn("Markdown converter dependencies check failed (PDF export may not work)", "error", err)
	}

	// Initialize tool generator
	toolGenerator := tools.NewToolGenerator(loadedConfiguration, llmProvider, promptManager)

	// Initialize job queue
	backgroundJobQueue := jobs.NewQueue(initializedDatabase, 4) // 4 concurrent workers

	// Register job handlers
	jobs.RegisterHandlers(
		backgroundJobQueue,
		initializedDatabase,
		loadedConfiguration,
		transcriptionService,
		documentProcessor,
		toolGenerator,
		markdownConverter,
		database.CheckLectureReadiness,
	)

	backgroundJobQueue.Start()
	defer backgroundJobQueue.Stop()

	// Create API server
	apiServer := api.NewServer(loadedConfiguration, initializedDatabase, backgroundJobQueue, llmProvider, promptManager)

	// Start HTTP server
	serverAddress := fmt.Sprintf("%s:%d", loadedConfiguration.Server.Host, loadedConfiguration.Server.Port)
	slog.Info("Server starting", "address", serverAddress)
	slog.Info("Data directory", "directory", loadedConfiguration.Storage.DataDirectory)

	if err := http.ListenAndServe(serverAddress, apiServer.Handler()); err != nil {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
}

func ensureDataDirectory(directoryPath string) error {
	// Expand home directory
	if len(directoryPath) > 0 && directoryPath[0] == '~' {
		homeDirectory, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		directoryPath = filepath.Join(homeDirectory, directoryPath[1:])
	}

	// Create necessary subdirectories
	targetDirectories := []string{
		directoryPath,
		filepath.Join(directoryPath, "files", "lectures"),
		filepath.Join(directoryPath, "files", "exports"),
		filepath.Join(directoryPath, "models"),
	}

	for _, directory := range targetDirectories {
		if err := os.MkdirAll(directory, 0755); err != nil {
			return err
		}
	}

	return nil
}
