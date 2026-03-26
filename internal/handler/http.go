package handler

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strings"
)

// ServeHTTP implements the main router logic.
func ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Route: Index
	if path == "/" || path == "/index.html" {
		serveIndex(w, r)
		return
	}

	// Route: 404
	if strings.HasPrefix(path, "/404") {
		serve404(w, r)
		return
	}

	// Route: VIDI - Form Submission (Simulated)
	if path == "/api/submit" && r.Method == http.MethodPost {
		handleFormSubmit(w, r)
		return
	}

	// Route: VENI - Virtual Endpoint (Simulated)
	if strings.HasPrefix(path, "/venie/") {
		handleVenieEndpoint(w, r)
		return
	}

	// Static Files (CSS, JS)
	if strings.HasPrefix(path, "/static/") {
		http.ServeFile(w, r, "."+path)
		return
	}

	// Default 404
	serve404(w, r)
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("static/index.html")
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	data := map[string]string{
		"Title":  "SONG - Azzurro Technology",
		"Author": "Azzurro Technology Inc.",
	}
	tmpl.Execute(w, data)
}

func serve404(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	tmpl, err := template.ParseFiles("static/404.html")
	if err != nil {
		w.Write([]byte("404 Not Found"))
		return
	}
	tmpl.Execute(w, nil)
}

func handleFormSubmit(w http.ResponseWriter, r *http.Request) {
	// VIDI Logic: Parse form data and store in DB
	r.ParseForm()
	formName := r.FormValue("form_name")
	data := r.FormValue("data")

	// In a real scenario, insert into PostgreSQL here using db.DB
	// ctx := r.Context()
	// _, err := db.DB.ExecContext(ctx, "INSERT INTO vidi_records (form_name, data) VALUES ($1, $2)", formName, data)

	response := map[string]string{
		"status":  "success",
		"message": "Data received by VIDI engine",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleVenieEndpoint(w http.ResponseWriter, r *http.Request) {
	// VENI Logic: Serve dynamic content based on path
	// This simulates a virtual endpoint hosting a web component
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte("<h1>VENI Component Loaded</h1><p>This is a dynamic virtual endpoint.</p>"))
}
