package api

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ResponseEnvelope is the standard wrapper for all API responses
type ResponseEnvelope struct {
	Success   bool        `json:"success" xml:"success"`
	Data      interface{} `json:"data,omitempty" xml:"data,omitempty"`
	Error     *ErrorInfo  `json:"error,omitempty" xml:"error,omitempty"`
	Meta      *MetaInfo   `json:"meta,omitempty" xml:"meta,omitempty"`
	Timestamp string      `json:"timestamp" xml:"timestamp"`
	RequestID string      `json:"request_id" xml:"request_id"`
}

// ErrorInfo contains error details
type ErrorInfo struct {
	Code    string `json:"code" xml:"code"`
	Message string `json:"message" xml:"message"`
	Details string `json:"details,omitempty" xml:"details,omitempty"`
}

// MetaInfo contains response metadata
type MetaInfo struct {
	ExecutionTime string `json:"execution_time" xml:"execution_time"`
	Version       string `json:"version" xml:"version"`
}

// SendJSON sends a JSON response with the given data
func SendJSON(w http.ResponseWriter, requestID string, statusCode int, data interface{}, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", requestID)
	w.WriteHeader(statusCode)

	response := buildResponse(requestID, data, err)
	json.NewEncoder(w).Encode(response)
}

// SendXML sends an XML response with the given data
func SendXML(w http.ResponseWriter, requestID string, statusCode int, data interface{}, err error) {
	w.Header().Set("Content-Type", "application/xml")
	w.Header().Set("X-Request-ID", requestID)
	w.WriteHeader(statusCode)

	response := buildResponse(requestID, data, err)
	xml.NewEncoder(w).Encode(response)
}

// SendRedirect sends a redirect response
func SendRedirect(w http.ResponseWriter, requestID string, url string, statusCode int) {
	w.Header().Set("X-Request-ID", requestID)
	http.Redirect(w, nil, url, statusCode) // req is not needed for Redirect
}

// SendError sends an error response with specific code and message
func SendError(w http.ResponseWriter, requestID string, statusCode int, code string, message string, details string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", requestID)
	w.WriteHeader(statusCode)

	response := ResponseEnvelope{
		Success:   false,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		RequestID: requestID,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
	json.NewEncoder(w).Encode(response)
}

// SendUnifiedResponse sends a response in the unified format:
// {"json":<JSON>,"xml":<XML>,"redirect":"<URL>"}
func SendUnifiedResponse(w http.ResponseWriter, requestID string, jsonPayload interface{}, xmlPayload string, redirectURL string, duration time.Duration) {
	w.Header().Set("X-Request-ID", requestID)

	// Determine content type based on Accept header
	accept := w.Header().Get("Accept")
	if strings.Contains(accept, "application/xml") {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)

		// Create XML structure
		xmlResp := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<response>
  <json>%s</json>
  <xml><![CDATA[%s]]></xml>
  <redirect>%s</redirect>
  <meta>
    <execution_time>%s</execution_time>
  </meta>
</response>`,
			escapeJSONForXML(jsonPayload),
			xmlPayload,
			redirectURL,
			duration.String())

		w.Write([]byte(xmlResp))
		return
	}

	// Default to JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	unified := map[string]interface{}{
		"json":     jsonPayload,
		"xml":      xmlPayload,
		"redirect": redirectURL,
		"meta": map[string]string{
			"execution_time": duration.String(),
			"request_id":     requestID,
		},
	}

	json.NewEncoder(w).Encode(unified)
}

// buildResponse creates a standard response envelope
func buildResponse(requestID string, data interface{}, err error) ResponseEnvelope {
	resp := ResponseEnvelope{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		RequestID: requestID,
		Meta: &MetaInfo{
			ExecutionTime: time.Since(time.Now()).String(), // Placeholder, should be passed in
			Version:       "1.0.0",
		},
	}

	if err != nil {
		resp.Success = false
		resp.Error = &ErrorInfo{
			Code:    "INTERNAL_ERROR",
			Message: err.Error(),
		}
	} else {
		resp.Success = true
		resp.Data = data
	}

	return resp
}

// escapeJSONForXML escapes JSON for embedding in XML
func escapeJSONForXML(v interface{}) string {
	data, _ := json.Marshal(v)
	return strings.ReplaceAll(string(data), "&", "&amp;")
}

// FormatError creates a standardized error message
func FormatError(code string, message string, details ...interface{}) *ErrorInfo {
	detailStr := ""
	if len(details) > 0 {
		detailStr = fmt.Sprintf("%v", details[0])
	}
	return &ErrorInfo{
		Code:    code,
		Message: message,
		Details: detailStr,
	}
}

// GetStatusCode returns the appropriate HTTP status code for an error
func GetStatusCode(err error) int {
	if err == nil {
		return http.StatusOK
	}

	// Check for common error types
	switch err.(type) {
	case *ValidationError:
		return http.StatusBadRequest
	case *NotFoundError:
		return http.StatusNotFound
	case *UnauthorizedError:
		return http.StatusUnauthorized
	case *ForbiddenError:
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}

// Custom error types for better error handling
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error on field %s: %s", e.Field, e.Message)
}

type NotFoundError struct {
	Resource string
	ID       string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s with id %s not found", e.Resource, e.ID)
}

type UnauthorizedError struct {
	Message string
}

func (e *UnauthorizedError) Error() string {
	return e.Message
}

type ForbiddenError struct {
	Message string
}

func (e *ForbiddenError) Error() string {
	return e.Message
}
