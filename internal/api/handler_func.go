package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/azzurrotech/song/internal/core"
	"github.com/azzurrotech/song/internal/faas"
	"github.com/gorilla/mux"
)

// TriggerFunctionHandler handles HTTP requests to execute a function
func (r *Router) triggerFunctionHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	functionName := vars["functionName"]

	// Get function from registry
	entry, err := r.faasEngine.Registry.GetByName(functionName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Function not found: %s", functionName), http.StatusNotFound)
		return
	}

	// Check if function is active
	if entry.Status != core.StatusActive {
		http.Error(w, fmt.Sprintf("Function is not active: %s (status: %s)", functionName, entry.Status), http.StatusServiceUnavailable)
		return
	}

	// Extract request context
	requestID := req.Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = generateRequestID()
	}

	// Get authenticated user (if any)
	userID := ""
	if user, ok := r.authSvc.GetCurrentUser(req); ok {
		userID = user.ID
	}

	// Parse request body based on content type
	var requestBody interface{}
	contentType := req.Header.Get("Content-Type")

	if req.Method == "POST" || req.Method == "PUT" {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer req.Body.Close()

		// Parse based on content type
		switch {
		case strings.Contains(contentType, "application/json"):
			var jsonBody map[string]interface{}
			if err := json.Unmarshal(body, &jsonBody); err != nil {
				http.Error(w, "Invalid JSON body", http.StatusBadRequest)
				return
			}
			requestBody = jsonBody

		case strings.Contains(contentType, "application/x-www-form-urlencoded"):
			req.ParseForm()
			formMap := make(map[string]interface{})
			for key, values := range req.Form {
				if len(values) == 1 {
					formMap[key] = values[0]
				} else {
					formMap[key] = values
				}
			}
			requestBody = formMap

		case strings.Contains(contentType, "multipart/form-data"):
			if err := req.ParseMultipartForm(32 << 20); err != nil {
				http.Error(w, "Failed to parse multipart form", http.StatusBadRequest)
				return
			}
			multipartMap := make(map[string]interface{})
			for key, values := range req.MultipartForm.Value {
				if len(values) == 1 {
					multipartMap[key] = values[0]
				} else {
					multipartMap[key] = values
				}
			}
			// Handle files
			for key, files := range req.MultipartForm.File {
				if len(files) == 1 {
					multipartMap[key] = files[0].Filename
				} else {
					filenames := make([]string, len(files))
					for i, f := range files {
						filenames[i] = f.Filename
					}
					multipartMap[key] = filenames
				}
			}
			requestBody = multipartMap

		default:
			// Treat as raw string
			requestBody = string(body)
		}
	} else {
		// GET/DELETE: Use query parameters
		queryMap := make(map[string]interface{})
		for key, values := range req.URL.Query() {
			if len(values) == 1 {
				queryMap[key] = values[0]
			} else {
				queryMap[key] = values
			}
		}
		requestBody = queryMap
	}

	// Build function input
	input := &faas.FunctionInput{
		FunctionID: entry.ID,
		Method:     req.Method,
		Path:       req.URL.Path,
		Query:      req.URL.Query(),
		Headers:    req.Header,
		Body:       requestBody,
		RequestID:  requestID,
		UserID:     userID,
		IPAddress:  getIPAddress(req),
		UserAgent:  req.UserAgent(),
	}

	// Execute the function
	result, err := r.faasEngine.Execute(input)
	if err != nil {
		http.Error(w, fmt.Sprintf("Function execution failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Increment execution count
	go r.faasEngine.Registry.IncrementExecCount(entry.ID)

	// Handle response
	if result.Redirect != "" {
		http.Redirect(w, req, result.Redirect, http.StatusFound)
		return
	}

	// Set response headers
	w.Header().Set("X-Request-ID", requestID)
	w.Header().Set("X-Execution-Time", result.Duration.String())

	// Return response in requested format
	accept := req.Header.Get("Accept")
	switch {
	case strings.Contains(accept, "application/xml"):
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(result.XML))
	case strings.Contains(accept, "text/html"):
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(result.HTML))
	default:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"json":       result.JSON,
			"xml":        result.XML,
			"redirect":   result.Redirect,
			"duration":   result.Duration.String(),
			"request_id": requestID,
		})
	}
}

// FunctionInput represents the input passed to a function
type FunctionInput struct {
	FunctionID string
	Method     string
	Path       string
	Query      url.Values
	Headers    http.Header
	Body       interface{}
	RequestID  string
	UserID     string
	IPAddress  string
	UserAgent  string
}

// FunctionResult represents the output from a function execution
type FunctionResult struct {
	JSON     interface{}
	XML      string
	HTML     string
	Redirect string
	Duration time.Duration
	Error    error
}

// getIPAddress extracts the client IP address from the request
func getIPAddress(req *http.Request) string {
	// Check X-Forwarded-For header first (for proxies)
	if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	if xri := req.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return strings.Split(req.RemoteAddr, ":")[0]
}

// functionExecutionMiddleware prepares context for function execution
func (r *Router) functionExecutionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Add timing header
		start := time.Now()
		req.Header.Set("X-Start-Time", start.Format(time.RFC3339Nano))

		next.ServeHTTP(w, req)
	})
}

// devTestFunctionHandler allows developers to test their functions
func (r *Router) devTestFunctionHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	functionID := vars["id"]

	// Get function
	entry, err := r.faasEngine.Registry.Get(functionID)
	if err != nil {
		http.Error(w, "Function not found", http.StatusNotFound)
		return
	}

	// Check ownership (developers can only test their own functions)
	user, ok := r.authSvc.GetCurrentUser(req)
	if !ok || (user.Role != "admin" && entry.OwnerID != user.ID) {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// Parse test input
	var testInput map[string]interface{}
	if err := json.NewDecoder(req.Body).Decode(&testInput); err != nil {
		http.Error(w, "Invalid test input", http.StatusBadRequest)
		return
	}

	// Build function input
	input := &faas.FunctionInput{
		FunctionID: entry.ID,
		Method:     "POST",
		Body:       testInput,
		RequestID:  generateRequestID(),
		UserID:     user.ID,
	}

	// Execute
	result, err := r.faasEngine.Execute(input)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Return result
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"json":     result.JSON,
		"duration": result.Duration.String(),
	})
}
