package faas

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/azzurrotech/song/internal/core"
)

// Registry manages the lifecycle and state of deployed functions
type Registry struct {
	mu          sync.RWMutex
	functions   map[string]*FunctionEntry
	coreStore   *core.Store
	functionMgr *core.FunctionManager
	binaryDir   string
	runtime     *Runtime
}

// FunctionEntry represents a loaded function in the registry
type FunctionEntry struct {
	ID           string
	Name         string
	BinaryPath   string
	Status       core.FunctionStatus
	LastExecuted time.Time
	ExecCount    int64
	LoadedAt     time.Time
	Health       string // "healthy", "unhealthy", "unknown"
}

// NewRegistry creates a new function registry
func NewRegistry(store *core.Store, binaryDir string, runtime *Runtime) *Registry {
	r := &Registry{
		functions:   make(map[string]*FunctionEntry),
		coreStore:   store,
		functionMgr: core.NewFunctionManager(store),
		binaryDir:   binaryDir,
		runtime:     runtime,
	}

	// Load existing functions from core store
	r.loadFromStore()

	return r
}

// loadFromStore populates the registry from the JSON core store
func (reg *Registry) loadFromStore() {
	funcs, err := reg.functionMgr.GetAll()
	if err != nil {
		fmt.Printf("Warning: Failed to load functions from store: %v\n", err)
		return
	}

	reg.mu.Lock()
	defer reg.mu.Unlock()

	for _, f := range funcs {
		if f.Status == core.StatusDeleted {
			continue
		}

		// Verify binary exists
		binaryPath := filepath.Join(reg.binaryDir, fmt.Sprintf("%s.bin", f.ID))
		if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
			// Binary missing, mark as failed
			f.Status = core.StatusFailed
			reg.functionMgr.Update(f.ID, map[string]interface{}{"status": core.StatusFailed})
			continue
		}

		reg.functions[f.ID] = &FunctionEntry{
			ID:           f.ID,
			Name:         f.Name,
			BinaryPath:   binaryPath,
			Status:       f.Status,
			LastExecuted: time.Time{},
			ExecCount:    int64(f.ExecutionCount),
			LoadedAt:     time.Now(),
			Health:       "unknown",
		}
	}
}

// Register adds a new function to the registry
func (reg *Registry) Register(id, name, binaryPath string, status core.FunctionStatus) error {
	reg.mu.Lock()
	defer reg.mu.Unlock()

	if _, exists := reg.functions[id]; exists {
		return fmt.Errorf("function %s already registered", id)
	}

	reg.functions[id] = &FunctionEntry{
		ID:         id,
		Name:       name,
		BinaryPath: binaryPath,
		Status:     status,
		LoadedAt:   time.Now(),
		Health:     "unknown",
	}

	return nil
}

// Unregister removes a function from the registry
func (reg *Registry) Unregister(id string) error {
	reg.mu.Lock()
	defer reg.mu.Unlock()

	if _, exists := reg.functions[id]; !exists {
		return fmt.Errorf("function %s not found", id)
	}

	delete(reg.functions, id)

	// Also update core store
	return reg.functionMgr.Delete(id)
}

// Get retrieves a function entry by ID
func (reg *Registry) Get(id string) (*FunctionEntry, error) {
	reg.mu.RLock()
	defer reg.mu.RUnlock()

	entry, exists := reg.functions[id]
	if !exists {
		return nil, fmt.Errorf("function %s not found", id)
	}

	if entry.Status != core.StatusActive {
		return nil, fmt.Errorf("function %s is not active (status: %s)", id, entry.Status)
	}

	return entry, nil
}

// GetByName retrieves a function entry by name
func (reg *Registry) GetByName(name string) (*FunctionEntry, error) {
	reg.mu.RLock()
	defer reg.mu.RUnlock()

	for _, entry := range reg.functions {
		if entry.Name == name && entry.Status == core.StatusActive {
			return entry, nil
		}
	}

	return nil, fmt.Errorf("function with name %s not found", name)
}

// List returns all active functions
func (reg *Registry) List() []*FunctionEntry {
	reg.mu.RLock()
	defer reg.mu.RUnlock()

	var list []*FunctionEntry
	for _, entry := range reg.functions {
		if entry.Status == core.StatusActive {
			list = append(list, entry)
		}
	}
	return list
}

// UpdateStatus updates the status of a function
func (reg *Registry) UpdateStatus(id string, status core.FunctionStatus) error {
	reg.mu.Lock()
	defer reg.mu.Unlock()

	entry, exists := reg.functions[id]
	if !exists {
		return fmt.Errorf("function %s not found", id)
	}

	entry.Status = status
	if status == core.StatusActive {
		entry.Health = "healthy"
	} else {
		entry.Health = "unhealthy"
	}

	// Sync with core store
	return reg.functionMgr.Update(id, map[string]interface{}{"status": status})
}

// IncrementExecCount increments the execution counter for a function
func (reg *Registry) IncrementExecCount(id string) error {
	reg.mu.Lock()
	defer reg.mu.Unlock()

	entry, exists := reg.functions[id]
	if !exists {
		return fmt.Errorf("function %s not found", id)
	}

	entry.ExecCount++
	entry.LastExecuted = time.Now()

	// Sync with core store
	return reg.functionMgr.IncrementExecutionCount(id)
}

// HealthCheck performs a basic health check on all active functions
// (In a real system, this might involve sending a test request)
func (reg *Registry) HealthCheck() map[string]string {
	reg.mu.RLock()
	defer reg.mu.RUnlock()

	results := make(map[string]string)
	for id, entry := range reg.functions {
		if entry.Status != core.StatusActive {
			results[id] = "inactive"
			continue
		}

		// Check if binary exists
		if _, err := os.Stat(entry.BinaryPath); os.IsNotExist(err) {
			results[id] = "binary_missing"
			entry.Health = "unhealthy"
			continue
		}

		results[id] = "healthy"
	}
	return results
}

// GetStats returns statistics for all functions
func (reg *Registry) GetStats() map[string]FunctionStats {
	reg.mu.RLock()
	defer reg.mu.RUnlock()

	stats := make(map[string]FunctionStats)
	for id, entry := range reg.functions {
		stats[id] = FunctionStats{
			Name:         entry.Name,
			Status:       entry.Status,
			ExecCount:    entry.ExecCount,
			LastExecuted: entry.LastExecuted,
			LoadedAt:     entry.LoadedAt,
			Health:       entry.Health,
		}
	}
	return stats
}

// FunctionStats holds read-only statistics for a function
type FunctionStats struct {
	Name         string
	Status       core.FunctionStatus
	ExecCount    int64
	LastExecuted time.Time
	LoadedAt     time.Time
	Health       string
}
