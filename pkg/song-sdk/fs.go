package song

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// FS provides safe file system access for user functions
type FS struct {
	staticDir string
	dataDir   string
	logger    Logger
}

// NewFS creates a new file system wrapper
func NewFS(staticDir, dataDir string, logger Logger) *FS {
	return &FS{
		staticDir: staticDir,
		dataDir:   dataDir,
		logger:    logger,
	}
}

// ReadStaticFile reads a file from the static directory.
// The path must be relative and cannot traverse outside the static directory.
func (fs *FS) ReadStaticFile(relativePath string) ([]byte, error) {
	return fs.readFile(fs.staticDir, relativePath, "static")
}

// ReadDataFile reads a file from the data directory.
// The path must be relative and cannot traverse outside the data directory.
func (fs *FS) ReadDataFile(relativePath string) ([]byte, error) {
	return fs.readFile(fs.dataDir, relativePath, "data")
}

// readFile is the internal helper that performs the actual read with safety checks
func (fs *FS) readFile(baseDir, relativePath, dirType string) ([]byte, error) {
	if baseDir == "" {
		return nil, fmt.Errorf("%s directory not configured", dirType)
	}

	// Clean the relative path to remove any .. or . components
	cleanPath := filepath.Clean(relativePath)

	// Reject absolute paths or paths trying to escape
	if filepath.IsAbs(cleanPath) || strings.HasPrefix(cleanPath, "..") || strings.Contains(cleanPath, "../") {
		fs.logger.Warn("Attempted path traversal blocked", "dir", dirType, "requested", relativePath, "cleaned", cleanPath)
		return nil, fmt.Errorf("invalid path: path traversal detected")
	}

	// Construct the full path
	fullPath := filepath.Join(baseDir, cleanPath)

	// Double-check that the resolved path is still within the base directory
	// This handles symlinks and other edge cases
	resolvedPath, err := filepath.Abs(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	baseAbs, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve base dir: %w", err)
	}

	if !strings.HasPrefix(resolvedPath, baseAbs+string(os.PathSeparator)) && resolvedPath != baseAbs {
		fs.logger.Warn("Path traversal attempt detected after resolution", "base", baseAbs, "resolved", resolvedPath)
		return nil, fmt.Errorf("access denied: path outside allowed directory")
	}

	// Check if file exists
	info, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("file not found: %s", relativePath)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Prevent reading directories
	if info.IsDir() {
		return nil, fmt.Errorf("cannot read directory: %s", relativePath)
	}

	// Read the file
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	fs.logger.Debug("File read successfully", "dir", dirType, "path", relativePath, "size", len(data))
	return data, nil
}

// ReadStaticFileStream opens a stream to read a file from the static directory.
// Useful for large files to avoid loading everything into memory.
func (fs *FS) ReadStaticFileStream(relativePath string) (io.ReadCloser, error) {
	return fs.openStream(fs.staticDir, relativePath, "static")
}

// ReadDataFileStream opens a stream to read a file from the data directory.
func (fs *FS) ReadDataFileStream(relativePath string) (io.ReadCloser, error) {
	return fs.openStream(fs.dataDir, relativePath, "data")
}

// openStream is the internal helper for streaming file access
func (fs *FS) openStream(baseDir, relativePath, dirType string) (io.ReadCloser, error) {
	if baseDir == "" {
		return nil, fmt.Errorf("%s directory not configured", dirType)
	}

	cleanPath := filepath.Clean(relativePath)

	if filepath.IsAbs(cleanPath) || strings.HasPrefix(cleanPath, "..") || strings.Contains(cleanPath, "../") {
		fs.logger.Warn("Attempted path traversal blocked (stream)", "dir", dirType, "requested", relativePath)
		return nil, fmt.Errorf("invalid path: path traversal detected")
	}

	fullPath := filepath.Join(baseDir, cleanPath)

	resolvedPath, err := filepath.Abs(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path: %w", err)
	}

	baseAbs, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve base dir: %w", err)
	}

	if !strings.HasPrefix(resolvedPath, baseAbs+string(os.PathSeparator)) && resolvedPath != baseAbs {
		fs.logger.Warn("Path traversal attempt detected (stream)", "base", baseAbs, "resolved", resolvedPath)
		return nil, fmt.Errorf("access denied: path outside allowed directory")
	}

	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", relativePath)
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	if info.IsDir() {
		file.Close()
		return nil, fmt.Errorf("cannot open directory as stream: %s", relativePath)
	}

	fs.logger.Debug("File stream opened", "dir", dirType, "path", relativePath)
	return file, nil
}

// GetStaticDir returns the absolute path to the static directory
func (fs *FS) GetStaticDir() string {
	return fs.staticDir
}

// GetDataDir returns the absolute path to the data directory
func (fs *FS) GetDataDir() string {
	return fs.dataDir
}

// FileExists checks if a file exists in the static or data directory
func (fs *FS) FileExists(relativePath string, dirType string) (bool, error) {
	var baseDir string
	if dirType == "static" {
		baseDir = fs.staticDir
	} else if dirType == "data" {
		baseDir = fs.dataDir
	} else {
		return false, fmt.Errorf("invalid directory type: %s", dirType)
	}

	if baseDir == "" {
		return false, fmt.Errorf("%s directory not configured", dirType)
	}

	cleanPath := filepath.Clean(relativePath)
	if filepath.IsAbs(cleanPath) || strings.HasPrefix(cleanPath, "..") || strings.Contains(cleanPath, "../") {
		return false, fmt.Errorf("invalid path")
	}

	fullPath := filepath.Join(baseDir, cleanPath)
	_, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check file existence: %w", err)
	}
	return true, nil
}
