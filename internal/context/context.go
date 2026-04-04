package context

import (
	"context"
	"errors"
	"time"

	"github.com/azzurrotech/song/internal/db"
)

// ErrDBNotFound is returned when a requested database pool is not found
var ErrDBNotFound = errors.New("database pool not found")

// Context is the main structure injected into user functions.
// It provides safe, controlled access to system resources like databases,
// file paths, and logging.
type Context struct {
	ctx       context.Context
	cancel    context.CancelFunc
	startTime time.Time

	// Database access (multiple pools supported)
	// Key is the database name (e.g., "default", "analytics")
	dbs map[string]*db.Pool

	// File system access (paths only, actual access via SDK)
	staticDir string
	dataDir   string

	// Logging interface
	logger Logger

	// Request metadata
	requestID   string
	userID      string
	executionID string
}

// Logger interface for logging within functions.
// Implementations should handle formatting and output.
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Critical(msg string, args ...interface{})
}

// NewContext creates a new function execution context.
// It initializes the internal context with a timeout and populates resource maps.
func NewContext(
	dbs map[string]*db.Pool,
	staticDir string,
	dataDir string,
	logger Logger,
	requestID string,
	userID string,
) *Context {
	// Default timeout of 30s, can be overridden by caller if needed
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	return &Context{
		ctx:         ctx,
		cancel:      cancel,
		startTime:   time.Now(),
		dbs:         dbs,
		staticDir:   staticDir,
		dataDir:     dataDir,
		logger:      logger,
		requestID:   requestID,
		userID:      userID,
		executionID: generateExecutionID(),
	}
}

// generateExecutionID creates a unique ID for this function execution.
// In production, use a UUID library.
func generateExecutionID() string {
	return time.Now().Format("20060102150405.000000") + "-" + time.Now().UnixNano()%1000000
}

// Cancel cancels the context and releases resources.
// This should be called when the function execution is complete or aborted.
func (c *Context) Cancel() {
	if c.cancel != nil {
		c.cancel()
	}
}

// Deadline returns the deadline set for the context.
func (c *Context) Deadline() (deadline time.Time, ok bool) {
	return c.ctx.Deadline()
}

// Done returns a channel that's closed when the context is canceled or times out.
func (c *Context) Done() <-chan struct{} {
	return c.ctx.Done()
}

// Err returns the error if the context is canceled or timed out.
func (c *Context) Err() error {
	return c.ctx.Err()
}

// Value returns the value associated with a key.
// Supports standard context keys plus custom SONG keys.
func (c *Context) Value(key interface{}) interface{} {
	switch k := key.(type) {
	case string:
		switch k {
		case "request_id":
			return c.requestID
		case "user_id":
			return c.userID
		case "execution_id":
			return c.executionID
		case "start_time":
			return c.startTime
		}
	}
	return c.ctx.Value(key)
}

// GetDB returns a database pool by name.
// Returns ErrDBNotFound if the pool is not configured for this function.
func (c *Context) GetDB(name string) (*db.Pool, error) {
	if pool, ok := c.dbs[name]; ok {
		return pool, nil
	}
	return nil, ErrDBNotFound
}

// GetDefaultDB returns the default database pool.
// Convenience method for accessing the primary database.
func (c *Context) GetDefaultDB() (*db.Pool, error) {
	return c.GetDB("default")
}

// GetAllDBs returns a copy of all available database pools.
// Note: The map itself is not copied, but the pools are references.
func (c *Context) GetAllDBs() map[string]*db.Pool {
	return c.dbs
}

// GetStaticDir returns the absolute path to the static files directory.
func (c *Context) GetStaticDir() string {
	return c.staticDir
}

// GetDataDir returns the absolute path to the data directory.
func (c *Context) GetDataDir() string {
	return c.dataDir
}

// GetLogger returns the logger instance.
func (c *Context) GetLogger() Logger {
	return c.logger
}

// GetRequestID returns the unique identifier for the HTTP request.
func (c *Context) GetRequestID() string {
	return c.requestID
}

// GetUserID returns the ID of the authenticated user (if any).
func (c *Context) GetUserID() string {
	return c.userID
}

// GetExecutionID returns the unique identifier for this specific function run.
func (c *Context) GetExecutionID() string {
	return c.executionID
}

// GetStartTime returns the time when the function execution started.
func (c *Context) GetStartTime() time.Time {
	return c.startTime
}

// Elapsed returns the time elapsed since the function started.
func (c *Context) Elapsed() time.Duration {
	return time.Since(c.startTime)
}

// IsCanceled checks if the context has been canceled or timed out.
func (c *Context) IsCanceled() bool {
	select {
	case <-c.ctx.Done():
		return true
	default:
		return false
	}
}

// GetElapsedSeconds returns the elapsed time in seconds as a float64.
// Useful for metrics and logging duration.
func (c *Context) GetElapsedSeconds() float64 {
	return c.Elapsed().Seconds()
}

// WithValue returns a new context derived from this one with an additional value.
// This is useful for passing request-scoped data down the call stack.
func (c *Context) WithValue(key, value interface{}) *Context {
	newCtx := context.WithValue(c.ctx, key, value)

	return &Context{
		ctx:         newCtx,
		cancel:      c.cancel, // Reuse the same cancel function
		startTime:   c.startTime,
		dbs:         c.dbs,
		staticDir:   c.staticDir,
		dataDir:     c.dataDir,
		logger:      c.logger,
		requestID:   c.requestID,
		userID:      c.userID,
		executionID: c.executionID,
	}
}
