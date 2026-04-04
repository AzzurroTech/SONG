package faas

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// ExecutionResult holds the outcome of a function execution
type ExecutionResult struct {
	Success    bool
	ExitCode   int
	Output     string
	Error      error
	Duration   time.Duration
	Timeout    bool
	MemoryUsed int64   // Approximate, if available
	CPUUsed    float64 // Approximate, if available
}

// Runtime manages the execution of function binaries
type Runtime struct {
	binaryDir  string
	timeout    time.Duration
	mu         sync.Mutex
	activeRuns map[string]*exec.Cmd
}

// NewRuntime creates a new execution runtime
func NewRuntime(binaryDir string, timeout time.Duration) *Runtime {
	return &Runtime{
		binaryDir:  binaryDir,
		timeout:    timeout,
		activeRuns: make(map[string]*exec.Cmd),
	}
}

// Execute runs a compiled function binary with the given arguments
func (rt *Runtime) Execute(functionID, binaryPath string, args []string) (*ExecutionResult, error) {
	rt.mu.Lock()
	if _, exists := rt.activeRuns[functionID]; exists {
		// Allow concurrent executions of the same function?
		// For now, we assume one execution per function ID at a time to simplify state.
		// In a real FaaS, you'd have a pool of workers.
		// Let's allow it but track by a unique execution ID, not function ID.
		// For this simplified version, we'll just use a unique execution ID passed in or generated.
		// Let's assume the caller passes a unique execID or we generate one.
		// Actually, let's change the signature to accept an executionID.
	}
	rt.mu.Unlock()

	// Generate a unique execution ID for tracking
	execID := fmt.Sprintf("%s-%d", functionID, time.Now().UnixNano())

	result := &ExecutionResult{
		Success: false,
	}

	// Check if binary exists
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("binary not found: %s", binaryPath)
	}

	// Create command
	cmd := exec.Command(binaryPath, args...)

	// Set working directory to a temp dir to avoid file system pollution
	tmpDir, err := os.MkdirTemp("", "song-exec-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)
	cmd.Dir = tmpDir

	// Set environment variables (minimal, no inherited env to prevent leakage)
	cmd.Env = []string{
		"PATH=/usr/bin:/bin",
		"HOME=" + tmpDir,
		"LANG=C.UTF-8",
	}

	// Create pipes for stdout/stderr
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the process
	startTime := time.Now()
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	// Track active run
	rt.mu.Lock()
	rt.activeRuns[execID] = cmd
	rt.mu.Unlock()
	defer func() {
		rt.mu.Lock()
		delete(rt.activeRuns, execID)
		rt.mu.Unlock()
	}()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), rt.timeout)
	defer cancel()

	// Wait for completion or timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	// Collect output concurrently
	var stdoutBuf, stderrBuf strings.Builder
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(&stdoutBuf, stdoutPipe)
	}()
	go func() {
		defer wg.Done()
		io.Copy(&stderrBuf, stderrPipe)
	}()

	select {
	case err := <-done:
		// Process finished
		wg.Wait() // Ensure all output is captured

		result.Duration = time.Since(startTime)
		result.ExitCode = cmd.ProcessState.ExitCode()
		result.Output = stdoutBuf.String()

		if err != nil {
			// Check if it was a timeout
			if ctx.Err() == context.DeadlineExceeded {
				result.Timeout = true
				result.Error = fmt.Errorf("execution timed out after %v", rt.timeout)
			} else {
				result.Error = fmt.Errorf("process exited with error: %w\nStderr: %s", err, stderrBuf.String())
			}
		} else {
			result.Success = true
			if len(stderrBuf.String()) > 0 {
				// Log stderr as warning but don't fail
				fmt.Printf("Function %s stderr: %s\n", functionID, stderrBuf.String())
			}
		}

	case <-ctx.Done():
		// Timeout occurred
		wg.Wait() // Try to finish reading buffers

		// Kill the process
		if cmd.Process != nil {
			cmd.Process.Kill()
		}

		result.Duration = time.Since(startTime)
		result.Timeout = true
		result.Error = fmt.Errorf("execution timed out after %v", rt.timeout)
		result.Output = stdoutBuf.String()
	}

	return result, nil
}

// StopAll forcibly stops all running functions
func (rt *Runtime) StopAll() error {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	var lastErr error
	for id, cmd := range rt.activeRuns {
		if cmd.Process != nil {
			if err := cmd.Process.Kill(); err != nil {
				lastErr = fmt.Errorf("failed to kill process %s: %w", id, err)
			}
		}
	}
	return lastErr
}

// GetActiveCount returns the number of currently running functions
func (rt *Runtime) GetActiveCount() int {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return len(rt.activeRuns)
}

// IsRunning checks if a specific execution is still active
func (rt *Runtime) IsRunning(execID string) bool {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	_, exists := rt.activeRuns[execID]
	return exists
}
