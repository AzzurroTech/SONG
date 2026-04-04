package templates

import (
	"html/template"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Renderer handles template loading and execution
type Renderer struct {
	mu          sync.RWMutex
	templates   map[string]*template.Template
	templateDir string
	devMode     bool
	lastMod     map[string]time.Time
}

// NewRenderer creates a new template renderer
func NewRenderer(templateDir string) (*Renderer, error) {
	r := &Renderer{
		templates:   make(map[string]*template.Template),
		templateDir: templateDir,
		devMode:     os.Getenv("SONG_ENV") == "development",
		lastMod:     make(map[string]time.Time),
	}

	// Initial load
	if err := r.Reload(); err != nil {
		return nil, err
	}

	return r, nil
}

// Reload loads all templates from the template directory
func (r *Renderer) Reload() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Find all template files
	files, err := filepath.Glob(filepath.Join(r.templateDir, "*.html"))
	if err != nil {
		return err
	}

	if len(files) == 0 {
		log.Printf("Warning: No templates found in %s", r.templateDir)
		return nil
	}

	// Parse all templates together to allow inheritance
	// We use "base" as the name for the combined set, but we track individual files
	tmpl := template.New("")

	for _, file := range files {
		// Read file info to check modification time
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		// Check if file changed since last load
		if !r.devMode && info.ModTime().Equal(r.lastMod[file]) {
			continue
		}
		r.lastMod[file] = info.ModTime()

		// Parse the file
		// We parse all files into the same template set to support {{define}} blocks
		_, err = tmpl.ParseFiles(files...)
		if err != nil {
			return err
		}
	}

	// Store the parsed templates
	// In a more complex setup, we might store them individually by name
	// For now, we store the whole set and look up by name during execution
	// This is a simplified approach. A better approach is to parse each file individually
	// and store them by their base name.

	// Let's refine: Parse each file individually and store by filename
	newTemplates := make(map[string]*template.Template)
	for _, file := range files {
		name := filepath.Base(file)
		t, err := template.ParseFiles(file)
		if err != nil {
			return err
		}
		newTemplates[name] = t
	}

	r.templates = newTemplates
	log.Printf("Loaded %d templates", len(r.templates))
	return nil
}

// Execute renders a template with the given name and data
func (r *Renderer) Execute(name string, w interface{}, data interface{}) error {
	r.mu.RLock()
	tmpl, ok := r.templates[name]
	r.mu.RUnlock()

	if !ok {
		// In dev mode, try to reload and retry
		if r.devMode {
			if err := r.Reload(); err != nil {
				return err
			}
			r.mu.RLock()
			tmpl, ok = r.templates[name]
			r.mu.RUnlock()
		}

		if !ok {
			return os.ErrNotExist
		}
	}

	return tmpl.Execute(w, data)
}

// GetTemplateNames returns a list of available template names
func (r *Renderer) GetTemplateNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.templates))
	for name := range r.templates {
		names = append(names, name)
	}
	return names
}

// Watch starts a background goroutine to watch for template changes (Dev Mode)
func (r *Renderer) Watch(interval time.Duration) {
	if !r.devMode {
		return
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			// Check if any file has changed
			needsReload := false
			files, _ := filepath.Glob(filepath.Join(r.templateDir, "*.html"))

			for _, file := range files {
				info, err := os.Stat(file)
				if err != nil {
					continue
				}
				if !info.ModTime().Equal(r.lastMod[file]) {
					needsReload = true
					break
				}
			}

			if needsReload {
				if err := r.Reload(); err != nil {
					log.Printf("Template reload error: %v", err)
				} else {
					log.Println("Templates reloaded")
				}
			}
		}
	}()
}
