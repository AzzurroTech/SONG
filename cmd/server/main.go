package main

import (
	"log"
	"net/http"
	"os"

	"github.com/azzurrotech/song/internal/context"
	"github.com/azzurrotech/song/internal/db"
	"github.com/azzurrotech/song/internal/handler"
)

func main() {
	// Initialize Context (VICI)
	context.GlobalManager.SetConfig("app_name", "SONG")
	context.GlobalManager.SetConfig("version", "1.0.0")
	context.GlobalManager.SetConfig("author", "Azzurro Technology Inc.")

	// Initialize Database (VIDI)
	if err := db.InitDB(); err != nil {
		log.Fatalf("Failed to initialize DB: %v", err)
	}

	// Setup Routes
	http.HandleFunc("/", handler.ServeHTTP)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("SONG Server starting on port %s", port)
	log.Printf("Contact: info@azzurro.tech | Website: azzurro.tech")

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
