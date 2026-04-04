package db

import (
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/source/iofs"
)

// MigrationsManager handles database schema migrations
type MigrationsManager struct {
	pool          *Pool
	migrationsDir string
}

// NewMigrationsManager creates a new migration manager
func NewMigrationsManager(pool *Pool, migrationsDir string) *MigrationsManager {
	return &MigrationsManager{
		pool:          pool,
		migrationsDir: migrationsDir,
	}
}

// Run executes all pending migrations
func (mm *MigrationsManager) Run() error {
	if mm.pool == nil {
		return fmt.Errorf("no database pool configured")
	}

	// Ensure migrations directory exists
	if _, err := os.Stat(mm.migrationsDir); os.IsNotExist(err) {
		fmt.Printf("Migrations directory not found: %s. Skipping migrations.\n", mm.migrationsDir)
		return nil
	}

	// Create a source for migrations
	sourceDriver, err := NewMigrationSource(mm.migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	// Create the migrate instance
	m, err := migrate.NewWithSourceInstance(
		"iofs",
		sourceDriver,
		mm.pool.Type,
		mm.pool.MaskDSN(), // Use masked DSN for logging, but full DSN for connection
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	// Run migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	if err == migrate.ErrNoChange {
		fmt.Printf("No new migrations found for database: %s\n", mm.pool.Name)
	} else {
		fmt.Printf("Migrations completed successfully for database: %s\n", mm.pool.Name)
	}

	return nil
}

// NewMigrationSource creates an io/fs source for migrations
func NewMigrationSource(dir string) (fs.FS, error) {
	return os.DirFS(dir), nil
}

// GetPendingMigrations lists migrations that haven't been applied yet
func (mm *MigrationsManager) GetPendingMigrations() ([]string, error) {
	var pending []string

	// Read all .sql files in the migrations directory
	files, err := os.ReadDir(mm.migrationsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".up.sql") {
			continue
		}

		// Extract version number from filename (e.g., 001_create_users.up.sql -> 001)
		name := file.Name()
		parts := strings.SplitN(name, "_", 2)
		if len(parts) < 1 {
			continue
		}

		version := strings.TrimPrefix(parts[0], "0")
		if version == "" {
			version = parts[0]
		}

		// Check if migration is already applied
		applied, err := mm.isMigrationApplied(version)
		if err != nil {
			return nil, fmt.Errorf("failed to check migration status for %s: %w", name, err)
		}

		if !applied {
			pending = append(pending, name)
		}
	}

	// Sort by version number
	sort.Strings(pending)
	return pending, nil
}

// isMigrationApplied checks if a specific migration version has been applied
func (mm *MigrationsManager) isMigrationApplied(version string) (bool, error) {
	// Check if the _migrations table exists
	var exists bool
	err := mm.pool.DB.QueryRow(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_name = '_migrations'
		)
	`).Scan(&exists)

	if err != nil {
		// Table doesn't exist yet, so no migrations applied
		return false, nil
	}

	// Check if this specific version is in the table
	var count int
	err = mm.pool.DB.QueryRow(`
		SELECT COUNT(*) FROM _migrations WHERE version = $1
	`, version).Scan(&count)

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// RecordMigration manually records a migration as applied (for manual schema changes)
func (mm *MigrationsManager) RecordMigration(version string) error {
	// Ensure _migrations table exists
	_, err := mm.pool.Exec(`
		CREATE TABLE IF NOT EXISTS _migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Insert the version
	_, err = mm.pool.Exec(`
		INSERT INTO _migrations (version, applied_at) 
		VALUES ($1, $2)
		ON CONFLICT (version) DO NOTHING
	`, version, time.Now())

	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return nil
}

// Rollback rolls back the last applied migration
func (mm *MigrationsManager) Rollback(steps int) error {
	if mm.pool == nil {
		return fmt.Errorf("no database pool configured")
	}

	sourceDriver, err := NewMigrationSource(mm.migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance(
		"iofs",
		sourceDriver,
		mm.pool.Type,
		mm.pool.MaskDSN(),
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err := m.Steps(-steps); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to rollback migrations: %w", err)
	}

	fmt.Printf("Rolled back %d migrations for database: %s\n", steps, mm.pool.Name)
	return nil
}

// Force sets the database version to a specific version (dangerous)
func (mm *MigrationsManager) Force(version int) error {
	if mm.pool == nil {
		return fmt.Errorf("no database pool configured")
	}

	sourceDriver, err := NewMigrationSource(mm.migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance(
		"iofs",
		sourceDriver,
		mm.pool.Type,
		mm.pool.MaskDSN(),
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err := m.Force(version); err != nil {
		return fmt.Errorf("failed to force migration version: %w", err)
	}

	fmt.Printf("Forced database version to %d for: %s\n", version, mm.pool.Name)
	return nil
}

// CurrentVersion returns the current version of the database
func (mm *MigrationsManager) CurrentVersion() (int, error) {
	if mm.pool == nil {
		return 0, fmt.Errorf("no database pool configured")
	}

	sourceDriver, err := NewMigrationSource(mm.migrationsDir)
	if err != nil {
		return 0, fmt.Errorf("failed to create migration source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance(
		"iofs",
		sourceDriver,
		mm.pool.Type,
		mm.pool.MaskDSN(),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	version, _, err := m.Version()
	if err != nil {
		return 0, err
	}

	return int(version), nil
}
