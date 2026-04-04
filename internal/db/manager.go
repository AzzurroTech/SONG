package db

import (
	"fmt"
	"sync"

	"github.com/azzurrotech/song/internal/config"
)

// Manager handles multiple database connections
type Manager struct {
	mu     sync.RWMutex
	pools  map[string]*Pool
	config *config.Config
}

// NewManager creates a new database manager
func NewManager(cfg *config.Config) (*Manager, error) {
	manager := &Manager{
		pools:  make(map[string]*Pool),
		config: cfg,
	}

	// Initialize default database if enabled
	if cfg.DBEnabled {
		defaultPool, err := NewPool(PoolConfig{
			Name:     "default",
			Type:     cfg.DefaultDB.Type,
			Host:     cfg.DefaultDB.Host,
			Port:     cfg.DefaultDB.Port,
			User:     cfg.DefaultDB.User,
			Password: cfg.DefaultDB.Password,
			Database: cfg.DefaultDB.Name,
			SSLMode:  cfg.DefaultDB.SSLMode,
			MaxConns: cfg.DefaultDB.MaxConns,
			MinConns: cfg.DefaultDB.MinConns,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to initialize default database: %w", err)
		}
		manager.pools["default"] = defaultPool
	}

	// Initialize additional databases
	for name, dbCfg := range cfg.AdditionalDBs {
		pool, err := NewPool(PoolConfig{
			Name:     name,
			Type:     dbCfg.Type,
			Host:     dbCfg.Host,
			Port:     dbCfg.Port,
			User:     dbCfg.User,
			Password: dbCfg.Password,
			Database: dbCfg.Name,
			SSLMode:  dbCfg.SSLMode,
			MaxConns: dbCfg.MaxConns,
			MinConns: dbCfg.MinConns,
			Extra:    dbCfg.Extra,
		})
		if err != nil {
			// Log warning but continue - don't fail startup for optional DBs
			fmt.Printf("Warning: Failed to connect to additional database '%s': %v\n", name, err)
			continue
		}
		manager.pools[name] = pool
	}

	return manager, nil
}

// GetPool retrieves a database pool by name
func (m *Manager) GetPool(name string) (*Pool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pool, exists := m.pools[name]
	if !exists {
		return nil, fmt.Errorf("database pool '%s' not found", name)
	}
	return pool, nil
}

// GetDefault returns the default database pool
func (m *Manager) GetDefault() (*Pool, error) {
	return m.GetPool("default")
}

// ListPools returns the names of all available pools
func (m *Manager) ListPools() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.pools))
	for name := range m.pools {
		names = append(names, name)
	}
	return names
}

// AddPool dynamically adds a new database connection
func (m *Manager) AddPool(name string, cfg PoolConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.pools[name]; exists {
		return fmt.Errorf("database pool '%s' already exists", name)
	}

	pool, err := NewPool(cfg)
	if err != nil {
		return fmt.Errorf("failed to create pool '%s': %w", name, err)
	}

	m.pools[name] = pool
	return nil
}

// RemovePool removes a database connection by name
func (m *Manager) RemovePool(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	pool, exists := m.pools[name]
	if !exists {
		return fmt.Errorf("database pool '%s' not found", name)
	}

	// Don't allow removing the default pool
	if name == "default" {
		return fmt.Errorf("cannot remove the default database pool")
	}

	if err := pool.Close(); err != nil {
		return fmt.Errorf("failed to close pool '%s': %w", name, err)
	}

	delete(m.pools, name)
	return nil
}

// HealthCheckAll checks the health of all database connections
func (m *Manager) HealthCheckAll() map[string]error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make(map[string]error)
	for name, pool := range m.pools {
		results[name] = pool.HealthCheck()
	}
	return results
}

// StatsAll returns statistics for all pools
func (m *Manager) StatsAll() map[string]PoolStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]PoolStats)
	for name, pool := range m.pools {
		stats[name] = PoolStats{
			Name:              name,
			Type:              pool.Type,
			OpenConns:         pool.Stats().OpenConnections,
			IdleConns:         pool.Stats().Idle,
			InUseConns:        pool.Stats().InUse,
			WaitCount:         pool.Stats().WaitCount,
			WaitDuration:      pool.Stats().WaitDuration,
			MaxIdleClosed:     pool.Stats().MaxIdleClosed,
			MaxLifetimeClosed: pool.Stats().MaxLifetimeClosed,
		}
	}
	return stats
}

// PoolStats represents statistics for a single pool
type PoolStats struct {
	Name              string
	Type              string
	OpenConns         int
	IdleConns         int
	InUseConns        int
	WaitCount         int64
	WaitDuration      string
	MaxIdleClosed     int64
	MaxLifetimeClosed int64
}

// Close closes all database connections
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var lastErr error
	for name, pool := range m.pools {
		if err := pool.Close(); err != nil {
			lastErr = fmt.Errorf("failed to close pool '%s': %w", name, err)
		}
		delete(m.pools, name)
	}
	return lastErr
}

// IsEnabled returns whether any databases are configured
func (m *Manager) IsEnabled() bool {
	return m.config.DBEnabled
}

// HasPool checks if a specific pool exists
func (m *Manager) HasPool(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.pools[name]
	return exists
}
