package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/azzurrotech/song/internal/config"
	"github.com/azzurrotech/song/internal/core"
	"github.com/azzurrotech/song/internal/db"
)

func main() {
	// Load configuration from .env
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate configuration
	if err := config.Validate(cfg); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	// Initialize Core Storage (JSON files)
	coreStore, err := core.NewStore(cfg.CoreDir)
	if err != nil {
		log.Fatalf("Failed to initialize core storage: %v", err)
	}

	// Initialize Database Manager (if enabled)
	var dbManager *db.Manager
	if cfg.DBEnabled {
		dbManager, err = db.NewManager(cfg)
		if err != nil {
			log.Fatalf("Failed to initialize database manager: %v", err)
		}
		defer dbManager.Close()
	}

	// Initialize the server
	server, err := NewServer(cfg, coreStore, dbManager)
	if err != nil {
		log.Fatalf("Failed to initialize server: %v", err)
	}

	// Start the server in a goroutine
	go func() {
		addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
		log.Printf("Starting SONG server on %s", addr)
		if err := server.Start(addr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down SONG server...")
	if err := server.Shutdown(); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("SONG server exited gracefully")
}
