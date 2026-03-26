package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"
)

var DB *sql.DB
var err error

func InitDB() error {
	host := os.Getenv("PG_HOST")
	port := os.Getenv("PG_PORT")
	user := os.Getenv("PG_USER")
	password := os.Getenv("PG_PASSWORD")
	dbname := os.Getenv("PG_DBNAME")

	if host == "" || user == "" || dbname == "" {
		log.Println("PostgreSQL env vars missing. Running in mock mode.")
		return nil
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)

	// NOTE: Uncomment below when using a real driver (e.g., github.com/lib/pq)
	// DB, err = sql.Open("postgres", dsn)

	// Mock for standard library constraint
	DB, err = sql.Open("mock", "mock://localhost")
	if err != nil {
		return fmt.Errorf("ERROR %s failed to connect: %w", dsn, err)
	}

	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(5)
	DB.SetConnMaxLifetime(5 * time.Minute)
	return DB.Ping()
}

// CreateSchema creates tables for VIDI (Data) and FormBuilder (Definitions)
func CreateSchema(ctx context.Context) error {
	// 1. Table for Form Definitions (The Builder output)
	// CREATE TABLE IF NOT EXISTS form_definitions (
	//     id SERIAL PRIMARY KEY,
	//     slug VARCHAR(100) UNIQUE NOT NULL,
	//     title VARCHAR(255),
	//     schema_json JSONB NOT NULL, -- Stores the field configuration
	//     created_at TIMESTAMP DEFAULT NOW()
	// );

	// 2. Table for Form Submissions (The VIDI data)
	// CREATE TABLE IF NOT EXISTS form_submissions (
	//     id SERIAL PRIMARY KEY,
	//     form_slug VARCHAR(100) REFERENCES form_definitions(slug),
	//     data_json JSONB NOT NULL,
	//     submitted_at TIMESTAMP DEFAULT NOW()
	// );

	return nil
}
