package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"lectures/internal/api"
	"lectures/internal/configuration"
	"lectures/internal/database"
	"lectures/internal/documents"
	"lectures/internal/jobs"
	"lectures/internal/llm"
	"lectures/internal/markdown"
	"lectures/internal/media"
	"lectures/internal/prompts"
	"lectures/internal/tools"
	"lectures/internal/transcription"
)

func main() {
	// Parse command-line flags
	configurationPath := flag.String("configuration", "", "Path to configuration file")
	flag.Parse()

	// 1. Auto-detect configuration if not provided
	finalConfigPath := *configurationPath
	if finalConfigPath == "" {
		if _, err := os.Stat("configuration.yaml"); err == nil {
			finalConfigPath = "configuration.yaml"
		}
	}

	// Load configuration
	loadedConfiguration, loadingError := configuration.Load(finalConfigPath)
	if loadingError != nil {
		log.Fatalf("Failed to load configuration: %v", loadingError)
	}

	// 2. Auto-detect bundled binaries if not configured
	if loadedConfiguration.Storage.BinDirectory == "" {
		if _, err := os.Stat("bin"); err == nil {
			loadedConfiguration.Storage.BinDirectory = "./bin"
			slog.Info("Auto-detected bundled binaries in ./bin")
		}
	}

	// Ensure data directory exists
	if directoryError := ensureDataDirectory(loadedConfiguration.Storage.DataDirectory); directoryError != nil {
		log.Fatalf("Failed to create data directory: %v", directoryError)
	}

	// Initialize JSON logging to a file
	logFilePath := filepath.Join(loadedConfiguration.Storage.DataDirectory, "server.log")
	logFile, fileError := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if fileError != nil {
		log.Fatalf("Failed to open log file: %v", fileError)
	}
	defer logFile.Close()

	// MultiWriter to log to both file and stdout
	multiWriter := io.MultiWriter(os.Stdout, logFile)

	logger := slog.New(slog.NewJSONHandler(multiWriter, nil))
	slog.SetDefault(logger)

	// Initialize database
	databasePath := filepath.Join(loadedConfiguration.Storage.DataDirectory, "database.db")
	initializedDatabase, databaseError := database.Initialize(databasePath)
	if databaseError != nil {
		slog.Error("Failed to initialize database", "error", databaseError)
		os.Exit(1)
	}
	defer initializedDatabase.Close()

	// Initialize prompt manager
	promptManager := prompts.NewManager("prompts")

	// Initialize LLM providers
	openRouterProvider := llm.NewOpenRouterProvider(loadedConfiguration.Providers.OpenRouter.APIKey)
	ollamaProvider := llm.NewOllamaProvider(loadedConfiguration.Providers.Ollama.BaseURL)

	var defaultProvider llm.Provider
	switch loadedConfiguration.LLM.Provider {
	case "openrouter":
		defaultProvider = openRouterProvider
	case "ollama":
		defaultProvider = ollamaProvider
	default:
		slog.Warn("Unknown LLM provider, falling back to openrouter with empty key", "provider", loadedConfiguration.LLM.Provider)
		defaultProvider = openRouterProvider
	}

	routingProvider := llm.NewRoutingProvider(defaultProvider)
	routingProvider.Register("openrouter", openRouterProvider)
	routingProvider.Register("ollama", ollamaProvider)

	llmProvider := routingProvider

	// Initialize transcription provider and service
	var transcriptionProvider transcription.Provider
	transcriptionModel := loadedConfiguration.Transcription.GetModel(&loadedConfiguration.LLM)

	switch loadedConfiguration.Transcription.Provider {
	case "openrouter":
		// Use the centralized llmProvider (RoutingProvider) which handles multiple providers
		transcriptionProvider = transcription.NewOpenRouterTranscriptionProvider(
			llmProvider,
			transcriptionModel,
		)
	default:
		slog.Warn("Unknown transcription provider or provider not supporting audio, falling back to openrouter", "provider", loadedConfiguration.Transcription.Provider)
		transcriptionProvider = transcription.NewOpenRouterTranscriptionProvider(
			llmProvider,
			transcriptionModel,
		)
	}
	transcriptionService := transcription.NewService(loadedConfiguration, transcriptionProvider, llmProvider, promptManager)

	// Initialize document processor
	ingestionModel := loadedConfiguration.LLM.GetModelForTask("documents_ingestion")
	if ingestionModel == "" {
		slog.Error("No model configured for documents_ingestion")
		os.Exit(1)
	}
	slog.Info("Document processor initialized", "model", ingestionModel)
	documentProcessor := documents.NewProcessor(llmProvider, ingestionModel, promptManager, loadedConfiguration.Documents.RenderDPI, loadedConfiguration.Storage.BinDirectory)

	// Initialize markdown converter
	markdownConverter := markdown.NewConverter(loadedConfiguration.Storage.DataDirectory, loadedConfiguration.Storage.BinDirectory)

	// Check dependencies
	if transcriptionError := transcriptionService.CheckDependencies(); transcriptionError != nil {
		slog.Error("Transcription dependencies check failed", "error", transcriptionError)
		// os.Exit(1) // Don't exit, allow UI to show error or user to fix config
	}
	if processorError := documentProcessor.CheckDependencies(); processorError != nil {
		slog.Error("Document processor dependencies check failed", "error", processorError)
		// os.Exit(1)
	}
	if converterError := markdownConverter.CheckDependencies(); converterError != nil {
		slog.Error("Markdown converter dependencies check failed", "error", converterError)
		// os.Exit(1)
	}
	if mediaError := media.CheckDependencies(loadedConfiguration.Storage.BinDirectory); mediaError != nil {
		slog.Error("Media dependencies check failed", "error", mediaError)
		// os.Exit(1)
	}

	// Initialize tool generator
	toolGenerator := tools.NewToolGenerator(loadedConfiguration, llmProvider, promptManager)

	// Initialize job queue
	backgroundJobQueue := jobs.NewQueue(initializedDatabase, 4) // 4 concurrent workers

	// Create API server
	apiServer := api.NewServer(loadedConfiguration, initializedDatabase, backgroundJobQueue, llmProvider, promptManager, toolGenerator, markdownConverter)

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
		func(channel string, msgType string, payload any) {
			apiServer.Broadcast(channel, msgType, payload)
		},
	)

	backgroundJobQueue.Start()

	// Start HTTP server
	serverAddress := fmt.Sprintf("%s:%d", loadedConfiguration.Server.Host, loadedConfiguration.Server.Port)
	slog.Info("Server starting", "address", serverAddress)
	slog.Info("Data directory", "directory", loadedConfiguration.Storage.DataDirectory)

	// Auto-open browser (skip if in Docker)
	if os.Getenv("IN_DOCKER_ENV") != "true" {
		go func() {
			// Wait a moment for the server to actually start listening
			time.Sleep(2 * time.Second)

			url := fmt.Sprintf("http://localhost:%d", loadedConfiguration.Server.Port)
			if loadedConfiguration.Server.Host != "0.0.0.0" && loadedConfiguration.Server.Host != "" {
				url = fmt.Sprintf("http://%s:%d", loadedConfiguration.Server.Host, loadedConfiguration.Server.Port)
			}

			var err error
			switch runtime.GOOS {
			case "linux":
				err = exec.Command("xdg-open", url).Start()
			case "windows":
				// cmd /c start is more robust for opening URLs on Windows
				err = exec.Command("cmd", "/c", "start", url).Start()
			case "darwin":
				err = exec.Command("open", url).Start()
			}
			if err != nil {
				slog.Warn("Failed to open browser automatically", "error", err)
			}
		}()
	}

	if serverError := http.ListenAndServe(serverAddress, apiServer.Handler()); serverError != nil {
		slog.Error("Server failed", "error", serverError)
		os.Exit(1)
	}
}

func ensureDataDirectory(directoryPath string) error {
	// Expand home directory
	if len(directoryPath) > 0 && directoryPath[0] == '~' {
		homeDirectory, homeDirError := os.UserHomeDir()
		if homeDirError != nil {
			return homeDirError
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
		if mkdirError := os.MkdirAll(directory, 0755); mkdirError != nil {
			return mkdirError
		}
	}

	return nil
}
