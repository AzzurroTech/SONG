package song

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// LogLevel represents the severity of a log entry
type LogLevel string

const (
	LogLevelDebug    LogLevel = "DEBUG"
	LogLevelInfo     LogLevel = "INFO"
	LogLevelWarn     LogLevel = "WARN"
	LogLevelError    LogLevel = "ERROR"
	LogLevelCritical LogLevel = "CRITICAL"
)

// LoggerImpl is the concrete implementation of the Logger interface
type LoggerImpl struct {
	requestID   string
	executionID string
	userID      string
	mu          sync.Mutex
}

// NewLogger creates a new logger instance for a function execution
func NewLogger(requestID, executionID, userID string) *LoggerImpl {
	return &LoggerImpl{
		requestID:   requestID,
		executionID: executionID,
		userID:      userID,
	}
}

// logEntry represents a structured log entry
type logEntry struct {
	Timestamp   time.Time              `json:"timestamp"`
	Level       LogLevel               `json:"level"`
	RequestID   string                 `json:"request_id"`
	ExecutionID string                 `json:"execution_id"`
	UserID      string                 `json:"user_id,omitempty"`
	Message     string                 `json:"message"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// log writes a log entry to the system logger
func (l *LoggerImpl) log(level LogLevel, msg string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Format message
	var formattedMsg string
	if len(args) > 0 {
		formattedMsg = fmt.Sprintf(msg, args...)
	} else {
		formattedMsg = msg
	}

	// Create log entry
	entry := logEntry{
		Timestamp:   time.Now().UTC(),
		Level:       level,
		RequestID:   l.requestID,
		ExecutionID: l.executionID,
		UserID:      l.userID,
		Message:     formattedMsg,
	}

	// Extract metadata from args if provided as key-value pairs
	if len(args)%2 == 0 && len(args) > 0 {
		metadata := make(map[string]interface{})
		for i := 0; i < len(args); i += 2 {
			if key, ok := args[i].(string); ok {
				metadata[key] = args[i+1]
			}
		}
		if len(metadata) > 0 {
			entry.Metadata = metadata
		}
	}

	// Output to stdout (captured by the runtime)
	// In a real implementation, this would write to the core LogManager
	// For now, we use standard log with a structured format
	log.Printf("[%s] %s | REQ:%s | EXEC:%s | MSG: %s",
		level,
		entry.Timestamp.Format(time.RFC3339),
		entry.RequestID,
		entry.ExecutionID,
		formattedMsg)

	// If metadata exists, log it separately
	if len(entry.Metadata) > 0 {
		log.Printf("METADATA: %v", entry.Metadata)
	}
}

// Debug logs a debug message
func (l *LoggerImpl) Debug(msg string, args ...interface{}) {
	l.log(LogLevelDebug, msg, args...)
}

// Info logs an informational message
func (l *LoggerImpl) Info(msg string, args ...interface{}) {
	l.log(LogLevelInfo, msg, args...)
}

// Warn logs a warning message
func (l *LoggerImpl) Warn(msg string, args ...interface{}) {
	l.log(LogLevelWarn, msg, args...)
}

// Error logs an error message
func (l *LoggerImpl) Error(msg string, args ...interface{}) {
	l.log(LogLevelError, msg, args...)
}

// Critical logs a critical error message
func (l *LoggerImpl) Critical(msg string, args ...interface{}) {
	l.log(LogLevelCritical, msg, args...)
}

// With returns a new logger with additional context
func (l *LoggerImpl) With(key string, value interface{}) Logger {
	// In a full implementation, this would create a child logger with context
	// For now, we just return the same logger as context is per-execution
	return l
}
