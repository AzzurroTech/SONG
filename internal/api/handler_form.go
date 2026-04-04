package api

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/azzurrotech/song/internal/core"
)

// FormProcessor handles HTML form submissions
type FormProcessor struct {
	coreStore *core.Store
	logMgr    *core.LogManager
}

// FormResult represents the unified response format
type FormResult struct {
	JSON     map[string]interface{} `json:"json"`
	XML      string                 `xml:"-"`
	Redirect string                 `json:"redirect,omitempty"`
}

// FormData represents parsed form data
type FormData struct {
	Method      string            `json:"method"`
	Action      string            `json:"action"`
	ContentType string            `json:"content_type"`
	Fields      map[string]string `json:"fields"`
	Files       map[string]string `json:"files,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
	RequestID   string            `json:"request_id"`
}

// NewFormProcessor creates a new form processor
func NewFormProcessor(store *core.Store) *FormProcessor {
	return &FormProcessor{
		coreStore: store,
		logMgr:    core.NewLogManager(store, "form-processor"),
	}
}

// ProcessFormHandler handles form submissions
func (fp *FormProcessor) ProcessFormHandler(w http.ResponseWriter, req *http.Request) {
	requestID := req.Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = generateRequestID()
	}

	// Log the form submission
	fp.logMgr.Info("Form submission received", map[string]interface{}{
		"request_id":   requestID,
		"method":       req.Method,
		"path":         req.URL.Path,
		"content_type": req.Header.Get("Content-Type"),
	})

	// Parse form based on method and content type
	formData, err := fp.parseForm(req)
	if err != nil {
		fp.logMgr.Error("Form parsing failed", map[string]interface{}{
			"request_id": requestID,
			"error":      err.Error(),
		})
		http.Error(w, fmt.Sprintf("Failed to parse form: %v", err), http.StatusBadRequest)
		return
	}

	// Process the form data
	result, err := fp.processFormData(formData, req)
	if err != nil {
		fp.logMgr.Error("Form processing failed", map[string]interface{}{
			"request_id": requestID,
			"error":      err.Error(),
		})
		http.Error(w, fmt.Sprintf("Form processing failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Determine response format based on Accept header
	accept := req.Header.Get("Accept")
	responseFormat := "json" // default

	if strings.Contains(accept, "application/xml") {
		responseFormat = "xml"
	} else if strings.Contains(accept, "text/html") {
		responseFormat = "html"
	}

	// Send response
	w.Header().Set("X-Request-ID", requestID)
	w.Header().Set("X-Form-Processed", "true")

	switch responseFormat {
	case "xml":
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(fp.resultToXML(result)))
	case "html":
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(fp.resultToHTML(result)))
	default:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}

	// Handle redirect if specified
	if result.Redirect != "" {
		// Note: For redirects, we typically send a 302 response before the body
		// This is handled by the caller or we can do it here if needed
	}
}

// parseForm extracts form data from the request
func (fp *FormProcessor) parseForm(req *http.Request) (*FormData, error) {
	formData := &FormData{
		Method:      req.Method,
		Action:      req.URL.Path,
		ContentType: req.Header.Get("Content-Type"),
		Fields:      make(map[string]string),
		Files:       make(map[string]string),
		Timestamp:   time.Now().UTC(),
		RequestID:   req.Header.Get("X-Request-ID"),
	}

	contentType := req.Header.Get("Content-Type")

	switch req.Method {
	case "GET":
		// GET requests use query parameters
		for key, values := range req.URL.Query() {
			if len(values) == 1 {
				formData.Fields[key] = values[0]
			} else {
				// Join multiple values with comma
				formData.Fields[key] = strings.Join(values, ",")
			}
		}

	case "POST", "PUT", "DELETE", "PATCH":
		// Check content type for proper parsing
		if strings.Contains(contentType, "multipart/form-data") {
			// Multipart form data (includes file uploads)
			if err := req.ParseMultipartForm(32 << 20); err != nil { // 32MB max
				return nil, fmt.Errorf("failed to parse multipart form: %w", err)
			}

			// Extract regular fields
			for key, values := range req.MultipartForm.Value {
				if len(values) == 1 {
					formData.Fields[key] = values[0]
				} else {
					formData.Fields[key] = strings.Join(values, ",")
				}
			}

			// Extract file information
			for key, files := range req.MultipartForm.File {
				fileNames := make([]string, len(files))
				for i, f := range files {
					fileNames[i] = f.Filename
				}
				if len(fileNames) == 1 {
					formData.Files[key] = fileNames[0]
				} else {
					formData.Files[key] = strings.Join(fileNames, ",")
				}
			}

		} else if strings.Contains(contentType, "application/x-www-form-urlencoded") {
			// URL-encoded form data
			if err := req.ParseForm(); err != nil {
				return nil, fmt.Errorf("failed to parse form: %w", err)
			}

			for key, values := range req.PostForm {
				if len(values) == 1 {
					formData.Fields[key] = values[0]
				} else {
					formData.Fields[key] = strings.Join(values, ",")
				}
			}

		} else if strings.Contains(contentType, "application/json") {
			// JSON body (treated as form data for compatibility)
			var jsonData map[string]interface{}
			if err := json.NewDecoder(req.Body).Decode(&jsonData); err != nil {
				return nil, fmt.Errorf("failed to parse JSON body: %w", err)
			}
			defer req.Body.Close()

			// Convert to string map
			for key, value := range jsonData {
				switch v := value.(type) {
				case string:
					formData.Fields[key] = v
				case []interface{}:
					strings := make([]string, len(v))
					for i, item := range v {
						if s, ok := item.(string); ok {
							strings[i] = s
						}
					}
					formData.Fields[key] = strings.Join(strings, ",")
				default:
					formData.Fields[key] = fmt.Sprintf("%v", value)
				}
			}

		} else if strings.Contains(contentType, "text/xml") {
			// XML body
			// For XML, we parse it into a map structure
			// This is a simplified implementation
			body, err := io.ReadAll(req.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read XML body: %w", err)
			}
			defer req.Body.Close()

			// Parse XML into fields (simplified)
			formData.Fields["raw_xml"] = string(body)

		} else {
			// Plain text or unknown format
			body, err := io.ReadAll(req.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read body: %w", err)
			}
			defer req.Body.Close()

			formData.Fields["raw_body"] = string(body)
		}

	default:
		return nil, fmt.Errorf("unsupported HTTP method: %s", req.Method)
	}

	return formData, nil
}

// processFormData handles the business logic for form processing
func (fp *FormProcessor) processFormData(formData *FormData, req *http.Request) (*FormResult, error) {
	result := &FormResult{
		JSON: make(map[string]interface{}),
	}

	// Add form data to JSON result
	result.JSON["method"] = formData.Method
	result.JSON["action"] = formData.Action
	result.JSON["fields"] = formData.Fields
	result.JSON["files"] = formData.Files
	result.JSON["timestamp"] = formData.Timestamp.Format(time.RFC3339)
	result.JSON["request_id"] = formData.RequestID

	// Check for redirect URL in form data
	if redirectURL, ok := formData.Fields["_redirect"]; ok {
		result.Redirect = redirectURL
		delete(formData.Fields, "_redirect")
		result.JSON["redirect_url"] = redirectURL
	}

	// Check for custom response format
	if responseType, ok := formData.Fields["_response_type"]; ok {
		result.JSON["response_type"] = responseType
		delete(formData.Fields, "_response_type")
	}

	// Log successful processing
	fp.logMgr.Info("Form processed successfully", map[string]interface{}{
		"request_id":   formData.RequestID,
		"field_count":  len(formData.Fields),
		"file_count":   len(formData.Files),
		"has_redirect": result.Redirect != "",
	})

	return result, nil
}

// resultToXML converts the result to XML format
func (fp *FormProcessor) resultToXML(result *FormResult) string {
	var sb strings.Builder
	sb.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	sb.WriteString("<form-result>\n")

	// Add JSON data as XML
	for key, value := range result.JSON {
		sb.WriteString(fmt.Sprintf("  <field name=\"%s\">%v</field>\n", key, xml.EscapeText([]byte(fmt.Sprintf("%v", value)))))
	}

	// Add redirect if present
	if result.Redirect != "" {
		sb.WriteString(fmt.Sprintf("  <redirect>%s</redirect>\n", xml.EscapeText([]byte(result.Redirect))))
	}

	sb.WriteString("</form-result>")
	return sb.String()
}

// resultToHTML converts the result to HTML format
func (fp *FormProcessor) resultToHTML(result *FormResult) string {
	var sb strings.Builder
	sb.WriteString("<!DOCTYPE html>\n<html><head><title>Form Result</title></head><body>\n")
	sb.WriteString("<h1>Form Processing Result</h1>\n")
	sb.WriteString("<pre>\n")

	for key, value := range result.JSON {
		sb.WriteString(fmt.Sprintf("%s: %v\n", key, value))
	}

	if result.Redirect != "" {
		sb.WriteString(fmt.Sprintf("\nRedirecting to: %s\n", result.Redirect))
	}

	sb.WriteString("</pre>\n</body></html>")
	return sb.String()
}

// formProcessingMiddleware adds form-specific headers and validation
func (r *Router) formProcessingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Validate form size (prevent DoS via large payloads)
		if req.ContentLength > 32<<20 { // 32MB
			http.Error(w, "Payload too large", http.StatusRequestEntityTooLarge)
			return
		}

		// Add form-specific headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")

		next.ServeHTTP(w, req)
	})
}
