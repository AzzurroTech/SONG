package core

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Store manages JSON file storage with thread-safe operations
type Store struct {
	mu        sync.RWMutex
	baseDir   string
	fileLocks map[string]*sync.Mutex
	lockMu    sync.Mutex // Protects the fileLocks map
}

// NewStore creates a new JSON store in the specified directory
func NewStore(baseDir string) (*Store, error) {
	// Ensure the core directory exists with restricted permissions
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create core directory: %w", err)
	}

	return &Store{
		baseDir:   baseDir,
		fileLocks: make(map[string]*sync.Mutex),
	}, nil
}

// getFileLock returns a mutex for a specific file to ensure process-wide safety
func (s *Store) getFileLock(filename string) *sync.Mutex {
	s.lockMu.Lock()
	defer s.lockMu.Unlock()

	if _, exists := s.fileLocks[filename]; !exists {
		s.fileLocks[filename] = &sync.Mutex{}
	}
	return s.fileLocks[filename]
}

// Read reads a JSON file into the provided interface
func (s *Store) Read(filename string, v interface{}) error {
	fileMu := s.getFileLock(filename)
	fileMu.Lock()
	defer fileMu.Unlock()

	s.mu.RLock()
	defer s.mu.RUnlock()

	path := filepath.Join(s.baseDir, filename)

	data, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty data if file doesn't exist yet
			return nil
		}
		return fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	if len(data) == 0 {
		return nil
	}

	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("failed to unmarshal JSON from %s: %w", filename, err)
	}

	return nil
}

// Write writes data to a JSON file atomically
func (s *Store) Write(filename string, v interface{}) error {
	fileMu := s.getFileLock(filename)
	fileMu.Lock()
	defer fileMu.Unlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.baseDir, filename)
	tempPath := path + ".tmp"

	// Marshal with indentation for readability
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON for %s: %w", filename, err)
	}

	// Write to temporary file first
	if err := ioutil.WriteFile(tempPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write temp file %s: %w", filename, err)
	}

	// Rename temp file to target (atomic operation on most systems)
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("failed to rename temp file to %s: %w", filename, err)
	}

	return nil
}

// AppendLog appends a JSON line to a log file (for logs.jsonl)
func (s *Store) AppendLog(filename string, entry interface{}) error {
	fileMu := s.getFileLock(filename)
	fileMu.Lock()
	defer fileMu.Unlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	path := filepath.Join(s.baseDir, filename)

	// Open file in append mode, create if not exists
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open log file %s: %w", filename, err)
	}
	defer f.Close()

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	// Write JSON line with newline
	if _, err := f.WriteString(string(data) + "\n"); err != nil {
		return fmt.Errorf("failed to append to log file %s: %w", filename, err)
	}

	return nil
}

// UpdateTimestamp updates the last_modified field in a JSON structure
func (s *Store) UpdateTimestamp(data map[string]interface{}) {
	data["last_updated"] = time.Now().UTC().Format(time.RFC3339)
}
