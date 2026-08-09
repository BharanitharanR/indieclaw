package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"gateway-service/internal/usercontext"
)

func main() {
	// Load configuration from environment
	dataDir := os.Getenv("USER_DATA_DIR")
	if dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("Failed to get home directory: %v", err)
		}
		dataDir = fmt.Sprintf("%s/.adiyan/users", home)
	}

	port := os.Getenv("UCS_PORT")
	if port == "" {
		port = ":8001"
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	log.Printf("[Main] User Context Service starting...")
	log.Printf("[Main] Data directory: %s", dataDir)
	log.Printf("[Main] Port: %s", port)
	log.Printf("[Main] Log level: %s", logLevel)

	// Initialize storage
	storage, err := usercontext.NewStorage(dataDir)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	// Initialize managers
	profileManager := usercontext.NewProfileManager(storage)
	knowledgeManager := usercontext.NewKnowledgeGraphManager(profileManager)

	// Create HTTP server
	server := usercontext.NewServer(profileManager, knowledgeManager, port)

	// Start server in a goroutine
	go func() {
		if err := server.Start(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	log.Printf("[Main] ✅ User Context Service is running on %s", port)

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	log.Printf("[Main] Received signal: %v", sig)

	// Graceful shutdown
	if err := server.Stop(); err != nil {
		log.Printf("[Main] Error during shutdown: %v", err)
	}

	log.Printf("[Main] ✅ User Context Service stopped gracefully")
}
