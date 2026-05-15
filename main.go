package song

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

type Config struct {
	Port        int
	PublicDir   string
	HandlersDir string
}

func DefaultConfig() Config {
	port := 8080
	if p := os.Getenv("SONG_PORT"); p != "" {
		fmt.Sscanf(p, "%d", &port)
	}
	return Config{
		Port:        port,
		PublicDir:   "./public",
		HandlersDir: "./handlers/post",
	}
}

func Run(cfg Config) error {
	if err := InitSecrets(); err != nil {
		return fmt.Errorf("failed to initialize secrets: %w", err)
	}

	srv := NewServer(cfg)

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: srv,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("SONG Server starting on port %d...", cfg.Port)
		log.Printf("Static files: %s", cfg.PublicDir)
		log.Printf("Handlers: %s", cfg.HandlersDir)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	<-quit
	log.Println("Shutting down...")
	return nil
}
