package context

import (
	"sync"
)

// Manager holds the global context for the SONG server.
type Manager struct {
	mu       sync.RWMutex
	config   map[string]string
	sessions map[string]interface{}
}

var GlobalManager = &Manager{
	config:   make(map[string]string),
	sessions: make(map[string]interface{}),
}

// SetConfig sets a configuration value (VICI integration point).
func (m *Manager) SetConfig(key, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config[key] = value
}

// GetConfig retrieves a configuration value.
func (m *Manager) GetConfig(key string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config[key]
}

// StoreSession stores user context.
func (m *Manager) StoreSession(id string, data interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[id] = data
}

// GetSession retrieves user context.
func (m *Manager) GetSession(id string) interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions[id]
}
