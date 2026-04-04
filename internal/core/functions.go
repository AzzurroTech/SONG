package core

import (
	"encoding/base64"
	"errors"
	"time"
)

// FunctionStatus represents the state of a function
type FunctionStatus string

const (
	StatusPending  FunctionStatus = "pending"
	StatusBuilding FunctionStatus = "building"
	StatusActive   FunctionStatus = "active"
	StatusInactive FunctionStatus = "inactive"
	StatusFailed   FunctionStatus = "failed"
	StatusDeleted  FunctionStatus = "deleted"
)

// Function represents a deployed function
type Function struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	OwnerID        string         `json:"owner_id"`
	SourceCode     string         `json:"source_code"` // Base64 encoded
	BinaryPath     string         `json:"binary_path"`
	Status         FunctionStatus `json:"status"`
	DBConnections  []string       `json:"db_connections,omitempty"` // List of DB names this function can access
	Description    string         `json:"description,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	LastExecuted   *time.Time     `json:"last_executed,omitempty"`
	ExecutionCount int            `json:"execution_count"`
}

// FunctionsData represents the structure of functions.json
type FunctionsData struct {
	Version     string     `json:"version"`
	LastUpdated string     `json:"last_updated"`
	Functions   []Function `json:"functions"`
}

// FunctionManager handles function metadata operations
type FunctionManager struct {
	store *Store
}

// NewFunctionManager creates a new function manager
func NewFunctionManager(store *Store) *FunctionManager {
	return &FunctionManager{store: store}
}

// ErrFunctionNotFound is returned when a function cannot be found
var ErrFunctionNotFound = errors.New("function not found")

// ErrFunctionExists is returned when a function with the same name already exists
var ErrFunctionExists = errors.New("function already exists")

// GetAll returns all functions
func (fm *FunctionManager) GetAll() ([]Function, error) {
	var data FunctionsData
	if err := fm.store.Read("functions.json", &data); err != nil {
		return nil, err
	}
	return data.Functions, nil
}

// GetByID retrieves a function by ID
func (fm *FunctionManager) GetByID(id string) (*Function, error) {
	funcs, err := fm.GetAll()
	if err != nil {
		return nil, err
	}

	for _, f := range funcs {
		if f.ID == id {
			return &f, nil
		}
	}
	return nil, ErrFunctionNotFound
}

// GetByName retrieves a function by name
func (fm *FunctionManager) GetByName(name string) (*Function, error) {
	funcs, err := fm.GetAll()
	if err != nil {
		return nil, err
	}

	for _, f := range funcs {
		if f.Name == name {
			return &f, nil
		}
	}
	return nil, ErrFunctionNotFound
}

// Create registers a new function
func (fm *FunctionManager) Create(id, name, ownerID, sourceCode, description string, dbConnections []string) (*Function, error) {
	// Check if function name already exists
	existing, _ := fm.GetByName(name)
	if existing != nil {
		return nil, ErrFunctionExists
	}

	now := time.Now().UTC()
	function := Function{
		ID:             id,
		Name:           name,
		OwnerID:        ownerID,
		SourceCode:     sourceCode, // Should be base64 encoded before passing
		Status:         StatusPending,
		DBConnections:  dbConnections,
		Description:    description,
		CreatedAt:      now,
		UpdatedAt:      now,
		ExecutionCount: 0,
	}

	var data FunctionsData
	if err := fm.store.Read("functions.json", &data); err != nil {
		return nil, err
	}

	if data.Functions == nil {
		data.Functions = []Function{}
		data.Version = "1.0"
	}

	data.Functions = append(data.Functions, function)
	data.LastUpdated = now.Format(time.RFC3339)

	if err := fm.store.Write("functions.json", data); err != nil {
		return nil, err
	}

	return &function, nil
}

// Update modifies an existing function's metadata
func (fm *FunctionManager) Update(id string, updates map[string]interface{}) (*Function, error) {
	var data FunctionsData
	if err := fm.store.Read("functions.json", &data); err != nil {
		return nil, err
	}

	found := false
	for i, f := range data.Functions {
		if f.ID == id {
			found = true

			if name, ok := updates["name"].(string); ok {
				// Check for name collision if name is changing
				if name != f.Name {
					if _, err := fm.GetByName(name); err == nil {
						return nil, ErrFunctionExists
					}
				}
				f.Name = name
			}
			if status, ok := updates["status"].(FunctionStatus); ok {
				f.Status = status
			}
			if binaryPath, ok := updates["binary_path"].(string); ok {
				f.BinaryPath = binaryPath
			}
			if sourceCode, ok := updates["source_code"].(string); ok {
				f.SourceCode = sourceCode
			}
			if dbConns, ok := updates["db_connections"].([]string); ok {
				f.DBConnections = dbConns
			}
			if desc, ok := updates["description"].(string); ok {
				f.Description = desc
			}

			f.UpdatedAt = time.Now().UTC()
			data.Functions[i] = f
			break
		}
	}

	if !found {
		return nil, ErrFunctionNotFound
	}

	data.LastUpdated = time.Now().UTC().Format(time.RFC3339)
	if err := fm.store.Write("functions.json", data); err != nil {
		return nil, err
	}

	return fm.GetByID(id)
}

// Delete removes a function (soft delete by setting status)
func (fm *FunctionManager) Delete(id string) error {
	_, err := fm.Update(id, map[string]interface{}{
		"status": StatusDeleted,
	})
	return err
}

// Activate sets a function status to active
func (fm *FunctionManager) Activate(id string) error {
	_, err := fm.Update(id, map[string]interface{}{
		"status": StatusActive,
	})
	return err
}

// IncrementExecutionCount increments the execution counter
func (fm *FunctionManager) IncrementExecutionCount(id string) error {
	funcs, err := fm.GetAll()
	if err != nil {
		return err
	}

	for i, f := range funcs {
		if f.ID == id {
			f.ExecutionCount++
			now := time.Now().UTC()
			f.LastExecuted = &now
			funcs[i] = f
			break
		}
	}

	return fm.store.Write("functions.json", FunctionsData{
		Version:     "1.0",
		LastUpdated: time.Now().UTC().Format(time.RFC3339),
		Functions:   funcs,
	})
}

// EncodeSource encodes source code to base64
func EncodeSource(source string) string {
	return base64.StdEncoding.EncodeToString([]byte(source))
}

// DecodeSource decodes base64 source code
func DecodeSource(encoded string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
