package handler

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strings"
)

func ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// --- Static & Core Routes ---
	if path == "/" || path == "/index.html" {
		serveIndex(w, r)
		return
	}
	if path == "/formbuilder" || path == "/formbuilder/" {
		serveFormBuilder(w, r)
		return
	}
	if strings.HasPrefix(path, "/static/") {
		http.ServeFile(w, r, "."+path)
		return
	}

	// --- API Routes ---
	if path == "/api/forms" && r.Method == http.MethodPost {
		saveFormDefinition(w, r)
		return
	}
	if path == "/api/forms/list" && r.Method == http.MethodGet {
		listForms(w, r)
		return
	}

	// --- Dynamic Form Rendering (VENI) ---
	// Pattern: /forms/{slug}
	if strings.HasPrefix(path, "/forms/") {
		renderDynamicForm(w, r)
		return
	}

	// --- 404 ---
	serve404(w, r)
}

func serveFormBuilder(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("static/formbuilder.html")
	if err != nil {
		http.Error(w, "Builder template missing", 500)
		return
	}
	tmpl.Execute(w, map[string]string{
		"Title": "SONG FormBuilder - Azzurro Tech",
	})
}

func saveFormDefinition(w http.ResponseWriter, r *http.Request) {
	// Expecting JSON: { "slug": "contact", "title": "Contact Us", "fields": [...] }
	var formDef map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&formDef); err != nil {
		http.Error(w, "Invalid JSON", 400)
		return
	}

	// TODO: Insert into PostgreSQL using db.DB
	// query: INSERT INTO form_definitions (slug, title, schema_json) VALUES ($1, $2, $3)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "saved", "slug": formDef["slug"].(string)})
}

func listForms(w http.ResponseWriter, r *http.Request) {
	// TODO: SELECT slug, title FROM form_definitions
	// Mock response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]map[string]string{
		{"slug": "contact", "title": "Contact Us"},
		{"slug": "feedback", "title": "Feedback Form"},
	})
}

func renderDynamicForm(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, "/forms/")

	// TODO: Fetch schema from DB based on slug
	// Mock Schema
	schema := map[string]interface{}{
		"title": "Dynamic Form: " + slug,
		"fields": []map[string]string{
			{"name": "email", "type": "email", "label": "Email Address"},
			{"name": "message", "type": "textarea", "label": "Message"},
		},
	}

	// Render HTML dynamically
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(generateFormHTML(schema)))
}

func generateFormHTML(schema map[string]interface{}) string {
	title := schema["title"].(string)
	fields := schema["fields"].([]map[string]string)

	html := `<!DOCTYPE html><html><head><title>` + title + `</title>
	<link rel="stylesheet" href="/static/style.css"></head><body>
	<div class="container"><h1>` + title + `</h1><form action="/api/submit/` + strings.Split(title, ": ")[1] + `" method="POST">`

	for _, f := range fields {
		label := f["label"]
		name := f["name"]
		fType := f["type"]

		if fType == "textarea" {
			html += `<label>` + label + `</label><textarea name="` + name + `"></textarea>`
		} else {
			html += `<label>` + label + `</label><input type="` + fType + `" name="` + name + `">`
		}
		html += `<br>`
	}

	html += `<button type="submit">Submit</button></form></div></body></html>`
	return html
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	tmpl, _ := template.ParseFiles("static/index.html")
	tmpl.Execute(w, map[string]string{
		"Title":  "SONG - Azzurro Technology",
		"Author": "Azzurro Technology Inc.",
	})
}

func serve404(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	tmpl, _ := template.ParseFiles("static/404.html")
	tmpl.Execute(w, nil)
}
