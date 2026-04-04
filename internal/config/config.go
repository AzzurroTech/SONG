package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the SONG application
type Config struct {
	// Server
	Host string
	Port string
	Env  string // development, production, test

	// Core Storage
	CoreDir         string
	BackupEnabled   bool
	BackupInterval  time.Duration
	FileLockTimeout time.Duration

	// Auth
	JWTSecret        string
	OIDCEnabled      bool
	OIDCIssuer       string
	OIDCClientID     string
	OIDCClientSecret string

	// Paths
	StaticDir    string
	TemplateDir  string
	FunctionsDir string
	DataDir      string

	// Security
	FunctionTimeout time.Duration
	MaxMemoryMB     int64
	MaxCPUPercent   int

	// Database
	DBEnabled         bool
	DefaultDB         DBConfig
	AdditionalDBs     map[string]DBConfig
	AdditionalDBNames []string
}

// DBConfig holds configuration for a single database connection
type DBConfig struct {
	Type     string // postgresql, mysql, sqlite, redis, mongo
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string // disable, require, verify-full
	MaxConns int
	MinConns int
	Extra    map[string]string // For Redis DB index, Mongo URI, etc.
}

// Load reads the .env file and populates the Config struct
func Load() (*Config, error) {
	// Attempt to load .env file (ignore error if not found in prod)
	_ = godotenv.Load()

	cfg := &Config{
		Host:         getEnv("SONG_HOST", "0.0.0.0"),
		Port:         getEnv("SONG_PORT", "8080"),
		Env:          getEnv("SONG_ENV", "development"),
		CoreDir:      getEnv("CORE_DIR", "./core"),
		StaticDir:    getEnv("STATIC_DIR", "./static"),
		TemplateDir:  getEnv("TEMPLATE_DIR", "./templates"),
		FunctionsDir: getEnv("FUNCTIONS_DIR", "./functions"),
		DataDir:      getEnv("DATA_DIR", "./data"),
		JWTSecret:    getEnv("JWT_SECRET", ""),

		// Boolean flags
		BackupEnabled: getBoolEnv("CORE_BACKUP_ENABLED", true),
		OIDCEnabled:   getBoolEnv("OIDC_ENABLED", false),
		DBEnabled:     getBoolEnv("DB_ENABLED", false),

		// Durations
		BackupInterval:  getDurationEnv("CORE_BACKUP_INTERVAL", 24*time.Hour),
		FileLockTimeout: getDurationEnv("CORE_FILE_LOCK_TIMEOUT", 5*time.Second),
		FunctionTimeout: getDurationEnv("MAX_FUNCTION_TIMEOUT", 30*time.Second),

		// Limits
		MaxMemoryMB:   getInt64Env("MAX_MEMORY_MB", 256),
		MaxCPUPercent: getIntEnv("MAX_CPU_PERCENT", 50),
	}

	// OIDC Config
	if cfg.OIDCEnabled {
		cfg.OIDCIssuer = getEnv("OIDC_ISSUER", "")
		cfg.OIDCClientID = getEnv("OIDC_CLIENT_ID", "")
		cfg.OIDCClientSecret = getEnv("OIDC_CLIENT_SECRET", "")
	}

	// Default Database Config
	if cfg.DBEnabled {
		cfg.DefaultDB = DBConfig{
			Type:     getEnv("DB_DEFAULT_TYPE", "postgresql"),
			Host:     getEnv("DB_DEFAULT_HOST", "localhost"),
			Port:     getEnv("DB_DEFAULT_PORT", "5432"),
			User:     getEnv("DB_DEFAULT_USER", "song_user"),
			Password: getEnv("DB_DEFAULT_PASSWORD", ""),
			Name:     getEnv("DB_DEFAULT_NAME", "song_db"),
			SSLMode:  getEnv("DB_DEFAULT_SSL_MODE", "disable"),
			MaxConns: getIntEnv("DB_DEFAULT_MAX_CONNS", 25),
			MinConns: getIntEnv("DB_DEFAULT_MIN_CONNS", 5),
		}

		// Additional Databases
		additionalNamesStr := getEnv("DB_ADDITIONAL_NAMES", "")
		if additionalNamesStr != "" {
			names := strings.Split(additionalNamesStr, ",")
			cfg.AdditionalDBNames = names
			cfg.AdditionalDBs = make(map[string]DBConfig)

			for _, name := range names {
				name = strings.TrimSpace(name)
				prefix := strings.ToUpper(name)

				dbCfg := DBConfig{
					Type:     getEnv(fmt.Sprintf("DB_%s_TYPE", prefix), "postgresql"),
					Host:     getEnv(fmt.Sprintf("DB_%s_HOST", prefix), ""),
					Port:     getEnv(fmt.Sprintf("DB_%s_PORT", prefix), "5432"),
					User:     getEnv(fmt.Sprintf("DB_%s_USER", prefix), ""),
					Password: getEnv(fmt.Sprintf("DB_%s_PASSWORD", prefix), ""),
					Name:     getEnv(fmt.Sprintf("DB_%s_NAME", prefix), ""),
					SSLMode:  getEnv(fmt.Sprintf("DB_%s_SSL_MODE", prefix), "disable"),
					MaxConns: getIntEnv(fmt.Sprintf("DB_%s_MAX_CONNS", prefix), 25),
					MinConns: getIntEnv(fmt.Sprintf("DB_%s_MIN_CONNS", prefix), 5),
				}

				// Special handling for Redis
				if dbCfg.Type == "redis" {
					dbCfg.Extra = make(map[string]string)
					dbCfg.Extra["db"] = getEnv(fmt.Sprintf("DB_%s_DB", prefix), "0")
					dbCfg.Extra["password"] = dbCfg.Password // Redis password
				}

				cfg.AdditionalDBs[name] = dbCfg
			}
		}
	}

	return cfg, nil
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	val := getEnv(key, "")
	if val == "" {
		return defaultValue
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return defaultValue
	}
	return b
}

func getIntEnv(key string, defaultValue int) int {
	val := getEnv(key, "")
	if val == "" {
		return defaultValue
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	return i
}

func getInt64Env(key string, defaultValue int64) int64 {
	val := getEnv(key, "")
	if val == "" {
		return defaultValue
	}
	i, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return defaultValue
	}
	return i
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	val := getEnv(key, "")
	if val == "" {
		return defaultValue
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		return defaultValue
	}
	return d
}
