package song

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// StaticServer handles serving static assets
type StaticServer struct {
	root        http.Dir
	cacheMaxAge time.Duration
	indexFiles  []string
}

// StaticConfig holds configuration for static file serving
type StaticConfig struct {
	RootDir     string
	CacheMaxAge time.Duration
	IndexFiles  []string
}

// DefaultStaticConfig returns sensible defaults
func DefaultStaticConfig() StaticConfig {
	return StaticConfig{
		RootDir:     "./public",
		CacheMaxAge: 24 * time.Hour,
		IndexFiles:  []string{"index.html", "index.htm"},
	}
}

// NewStaticServer creates a new static file server
func NewStaticServer(cfg StaticConfig) *StaticServer {
	if _, err := os.Stat(cfg.RootDir); os.IsNotExist(err) {
		log.Printf("⚠️  Static directory does not exist: %s, creating it", cfg.RootDir)
		os.MkdirAll(cfg.RootDir, 0755)
	}

	return &StaticServer{
		root:        http.Dir(cfg.RootDir),
		cacheMaxAge: cfg.CacheMaxAge,
		indexFiles:  cfg.IndexFiles,
	}
}

// ServeHTTP implements http.Handler
func (ss *StaticServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ss.addCacheHeaders(w, r)
	fs := http.FileServer(ss.root)
	fs.ServeHTTP(w, r)
}

// addCacheHeaders sets appropriate cache control headers
func (ss *StaticServer) addCacheHeaders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", int(ss.cacheMaxAge.Seconds())))

	contentType := ss.detectContentType(r.URL.Path)
	if contentType != "" && w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", contentType)
	}

	// Security headers
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
}

// detectContentType determines MIME type based on extension
func (ss *StaticServer) detectContentType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))

	contentTypes := map[string]string{
		".html": "text/html; charset=utf-8",
		".htm":  "text/html; charset=utf-8",
		".css":  "text/css; charset=utf-8",
		".js":   "application/javascript",
		".mjs":  "application/javascript",
		".json": "application/json",
		".svg":  "image/svg+xml",
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".gif":  "image/gif",
		".webp": "image/webp",
		".ico":  "image/x-icon",
		".txt":  "text/plain; charset=utf-8",
		".xml":  "application/xml",
		".pdf":  "application/pdf",
	}

	if ct, ok := contentTypes[ext]; ok {
		return ct
	}
	return ""
}

// FileExists checks if a static file exists
func (ss *StaticServer) FileExists(path string) bool {
	_, err := ss.root.Open(path)
	return err == nil
}
