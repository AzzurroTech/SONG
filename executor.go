package song

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Executor struct {
	handlersDir string
	timeout     time.Duration
}

func NewExecutor(handlersDir string) *Executor {
	return &Executor{
		handlersDir: handlersDir,
		timeout:     30 * time.Second,
	}
}

func (e *Executor) SetTimeout(d time.Duration) {
	e.timeout = d
}

func (e *Executor) Execute(handlerName string, r *http.Request, pathParams map[string]string) ([]byte, error) {
	handlerPath := filepath.Join(e.handlersDir, handlerName+".go")
	if _, err := os.Stat(handlerPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("handler file not found: %s", handlerPath)
	}

	env := os.Environ()

	for k, v := range pathParams {
		env = append(env, fmt.Sprintf("SONG_PATH_%s=%s", strings.ToUpper(k), v))
	}

	query := r.URL.Query()
	for k, vals := range query {
		env = append(env, fmt.Sprintf("SONG_QUERY_%s=%s", strings.ToUpper(k), strings.Join(vals, ",")))
	}

	if r.Method == http.MethodPost {
		r.ParseForm()
		for k, vals := range r.Form {
			env = append(env, fmt.Sprintf("SONG_FORM_%s=%s", strings.ToUpper(k), strings.Join(vals, ",")))
		}
	}

	bodyBytes, _ := io.ReadAll(r.Body)
	if len(bodyBytes) > 0 {
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		env = append(env, fmt.Sprintf("SONG_RAW_BODY=%s", string(bodyBytes)))
	}

	secrets := GetSecrets()
	for k, v := range secrets {
		env = append(env, fmt.Sprintf("SONG_SECRET_%s=%s", strings.ToUpper(k), v))
	}

	cmd := exec.Command("go", "run", handlerPath)
	cmd.Env = env

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	// Start the process
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start handler: %w", err)
	}

	// Wait for completion with context-based timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		cmd.Process.Kill()
		<-done
		return nil, fmt.Errorf("handler timed out after %v", e.timeout)
	case err := <-done:
		if err != nil {
			log.Printf("Handler %s failed: %v\n%s", handlerName, err, stderr.String())
			return nil, fmt.Errorf("handler execution failed: %w", err)
		}
	}

	output := stdout.Bytes()
	if len(output) > 0 {
		var js json.RawMessage
		if err := json.Unmarshal(output, &js); err != nil {
			log.Printf("Handler %s output is not valid JSON", handlerName)
		}
	}

	log.Printf("Handler %s completed", handlerName)
	return output, nil
}
