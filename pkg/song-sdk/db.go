package song

import (
	"database/sql"
	"fmt"
	"time"
)

// DB provides a safe wrapper around *sql.DB for user functions
type DB struct {
	db     *sql.DB
	name   string
	logger Logger
}

// NewDB creates a new DB wrapper
func NewDB(db *sql.DB, name string, logger Logger) *DB {
	return &DB{
		db:     db,
		name:   name,
		logger: logger,
	}
}

// Query executes a query and returns rows
func (d *DB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	rows, err := d.db.Query(query, args...)
	duration := time.Since(start)

	if err != nil {
		d.logger.Error("DB Query failed", "db", d.name, "query", truncate(query, 100), "error", err, "duration_ms", duration.Milliseconds())
		return nil, fmt.Errorf("query failed: %w", err)
	}

	d.logger.Debug("DB Query executed", "db", d.name, "query", truncate(query, 100), "duration_ms", duration.Milliseconds())
	return rows, nil
}

// QueryRow executes a query that returns a single row
func (d *DB) QueryRow(query string, args ...interface{}) *sql.Row {
	start := time.Now()
	row := d.db.QueryRow(query, args...)
	duration := time.Since(start)

	d.logger.Debug("DB QueryRow executed", "db", d.name, "query", truncate(query, 100), "duration_ms", duration.Milliseconds())
	return row
}

// Exec executes a query that doesn't return rows (INSERT, UPDATE, DELETE)
func (d *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()
	result, err := d.db.Exec(query, args...)
	duration := time.Since(start)

	if err != nil {
		d.logger.Error("DB Exec failed", "db", d.name, "query", truncate(query, 100), "error", err, "duration_ms", duration.Milliseconds())
		return nil, fmt.Errorf("exec failed: %w", err)
	}

	d.logger.Debug("DB Exec completed", "db", d.name, "query", truncate(query, 100), "duration_ms", duration.Milliseconds())
	return result, nil
}

// Transaction executes a function within a database transaction
// If the function returns an error, the transaction is rolled back
func (d *DB) Transaction(fn func(tx *sql.Tx) error) error {
	start := time.Now()
	tx, err := d.db.Begin()
	if err != nil {
		d.logger.Error("Failed to begin transaction", "db", d.name, "error", err)
		return fmt.Errorf("begin tx failed: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			d.logger.Critical("Transaction panicked", "db", d.name, "panic", p)
			panic(p)
		}
	}()

	err = fn(tx)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			d.logger.Error("Failed to rollback transaction", "db", d.name, "original_error", err, "rollback_error", rbErr)
			return fmt.Errorf("tx failed and rollback failed: %w (rollback: %v)", err, rbErr)
		}
		d.logger.Warn("Transaction rolled back", "db", d.name, "error", err, "duration_ms", time.Since(start).Milliseconds())
		return err
	}

	if err := tx.Commit(); err != nil {
		d.logger.Error("Failed to commit transaction", "db", d.name, "error", err)
		return fmt.Errorf("commit failed: %w", err)
	}

	d.logger.Info("Transaction committed", "db", d.name, "duration_ms", time.Since(start).Milliseconds())
	return nil
}

// Stats returns database connection statistics
func (d *DB) Stats() sql.DBStats {
	return d.db.Stats()
}

// Ping checks if the database is reachable
func (d *DB) Ping() error {
	return d.db.Ping()
}

// Name returns the database name
func (d *DB) Name() string {
	return d.name
}

// Truncate helper to prevent log spam
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
