package core

import (
	"errors"
	"strings"
	"time"
)

// Session represents an active user session
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Token     string    `json:"token"` // JWT token
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	IsActive  bool      `json:"is_active"`
}

// SessionsData represents the structure of sessions.json
type SessionsData struct {
	Version     string    `json:"version"`
	LastUpdated string    `json:"last_updated"`
	Sessions    []Session `json:"sessions"`
}

// SessionManager handles session operations
type SessionManager struct {
	store *Store
}

// NewSessionManager creates a new session manager
func NewSessionManager(store *Store) *SessionManager {
	return &SessionManager{store: store}
}

// ErrSessionNotFound is returned when a session cannot be found
var ErrSessionNotFound = errors.New("session not found")

// ErrSessionExpired is returned when a session has expired
var ErrSessionExpired = errors.New("session expired")

// ErrSessionInactive is returned when a session has been revoked
var ErrSessionInactive = errors.New("session inactive")

// Create creates a new session
func (sm *SessionManager) Create(id, userID, token, ip, userAgent string, duration time.Duration) (*Session, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(duration)

	session := Session{
		ID:        id,
		UserID:    userID,
		Token:     token,
		ExpiresAt: expiresAt,
		CreatedAt: now,
		IPAddress: ip,
		UserAgent: userAgent,
		IsActive:  true,
	}

	var data SessionsData
	if err := sm.store.Read("sessions.json", &data); err != nil {
		return nil, err
	}

	if data.Sessions == nil {
		data.Sessions = []Session{}
		data.Version = "1.0"
	}

	// Remove expired/inactive sessions for this user before adding new one
	cleanedSessions := make([]Session, 0, len(data.Sessions))
	for _, s := range data.Sessions {
		if s.UserID != userID || (s.IsActive && !s.ExpiresAt.Before(now)) {
			cleanedSessions = append(cleanedSessions, s)
		}
	}
	data.Sessions = cleanedSessions

	// Add new session
	data.Sessions = append(data.Sessions, session)
	data.LastUpdated = now.Format(time.RFC3339)

	if err := sm.store.Write("sessions.json", data); err != nil {
		return nil, err
	}

	return &session, nil
}

// GetByToken retrieves a session by its JWT token
func (sm *SessionManager) GetByToken(token string) (*Session, error) {
	var data SessionsData
	if err := sm.store.Read("sessions.json", &data); err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	for _, s := range data.Sessions {
		if s.Token == token {
			if !s.IsActive {
				return nil, ErrSessionInactive
			}
			if s.ExpiresAt.Before(now) {
				// Mark as expired but return error
				return nil, ErrSessionExpired
			}
			return &s, nil
		}
	}

	return nil, ErrSessionNotFound
}

// GetUserSessions returns all active sessions for a user
func (sm *SessionManager) GetUserSessions(userID string) ([]Session, error) {
	var data SessionsData
	if err := sm.store.Read("sessions.json", &data); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	activeSessions := make([]Session, 0)

	for _, s := range data.Sessions {
		if s.UserID == userID && s.IsActive && !s.ExpiresAt.Before(now) {
			activeSessions = append(activeSessions, s)
		}
	}

	return activeSessions, nil
}

// Revoke marks a session as inactive
func (sm *SessionManager) Revoke(token string) error {
	var data SessionsData
	if err := sm.store.Read("sessions.json", &data); err != nil {
		return err
	}

	found := false
	for i, s := range data.Sessions {
		if s.Token == token {
			data.Sessions[i].IsActive = false
			found = true
			break
		}
	}

	if !found {
		return ErrSessionNotFound
	}

	data.LastUpdated = time.Now().UTC().Format(time.RFC3339)
	return sm.store.Write("sessions.json", data)
}

// RevokeAll revokes all sessions for a specific user
func (sm *SessionManager) RevokeAll(userID string) error {
	var data SessionsData
	if err := sm.store.Read("sessions.json", &data); err != nil {
		return err
	}

	found := false
	for i, s := range data.Sessions {
		if s.UserID == userID {
			data.Sessions[i].IsActive = false
			found = true
		}
	}

	if !found {
		return ErrSessionNotFound
	}

	data.LastUpdated = time.Now().UTC().Format(time.RFC3339)
	return sm.store.Write("sessions.json", data)
}

// Cleanup removes expired and inactive sessions
func (sm *SessionManager) Cleanup() error {
	var data SessionsData
	if err := sm.store.Read("sessions.json", &data); err != nil {
		return err
	}

	now := time.Now().UTC()
	cleanedSessions := make([]Session, 0)

	for _, s := range data.Sessions {
		// Keep only active and non-expired sessions
		if s.IsActive && !s.ExpiresAt.Before(now) {
			cleanedSessions = append(cleanedSessions, s)
		}
	}

	// Only write if changes were made
	if len(cleanedSessions) != len(data.Sessions) {
		data.Sessions = cleanedSessions
		data.LastUpdated = now.Format(time.RFC3339)
		return sm.store.Write("sessions.json", data)
	}

	return nil
}

// Count returns the number of active sessions
func (sm *SessionManager) Count() (int, error) {
	sessions, err := sm.GetUserSessions("") // Get all, filter in logic if needed
	if err != nil {
		return 0, err
	}

	// Filter active ones
	now := time.Now().UTC()
	count := 0
	for _, s := range sessions {
		if s.IsActive && !s.ExpiresAt.Before(now) {
			count++
		}
	}
	return count, nil
}

// ExtractToken extracts the token from an Authorization header
func ExtractToken(authHeader string) string {
	if authHeader == "" {
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}
	return parts[1]
}
