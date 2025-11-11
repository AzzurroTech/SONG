package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path"
	"strings"
	"sync"
)

// ---------- In‑memory data structures ----------
type Component struct {
	ID   string `json:"id"`   // generated UUID‑like string
	HTML string `json:"html"` // inner HTML of the custom element
	JS   string `json:"js"`   // optional script for the element
}

type Page struct {
	Slug        string   `json:"slug"`        // URL part, e.g. "home"
	Title       string   `json:"title"`       // shown in <title>
	Components  []string `json:"components"`  // slice of Component.ID
	Description string   `json:"description"` // optional meta description
}

// Simple thread‑safe stores
var (
	compStore = struct {
		sync.RWMutex
		m map[string]Component
	}{m: make(map[string]Component)}

	pageStore = struct {
		sync.RWMutex
		m map[string]Page
	}{m: make(map[string]Page)}

	jsStore = struct {
		sync.RWMutex
		m map[string]string // key → raw JS source
	}{m: make(map[string]string)}
)

// ---------- Basic auth ----------
const (
	authUser = "admin"
	authPass = "changeme" // <-- replace with a strong password
)

func basicAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != authUser || pass != authPass {
			w.Header().Set("WWW-Authenticate", `Basic realm="SONG"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}
}

// ---------- Helper utilities ----------
func jsonResponse(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// Very small UUID‑like generator (no external lib)
func genID() string {
	b := make([]byte, 9)
	_, _ = base64.URLEncoding.Encode(b, b) // ignore error – deterministic length
	return base64.RawURLEncoding.EncodeToString(b)[:12]
}

// ---------- API Handlers ----------
func createComponent(w http.ResponseWriter, r *http.Request) {
	var c Component
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	c.ID = genID()
	compStore.Lock()
	compStore.m[c.ID] = c
	compStore.Unlock()
	jsonResponse(w, c)
}

func listComponents(w http.ResponseWriter, r *http.Request) {
	compStore.RLock()
	defer compStore.RUnlock()
	list := make([]Component, 0, len(compStore.m))
	for _, c := range compStore.m {
		list = append(list, c)
	}
	jsonResponse(w, list)
}

func createPage(w http.ResponseWriter, r *http.Request) {
	var p Page
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if p.Slug == "" {
		http.Error(w, "slug required", http.StatusBadRequest)
		return
	}
	pageStore.Lock()
	pageStore.m[p.Slug] = p
	pageStore.Unlock()
	jsonResponse(w, p)
}

func updatePage(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, "/api/pages/")
	pageStore.Lock()
	defer pageStore.Unlock()
	p, ok := pageStore.m[slug]
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	p.Slug = slug // enforce unchanged slug
	pageStore.m[slug] = p
	jsonResponse(w, p)
}

func getPage(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, "/pages/")
	pageStore.RLock()
	p, ok := pageStore.m[slug]
	pageStore.RUnlock()
	if !ok {
		http.NotFound(w, r)
		return
	}
	// Render a very simple HTML page that includes the stored components.
	var sb strings.Builder
	sb.WriteString("<!doctype html><html><head>")
	sb.WriteString(fmt.Sprintf("<title>%s</title>", p.Title))
	if p.Description != "" {
		sb.WriteString(fmt.Sprintf(`<meta name="description" content="%s">`, p.Description))
	}
	sb.WriteString(`<style>nav{margin-bottom:1rem;} nav a{margin-right:.5rem;}</style>`)
	sb.WriteString("</head><body>")
	sb.WriteString(`<nav><a href="/">Home</a>`)
	// list all known pages for quick navigation
	pageStore.RLock()
	for _, pg := range pageStore.m {
		if pg.Slug != slug {
			sb.WriteString(fmt.Sprintf(`<a href="/pages/%s">%s</a>`, pg.Slug, pg.Title))
		}
	}
	pageStore.RUnlock()
	sb.WriteString(`</nav>`)

	// Insert each component as a custom element.
	for _, cid := range p.Components {
		compStore.RLock()
		c, exists := compStore.m[cid]
		compStore.RUnlock()
		if !exists {
			continue // silently skip missing components
		}
		// The component is defined as a custom element with tag name "comp-{id}"
		tag := fmt.Sprintf("comp-%s", cid[:8])
		sb.WriteString(fmt.Sprintf("<%s></%s>", tag, tag))
	}
	sb.WriteString("</body></html>")
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(sb.String()))
}

// Store arbitrary JS (display‑only)
func storeJS(w http.ResponseWriter, r *http.Request) {
	type payload struct {
		Key  string `json:"key"`
		Code string `json:"code"`
	}
	var p payload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if p.Key == "" {
		http.Error(w, "key required", http.StatusBadRequest)
		return
	}
	jsStore.Lock()
	jsStore.m[p.Key] = p.Code
	jsStore.Unlock()
	jsonResponse(w, map[string]string{"status": "saved"})
}

// Retrieve stored JS (read‑only)
func getJS(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/api/js/")
	jsStore.RLock()
	code, ok := jsStore.m[key]
	jsStore.RUnlock()
	if !ok {
		http.NotFound(w, r)
		return
	}
	jsonResponse(w, map[string]string{"code": code})
}

// ---------- UI (static) ----------
func uiHandler(w http.ResponseWriter, r *http.Request) {
	// All UI files live under ./static/.  The path is stripped of the "/ui/" prefix.
	upath := strings.TrimPrefix(r.URL.Path, "/ui/")
	if upath == "" {
		upath = "index.html"
	}
	fpath := path.Join("static", upath)
	http.ServeFile(w, r, fpath)
}

// ---------- Main ----------
func main() {
	// ----- API (protected) -----
	api := http.NewServeMux()
	api.HandleFunc("/api/components", basicAuth(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			createComponent(w, r)
		case http.MethodGet:
			listComponents(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	api.HandleFunc("/api/pages", basicAuth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			createPage(w, r)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}))
	api.HandleFunc("/api/pages/", basicAuth(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			updatePage(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	api.HandleFunc("/api/js", basicAuth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			storeJS(w, r)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}))
	api.HandleFunc("/api/js/", basicAuth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			getJS(w, r)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}))

	// ----- Public routes -----
	mux := http.NewServeMux()
	mux.Handle("/", api)               // API under root
	mux.HandleFunc("/ui/", uiHandler)  // UI assets
	mux.HandleFunc("/pages/", getPage) // render assembled pages

	// Serve static assets (CSS/JS) that are not UI‑protected
	fileServer := http.FileServer(http.Dir("./static"))
	mux.Handle("/assets/", http.StripPrefix("/assets/", fileServer))

	addr := ":8080"
	log.Printf("🚀 SONG server listening on %s – basic auth user=%s, pass=%s", addr, authUser, authPass)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
