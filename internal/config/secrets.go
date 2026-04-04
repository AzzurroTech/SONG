package config

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
)

// SecretMasker provides utilities for handling sensitive data
type SecretMasker struct{}

// NewSecretMasker creates a new instance
func NewSecretMasker() *SecretMasker {
	return &SecretMasker{}
}

// Mask replaces sensitive parts of a string with asterisks
// Useful for logging connection strings or tokens
func (sm *SecretMasker) Mask(secret string, visibleChars int) string {
	if len(secret) <= visibleChars {
		return strings.Repeat("*", len(secret))
	}
	return secret[:visibleChars] + strings.Repeat("*", len(secret)-visibleChars)
}

// MaskConnectionString masks the password portion of a database connection string
func (sm *SecretMasker) MaskConnectionString(connString string) string {
	// Simple heuristic: find "password=..." and mask it
	parts := strings.Split(connString, "password=")
	if len(parts) < 2 {
		return connString
	}

	// Find the end of the password (space, semicolon, or end of string)
	endIdx := strings.IndexAny(parts[1], " ;")
	if endIdx == -1 {
		endIdx = len(parts[1])
	}

	maskedPassword := strings.Repeat("*", endIdx)
	return parts[0] + "password=" + maskedPassword + parts[1][endIdx:]
}

// GenerateSecureSecret generates a cryptographically secure random string
// Useful for creating initial JWT secrets or API keys
func (sm *SecretMasker) GenerateSecureSecret(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// SanitizeLogMessage removes or masks known sensitive patterns from log messages
func (sm *SecretMasker) SanitizeLogMessage(msg string) string {
	sensitivePatterns := []string{
		"password=", "Password=", "PASSWORD=",
		"secret=", "Secret=", "SECRET=",
		"token=", "Token=", "TOKEN=",
		"api_key=", "ApiKey=", "API_KEY=",
		"authorization:", "Authorization:",
	}

	sanitized := msg
	for _, pattern := range sensitivePatterns {
		// Replace value after pattern with ***
		idx := strings.Index(sanitized, pattern)
		if idx != -1 {
			// Find the end of the value (space or end of string)
			start := idx + len(pattern)
			end := start
			for end < len(sanitized) && sanitized[end] != ' ' && sanitized[end] != '\n' && sanitized[end] != '&' {
				end++
			}
			if end > start {
				sanitized = sanitized[:start] + "***" + sanitized[end:]
			}
		}
	}
	return sanitized
}
