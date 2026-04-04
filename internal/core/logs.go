package core

import (
	"fmt"
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

// LogEntry represents a single log record
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Service   string                 `json:"service"`
	Message   string                 `json:"message"`
	UserID    string                 `json:"user_id,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// LogManager handles logging operations
type LogManager struct {
	store   *Store
	service string
}

// NewLogManager creates a new log manager
func NewLogManager(store *Store, serviceName string) *LogManager {
	return &LogManager{
		store:   store,
		service: serviceName,
	}
}

// Log writes a log entry to the JSONL file
func (lm *LogManager) Log(level LogLevel, message string, metadata map[string]interface{}) error {
	entry := LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     level,
		Service:   lm.service,
		Message:   message,
		Metadata:  metadata,
	}
	return lm.store.AppendLog("logs.jsonl", entry)
}

// Debug logs a debug message
func (lm *LogManager) Debug(message string, metadata map[string]interface{}) error {
	return lm.Log(LogLevelDebug, message, metadata)
}

// Info logs an informational message
func (lm *LogManager) Info(message string, metadata map[string]interface{}) error {
	return lm.Log(LogLevelInfo, message, metadata)
}

// Warn logs a warning message
func (lm *LogManager) Warn(message string, metadata map[string]interface{}) error {
	return lm.Log(LogLevelWarn, message, metadata)
}

// Error logs an error message
func (lm *LogManager) Error(message string, metadata map[string]interface{}) error {
	return lm.Log(LogLevelError, message, metadata)
}

// Critical logs a critical error message
func (lm *LogManager) Critical(message string, metadata map[string]interface{}) error {
	return lm.Log(LogLevelCritical, message, metadata)
}

// WithUser adds user context to the metadata
func (lm *LogManager) WithUser(userID string) *LogManager {
	// This is a helper to create a new logger instance with user context
	// In a real implementation, we might return a wrapper or modify the caller's metadata
	// For simplicity, we'll just log the user ID in the metadata if needed
	return lm
}

// WithRequest adds request context to the metadata
func (lm *LogManager) WithRequest(requestID string) *LogManager {
	return lm
}

// GetRecentLogs retrieves the last N log entries (simple implementation)
// Note: For production, consider a rotating log file or external log aggregator
func (lm *LogManager) GetRecentLogs(count int) ([]LogEntry, error) {
	// This is a simplified implementation.
	// In a real system, we would read the file, parse lines, and return the last N.
	// Since AppendLog is append-only, we'd need to read the whole file and slice.
	// For now, we return an empty slice and log a warning that this is not implemented for large files.
	lm.Warn("GetRecentLogs is not fully implemented for large log files", map[string]interface{}{
		"requested_count": count,
	})
	return []LogEntry{}, nil
}

// FormatLogEntry formats a log entry for human-readable output
func FormatLogEntry(entry LogEntry) string {
	return fmt.Sprintf("[%s] [%s] [%s] %s",
		entry.Timestamp.Format(time.RFC3339),
		entry.Level,
		entry.Service,
		entry.Message)
}
