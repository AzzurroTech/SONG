package song

import (
	"context"
	"database/sql"
	"time"
)

// Context is the main interface provided to user functions.
// It provides safe access to databases, file system, and logging.
type Context struct {
	ctx         context.Context
	cancel      context.CancelFunc
	dbs         map[string]*sql.DB
	staticDir   string
	dataDir     string
	logger      Logger
	requestID   string
	userID      string
	executionID string
	startTime   time.Time
}

// Logger provides structured logging for functions
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Critical(msg string, args ...interface{})
}

// Request contains information about the incoming HTTP request
type Request struct {
	Method    string
	Path      string
	Query     map[string][]string
	Headers   map[string][]string
	Body      interface{}
	IPAddress string
	UserAgent string
	RequestID string
	UserID    string
}

// Response is what the function returns to be sent back to the client
type Response struct {
	StatusCode int
	Headers    map[string]string
	Body       interface{}
	JSON       interface{}
	XML        string
	HTML       string
	Redirect   string
}

// Function is the interface that user functions must implement
type Function interface {
	// Handle is called when the function is invoked
	Handle(ctx *Context, req *Request) (*Response, error)
}

// FunctionHandler is a simpler function signature for basic use cases
type FunctionHandler func(ctx *Context, req *Request) (*Response, error)

// DBStats contains database connection statistics
type DBStats struct {
	OpenConnections int
	InUse           int
	Idle            int
	WaitCount       int64
	WaitDuration    time.Duration
}

// FileInfo contains information about a file
type FileInfo struct {
	Name    string
	Size    int64
	Mode    string
	ModTime time.Time
	IsDir   bool
}

// Config contains function configuration
type Config struct {
	Name          string
	Version       string
	Timeout       time.Duration
	MemoryLimitMB int64
	DBConnections []string
	Description   string
}

// HealthStatus represents the health of a service
type HealthStatus struct {
	Status    string    `json:"status"` // "healthy", "unhealthy", "degraded"
	Timestamp time.Time `json:"timestamp"`
	Checks    []Check   `json:"checks,omitempty"`
}

// Check represents a single health check
type Check struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// Error represents a function error
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

func (e *Error) Error() string {
	return e.Message
}

// NewError creates a new function error
func NewError(code, message string, detail ...string) *Error {
	d := ""
	if len(detail) > 0 {
		d = detail[0]
	}
	return &Error{
		Code:    code,
		Message: message,
		Detail:  d,
	}
}

// Common error codes
const (
	ErrCodeInvalidInput = "INVALID_INPUT"
	ErrCodeNotFound     = "NOT_FOUND"
	ErrCodeUnauthorized = "UNAUTHORIZED"
	ErrCodeForbidden    = "FORBIDDEN"
	ErrCodeInternal     = "INTERNAL_ERROR"
	ErrCodeTimeout      = "TIMEOUT"
	ErrCodeDBError      = "DATABASE_ERROR"
	ErrCodeFileError    = "FILE_ERROR"
)

// NewResponse creates a new response with default values
func NewResponse() *Response {
	return &Response{
		StatusCode: 200,
		Headers:    make(map[string]string),
	}
}

// WithJSON sets the JSON body and returns the response
func (r *Response) WithJSON(data interface{}) *Response {
	r.JSON = data
	r.Headers["Content-Type"] = "application/json"
	return r
}

// WithXML sets the XML body and returns the response
func (r *Response) WithXML(xml string) *Response {
	r.XML = xml
	r.Headers["Content-Type"] = "application/xml"
	return r
}

// WithHTML sets the HTML body and returns the response
func (r *Response) WithHTML(html string) *Response {
	r.HTML = html
	r.Headers["Content-Type"] = "text/html"
	return r
}

// WithStatus sets the status code and returns the response
func (r *Response) WithStatus(code int) *Response {
	r.StatusCode = code
	return r
}

// WithRedirect sets a redirect and returns the response
func (r *Response) WithRedirect(url string, code ...int) *Response {
	r.Redirect = url
	if len(code) > 0 {
		r.StatusCode = code[0]
	} else {
		r.StatusCode = 302
	}
	return r
}

// WithHeader adds a header and returns the response
func (r *Response) WithHeader(key, value string) *Response {
	r.Headers[key] = value
	return r
}
