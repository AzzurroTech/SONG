package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql" // MySQL driver
	_ "github.com/lib/pq"              // PostgreSQL driver
	_ "modernc.org/sqlite"             // SQLite driver
)

// Pool wraps a sql.DB connection pool with additional metadata
type Pool struct {
	*sql.DB
	Name     string
	Type     string
	Host     string
	Port     string
	User     string
	MaxConns int
	MinConns int
}

// PoolConfig holds configuration for a single connection pool
type PoolConfig struct {
	Name     string
	Type     string
	Host     string
	Port     string
	User     string
	Password string
	Database string
	SSLMode  string
	MaxConns int
	MinConns int
	Extra    map[string]string
}

// NewPool creates a new database connection pool
func NewPool(config PoolConfig) (*Pool, error) {
	dsn, err := buildDSN(config)
	if err != nil {
		return nil, fmt.Errorf("failed to build DSN: %w", err)
	}

	db, err := sql.Open(config.Type, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(config.MaxConns)
	db.SetMinOpenConns(config.MinConns)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(30 * time.Second)

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Pool{
		DB:       db,
		Name:     config.Name,
		Type:     config.Type,
		Host:     config.Host,
		Port:     config.Port,
		User:     config.User,
		MaxConns: config.MaxConns,
		MinConns: config.MinConns,
	}, nil
}

// buildDSN constructs a database connection string based on the database type
func buildDSN(config PoolConfig) (string, error) {
	switch config.Type {
	case "postgresql", "postgres":
		return buildPostgresDSN(config)
	case "mysql":
		return buildMySQLDSN(config)
	case "sqlite", "sqlite3":
		return buildSQLiteDSN(config)
	case "redis":
		// Redis handled separately (not sql.DB compatible)
		return "", fmt.Errorf("redis requires separate connection handling")
	default:
		return "", fmt.Errorf("unsupported database type: %s", config.Type)
	}
}

// buildPostgresDSN creates a PostgreSQL DSN
func buildPostgresDSN(config PoolConfig) (string, error) {
	sslMode := config.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		config.Host,
		config.Port,
		config.User,
		config.Password,
		config.Database,
		sslMode,
	)

	return dsn, nil
}

// buildMySQLDSN creates a MySQL DSN
func buildMySQLDSN(config PoolConfig) (string, error) {
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=Local",
		config.User,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
	)

	return dsn, nil
}

// buildSQLiteDSN creates a SQLite DSN (file path)
func buildSQLiteDSN(config PoolConfig) (string, error) {
	// For SQLite, the "database" field is actually the file path
	if config.Database == "" {
		return ":memory:", nil
	}
	return config.Database, nil
}

// Close closes the database connection pool
func (p *Pool) Close() error {
	if p.DB != nil {
		return p.DB.Close()
	}
	return nil
}

// Stats returns connection pool statistics
func (p *Pool) Stats() sql.DBStats {
	if p.DB != nil {
		return p.DB.Stats()
	}
	return sql.DBStats{}
}

// HealthCheck performs a basic health check on the database
func (p *Pool) HealthCheck() error {
	return p.DB.Ping()
}

// MaskDSN returns a DSN with the password masked for logging
func (p *Pool) MaskDSN() string {
	switch p.Type {
	case "postgresql", "postgres":
		return fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=%s",
			p.Host, p.Port, p.User, p.Database, p.SSLMode)
	case "mysql":
		return fmt.Sprintf("%s:@tcp(%s:%s)/%s",
			p.User, p.Host, p.Port, p.Database)
	case "sqlite", "sqlite3":
		return p.Database
	default:
		return "unknown"
	}
}
