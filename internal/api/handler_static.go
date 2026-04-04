package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/azzurrotech/song/internal/templates"
)

// StaticHandler handles static file serving and template rendering
type StaticHandler struct {
	templateR *templates.Renderer
	staticDir string
	devMode   bool
}

// NewStaticHandler creates a new static handler
func NewStaticHandler(templateR *templates.Renderer, staticDir string, devMode bool) *StaticHandler {
	return &StaticHandler{
		templateR: templateR,
		staticDir: staticDir,
		devMode:   devMode,
	}
}

// IndexHandler serves the landing page
func (sh *StaticHandler) IndexHandler(w http.ResponseWriter, req *http.Request) {
	data := map[string]interface{}{
		"title":       "SONG - Secure Open Network Gateway",
		"version":     "1.0.0",
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"environment": sh.getEnv(),
	}

	sh.renderTemplate(w, "index.html", data)
}

// AdminDashboardHandler serves the admin dashboard
func (sh *StaticHandler) AdminDashboardHandler(w http.ResponseWriter, req *http.Request) {
	data := map[string]interface{}{
		"title":      "Admin Dashboard",
		"page":       "dashboard",
		"user":       sh.getCurrentUser(req),
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"nav_active": "dashboard",
	}

	sh.renderTemplate(w, "admin/dashboard.html", data)
}

// FunctionListHandler serves the function management page
func (sh *StaticHandler) FunctionListHandler(w http.ResponseWriter, req *http.Request) {
	data := map[string]interface{}{
		"title":      "Function Management",
		"page":       "functions",
		"user":       sh.getCurrentUser(req),
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"nav_active": "functions",
	}

	sh.renderTemplate(w, "admin/functions.html", data)
}

// CreateFunctionHandler serves the function creation page
func (sh *StaticHandler) CreateFunctionHandler(w http.ResponseWriter, req *http.Request) {
	data := map[string]interface{}{
		"title":      "Create Function",
		"page":       "create_function",
		"user":       sh.getCurrentUser(req),
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"nav_active": "functions",
		"methods":    []string{"GET", "POST", "PUT", "DELETE"},
	}

	sh.renderTemplate(w, "admin/create_function.html", data)
}

// SettingsHandler serves the system settings page
func (sh *StaticHandler) SettingsHandler(w http.ResponseWriter, req *http.Request) {
	data := map[string]interface{}{
		"title":      "System Settings",
		"page":       "settings",
		"user":       sh.getCurrentUser(req),
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"nav_active": "settings",
	}

	sh.renderTemplate(w, "admin/settings.html", data)
}

// LoginHandler serves the login page
func (sh *StaticHandler) LoginHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method == "POST" {
		// Handle login submission (delegated to auth service)
		// This is just the page rendering
	}

	data := map[string]interface{}{
		"title":     "Login",
		"page":      "login",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"error":     req.URL.Query().Get("error"),
	}

	sh.renderTemplate(w, "auth/login.html", data)
}

// RegisterHandler serves the registration page
func (sh *StaticHandler) RegisterHandler(w http.ResponseWriter, req *http.Request) {
	data := map[string]interface{}{
		"title":     "Register",
		"page":      "register",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"error":     req.URL.Query().Get("error"),
	}

	sh.renderTemplate(w, "auth/register.html", data)
}

// renderTemplate renders a template with the given data
func (sh *StaticHandler) renderTemplate(w http.ResponseWriter, tmplName string, data interface{}) {
	// Set security headers
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")

	// Try to execute template
	err := sh.templateR.Execute(tmplName, w, data)
	if err != nil {
		// In dev mode, try to reload templates
		if sh.devMode {
			if reloadErr := sh.templateR.Reload(); reloadErr == nil {
				err = sh.templateR.Execute(tmplName, w, data)
			}
		}

		if err != nil {
			http.Error(w, "Template rendering error: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// ServeStaticFile serves a static file with proper MIME type detection
func (sh *StaticHandler) ServeStaticFile(w http.ResponseWriter, req *http.Request) {
	// Get the requested path
	path := req.URL.Path

	// Strip the /static/ prefix if present
	if strings.HasPrefix(path, "/static/") {
		path = strings.TrimPrefix(path, "/static/")
	}

	// Security: Prevent directory traversal
	if strings.Contains(path, "..") || strings.HasPrefix(path, "/") {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Clean the path
	cleanPath := filepath.Clean(path)

	// Double-check for traversal after cleaning
	if strings.Contains(cleanPath, "..") {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Construct full file path
	fullPath := filepath.Join(sh.staticDir, cleanPath)

	// Verify the resolved path is within staticDir
	resolvedPath, err := filepath.Abs(fullPath)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	staticDirAbs, err := filepath.Abs(sh.staticDir)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if !strings.HasPrefix(resolvedPath, staticDirAbs+string(filepath.Separator)) && resolvedPath != staticDirAbs {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Check if file exists
	info, err := filepath.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "File not found", http.StatusNotFound)
		} else {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	// Prevent directory listing
	if info.IsDir() {
		// Try to serve index.html if it exists
		indexPath := filepath.Join(fullPath, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			fullPath = indexPath
		} else {
			http.Error(w, "Directory listing not allowed", http.StatusForbidden)
			return
		}
	}

	// Serve the file with proper MIME type
	http.ServeFile(w, req, fullPath)
}

// getEnv returns the current environment
func (sh *StaticHandler) getEnv() string {
	env := os.Getenv("SONG_ENV")
	if env == "" {
		return "development"
	}
	return env
}

// getCurrentUser extracts the current user from the request
func (sh *StaticHandler) getCurrentUser(req *http.Request) map[string]interface{} {
	// This would be populated by auth middleware in production
	// For now, return empty or basic info
	user := make(map[string]interface{})

	if userVal := req.Context().Value("user"); userVal != nil {
		if u, ok := userVal.(map[string]interface{}); ok {
			return u
		}
	}

	return user
}

// staticFileMiddleware adds caching headers for static files
func (sh *StaticHandler) staticFileMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Add cache control headers for static files
		if strings.HasPrefix(req.URL.Path, "/static/") {
			// Cache for 1 hour in dev, 1 day in production
			maxAge := 3600
			if !sh.devMode {
				maxAge = 86400
			}

			w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAge))
			w.Header().Set("ETag", fmt.Sprintf("\"%d\"", time.Now().Unix()))
		}

		next.ServeHTTP(w, req)
	})
}
