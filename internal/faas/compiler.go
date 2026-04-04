package faas

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/azzurrotech/song/internal/core"
)

// Compiler orchestrates the compilation of user functions
type Compiler struct {
	buildEnv    *BuildEnv
	coreStore   *core.Store
	functionMgr *core.FunctionManager
	binaryDir   string
}

// NewCompiler creates a new compiler instance
func NewCompiler(buildDir, binaryDir string, store *core.Store) *Compiler {
	return &Compiler{
		buildEnv:    NewBuildEnv(buildDir, false),
		coreStore:   store,
		binaryDir:   binaryDir,
		functionMgr: core.NewFunctionManager(store),
	}
}

// CompileResult wraps the build result with function metadata
type CompileResult struct {
	FunctionID string
	BinaryPath string
	Success    bool
	Error      error
	Duration   time.Duration
	Warnings   []string
	SourceHash string
}

// Compile compiles a function source code into a binary
func (c *Compiler) Compile(functionID, sourceCode string) (*CompileResult, error) {
	startTime := time.Now()
	result := &CompileResult{
		FunctionID: functionID,
		Success:    false,
	}

	// Ensure binary directory exists
	if err := os.MkdirAll(c.binaryDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create binary directory: %w", err)
	}

	// Create a temporary file for the source code
	tmpFile, err := os.CreateTemp("", "song-func-*.go")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp source file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath) // Clean up temp file

	// Write source code to temp file
	if _, err := tmpFile.WriteString(sourceCode); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("failed to write source code: %w", err)
	}
	tmpFile.Close()

	// Define output binary path
	binaryName := fmt.Sprintf("%s.bin", functionID)
	binaryPath := filepath.Join(c.binaryDir, binaryName)

	// Remove existing binary if it exists
	if _, err := os.Stat(binaryPath); err == nil {
		os.Remove(binaryPath)
	}

	// Run the build
	buildResult := c.buildEnv.Build(tmpPath, binaryPath, functionID)

	result.Duration = time.Since(startTime)
	result.Success = buildResult.Success
	result.BinaryPath = binaryPath
	result.Warnings = buildResult.Warnings
	result.Error = buildResult.Error

	// Calculate source hash for versioning
	// (In production, use a proper hash library like crypto/sha256)
	result.SourceHash = fmt.Sprintf("%x", []byte(sourceCode)[:8])

	// Update function metadata in core store
	if result.Success {
		// Update function status to active
		if err := c.functionMgr.Update(functionID, map[string]interface{}{
			"status":      core.StatusActive,
			"binary_path": binaryPath,
			"source_code": core.EncodeSource(sourceCode),
		}); err != nil {
			// Log error but don't fail the compile
			fmt.Printf("Warning: Failed to update function metadata: %v\n", err)
		}
	} else {
		// Update function status to failed
		if err := c.functionMgr.Update(functionID, map[string]interface{}{
			"status": core.StatusFailed,
		}); err != nil {
			fmt.Printf("Warning: Failed to update function status: %v\n", err)
		}
	}

	return result, nil
}

// Rebuild triggers a rebuild of an existing function
func (c *Compiler) Rebuild(functionID string) (*CompileResult, error) {
	// Get current function source
	fn, err := c.functionMgr.GetByID(functionID)
	if err != nil {
		return nil, fmt.Errorf("function not found: %w", err)
	}

	// Decode source
	source, err := core.DecodeSource(fn.SourceCode)
	if err != nil {
		return nil, fmt.Errorf("failed to decode source: %w", err)
	}

	return c.Compile(functionID, source)
}

// GetBinaryPath returns the expected binary path for a function
func (c *Compiler) GetBinaryPath(functionID string) string {
	return filepath.Join(c.binaryDir, fmt.Sprintf("%s.bin", functionID))
}

// BinaryExists checks if a compiled binary exists for a function
func (c *Compiler) BinaryExists(functionID string) bool {
	path := c.GetBinaryPath(functionID)
	_, err := os.Stat(path)
	return err == nil
}
