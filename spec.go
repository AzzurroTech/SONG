package song

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SpecGenerator creates API specifications from the handlers directory
type SpecGenerator struct {
	handlersDir string
	lastScan    time.Time
	cachedSpec  []byte
}

// APISpec represents the top-level API specification
type APISpec struct {
	Name      string         `json:"name"`
	Version   string         `json:"version"`
	Generated string         `json:"generated"`
	Endpoints []EndpointSpec `json:"endpoints"`
}

// EndpointSpec describes a single API endpoint
type EndpointSpec struct {
	Path        string          `json:"path"`
	Method      string          `json:"method"`
	Handler     string          `json:"handler"`
	Description string          `json:"description,omitempty"`
	Parameters  []ParameterSpec `json:"parameters,omitempty"`
}

// ParameterSpec describes a parameter accepted by an endpoint
type ParameterSpec struct {
	Name string `json:"name"`
	In   string `json:"in"`
}

// NewSpecGenerator creates a new spec generator
func NewSpecGenerator(handlersDir string) *SpecGenerator {
	return &SpecGenerator{
		handlersDir: handlersDir,
	}
}

// Generate scans the handlers directory and produces a JSON API specification
func (sg *SpecGenerator) Generate() ([]byte, error) {
	spec := APISpec{
		Name:      "SONG API",
		Version:   "1.0.0",
		Generated: time.Now().UTC().Format(time.RFC3339),
		Endpoints: []EndpointSpec{},
	}

	if _, err := os.Stat(sg.handlersDir); os.IsNotExist(err) {
		sg.cachedSpec, _ = json.MarshalIndent(spec, "", "  ")
		return sg.cachedSpec, nil
	}

	err := filepath.Walk(sg.handlersDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		relPath, err := filepath.Rel(sg.handlersDir, path)
		if err != nil {
			return err
		}

		urlPath := "/" + strings.TrimSuffix(relPath, ".go")
		urlPath = strings.ReplaceAll(urlPath, "\\", "/")

		handlerName := strings.TrimSuffix(filepath.Base(relPath), ".go")

		// Extract params from path
		params := []ParameterSpec{}
		// Simple regex to find {param}
		// In a real scenario, you might parse the path more robustly
		// For now, we just note that params exist if braces are present
		if strings.Contains(urlPath, "{") {
			// Basic extraction logic could go here, but for v1 we keep it simple
			// and just list the path.
		}

		endpoint := EndpointSpec{
			Path:        urlPath,
			Method:      "POST",
			Handler:     handlerName,
			Description: fmt.Sprintf("Handler: %s", handlerName),
			Parameters:  params,
		}

		spec.Endpoints = append(spec.Endpoints, endpoint)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan handlers directory: %w", err)
	}

	output, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal spec: %w", err)
	}

	sg.cachedSpec = output
	sg.lastScan = time.Now()
	log.Printf("📋 API spec generated: %d endpoint(s)", len(spec.Endpoints))

	return output, nil
}

func (sg *SpecGenerator) GetCachedSpec() []byte {
	return sg.cachedSpec
}

func (sg *SpecGenerator) Refresh() ([]byte, error) {
	return sg.Generate()
}
