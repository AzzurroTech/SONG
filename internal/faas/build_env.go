package faas

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// BuildEnv manages the isolated build environment for user functions
type BuildEnv struct {
	// Base directory for build artifacts
	buildDir string

	// Allowed import paths (whitelist)
	allowedImports []string

	// Go version to use for compilation
	goVersion string

	// Build timeout
	timeout time.Duration

	// Whether to enable verbose build output
	verbose bool
}

// BuildResult contains the outcome of a build operation
type BuildResult struct {
	Success    bool
	BinaryPath string
	Output     string
	Duration   time.Duration
	Error      error
	Warnings   []string
}

// NewBuildEnv creates a new build environment
func NewBuildEnv(buildDir string, verbose bool) *BuildEnv {
	// Define the standard library + song-sdk imports that are allowed
	allowedImports := []string{
		// Standard library
		"bytes", "context", "encoding", "encoding/json", "encoding/xml",
		"errors", "fmt", "io", "log", "net/http", "os", "path/filepath",
		"sort", "strconv", "strings", "sync", "time", "unicode",
		// Song SDK (will be vendored or replaced during build)
		"song",
	}

	return &BuildEnv{
		buildDir:       buildDir,
		allowedImports: allowedImports,
		goVersion:      runtime.Version(),
		timeout:        60 * time.Second,
		verbose:        verbose,
	}
}

// PrepareBuildDir creates a clean build directory for a function
func (be *BuildEnv) PrepareBuildDir(functionID string) (string, error) {
	// Create a unique build directory
	buildPath := filepath.Join(be.buildDir, functionID, time.Now().Format("20060102-150405"))

	if err := os.MkdirAll(buildPath, 0700); err != nil {
		return "", fmt.Errorf("failed to create build directory: %w", err)
	}

	return buildPath, nil
}

// CleanupBuildDir removes a build directory after compilation
func (be *BuildEnv) CleanupBuildDir(buildPath string) error {
	parentDir := filepath.Dir(buildPath)
	return os.RemoveAll(parentDir)
}

// Build compiles a Go source file in the isolated environment
func (be *BuildEnv) Build(sourcePath, outputPath, functionID string) *BuildResult {
	startTime := time.Now()
	result := &BuildResult{
		Success: false,
	}

	// Create build directory
	buildPath, err := be.PrepareBuildDir(functionID)
	if err != nil {
		result.Error = err
		return result
	}
	defer be.CleanupBuildDir(buildPath)

	// Copy source file to build directory
	sourceFileName := filepath.Base(sourcePath)
	targetSourcePath := filepath.Join(buildPath, sourceFileName)

	if err := copyFile(sourcePath, targetSourcePath); err != nil {
		result.Error = fmt.Errorf("failed to copy source file: %w", err)
		return result
	}

	// Initialize Go module
	modulePath := fmt.Sprintf("song/function/%s", functionID)
	if err := be.initModule(buildPath, modulePath); err != nil {
		result.Error = fmt.Errorf("failed to initialize module: %w", err)
		return result
	}

	// Analyze imports
	imports, err := be.analyzeImports(targetSourcePath)
	if err != nil {
		result.Error = fmt.Errorf("failed to analyze imports: %w", err)
		return result
	}

	// Validate imports against whitelist
	violations := be.validateImports(imports)
	if len(violations) > 0 {
		result.Warnings = violations
		result.Error = fmt.Errorf("disallowed imports detected: %v", violations)
		return result
	}

	// Run go build with timeout
	cmd := exec.Command("go", "build", "-o", outputPath, sourceFileName)
	cmd.Dir = buildPath
	cmd.Env = append(os.Environ(),
		"GOOS="+runtime.GOOS,
		"GOARCH="+runtime.GOARCH,
		"CGO_ENABLED=0", // Disable CGO for portability
	)

	// Set timeout
	ctx, cancel := be.createTimeoutContext()
	defer cancel()
	cmd.Cancel = ctx.Cancel
	cmd.WaitDelay = 5 * time.Second

	// Capture output
	output, err := cmd.CombinedOutput()
	result.Output = string(output)
	result.Duration = time.Since(startTime)

	if err != nil {
		result.Error = fmt.Errorf("build failed: %w\nOutput: %s", err, result.Output)
		return result
	}

	// Verify binary was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		result.Error = fmt.Errorf("binary was not created at %s", outputPath)
		return result
	}

	result.Success = true
	return result
}

// initModule creates a go.mod file in the build directory
func (be *BuildEnv) initModule(buildPath, modulePath string) error {
	modContent := fmt.Sprintf("module %s\n\ngo 1.21\n", modulePath)

	modPath := filepath.Join(buildPath, "go.mod")
	if err := os.WriteFile(modPath, []byte(modContent), 0600); err != nil {
		return err
	}

	return nil
}

// analyzeImports extracts import paths from a Go source file
func (be *BuildEnv) analyzeImports(sourcePath string) ([]string, error) {
	// Use go list to analyze imports
	cmd := exec.Command("go", "list", "-f", "{{join .Imports \"\\n\"}}", sourcePath)
	output, err := cmd.Output()
	if err != nil {
		// Fallback: try to parse imports manually
		return be.parseImportsManually(sourcePath)
	}

	imports := strings.Split(strings.TrimSpace(string(output)), "\n")
	return imports, nil
}

// parseImportsManually extracts imports from source code (fallback)
func (be *BuildEnv) parseImportsManually(sourcePath string) ([]string, error) {
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, err
	}

	var imports []string
	lines := strings.Split(string(content), "\n")
	inImportBlock := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "import (") {
			inImportBlock = true
			continue
		}

		if inImportBlock && line == ")" {
			inImportBlock = false
			continue
		}

		if inImportBlock {
			// Extract import path from "package" or "package"
			importPath := strings.Trim(line, "\"")
			if importPath != "" {
				imports = append(imports, importPath)
			}
		} else if strings.HasPrefix(line, "import \"") {
			importPath := strings.Trim(strings.TrimPrefix(line, "import \""), "\"")
			imports = append(imports, importPath)
		}
	}

	return imports, nil
}

// validateImports checks if all imports are in the allowed list
func (be *BuildEnv) validateImports(imports []string) []string {
	var violations []string

	for _, imp := range imports {
		// Skip empty imports
		if imp == "" {
			continue
		}

		// Check if import is allowed
		isAllowed := false
		for _, allowed := range be.allowedImports {
			// Exact match or standard library prefix
			if imp == allowed || strings.HasPrefix(imp, allowed+"/") {
				isAllowed = true
				break
			}
		}

		// Also allow standard library packages (they don't have a prefix)
		if !isAllowed && isStandardLibrary(imp) {
			isAllowed = true
		}

		if !isAllowed {
			violations = append(violations, imp)
		}
	}

	return violations
}

// isStandardLibrary checks if an import is from the Go standard library
func isStandardLibrary(importPath string) bool {
	// Standard library packages don't contain dots in their root path
	// (except for subpackages like encoding/json)
	parts := strings.Split(importPath, "/")
	root := parts[0]

	// Standard library roots don't contain dots
	return !strings.Contains(root, ".")
}

// createTimeoutContext creates a context with timeout for the build command
func (be *BuildEnv) createTimeoutContext() (interface{}, func()) {
	// This is a simplified version. In production, use context.WithTimeout
	// For now, we return a dummy implementation
	return nil, func() {}
}

// copyFile copies a file from source to destination
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
