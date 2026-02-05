package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"lectures/internal/api"
	"lectures/internal/configuration"
	"lectures/internal/database"
	"lectures/internal/jobs"
)

func main() {
	// Parse command-line flags
	configPath := flag.String("config", "", "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Ensure data directory exists
	if err := ensureDataDirectory(cfg.Storage.DataDirectory); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Initialize database
	dbPath := filepath.Join(cfg.Storage.DataDirectory, "database.db")
	db, err := database.Initialize(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize job queue
	jobQueue := jobs.NewQueue(db, 4) // 4 concurrent workers
	jobQueue.Start()
	defer jobQueue.Stop()

	// Create API server
	server := api.NewServer(cfg, db, jobQueue)

	// Start HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Server starting on %s", addr)
	log.Printf("Data directory: %s", cfg.Storage.DataDirectory)

	if err := http.ListenAndServe(addr, server.Handler()); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func ensureDataDirectory(path string) error {
	// Expand home directory
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		path = filepath.Join(home, path[1:])
	}

	// Create necessary subdirectories
	dirs := []string{
		path,
		filepath.Join(path, "files", "lectures"),
		filepath.Join(path, "files", "exports"),
		filepath.Join(path, "models"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}
