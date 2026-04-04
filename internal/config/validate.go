package config

import (
	"errors"
	"fmt"
	"strings"
)

// ErrInvalidConfig is returned when configuration validation fails
var ErrInvalidConfig = errors.New("invalid configuration")

// Validate checks that the configuration meets all requirements
func Validate(cfg *Config) error {
	var errs []error

	// 1. Validate Server Settings
	if cfg.Host == "" {
		errs = append(errs, errors.New("SONG_HOST cannot be empty"))
	}
	if cfg.Port == "" {
		errs = append(errs, errors.New("SONG_PORT cannot be empty"))
	} else {
		// Basic port range check (1-65535)
		port := 0
		fmt.Sscanf(cfg.Port, "%d", &port)
		if port < 1 || port > 65535 {
			errs = append(errs, fmt.Errorf("SONG_PORT must be between 1 and 65535, got %s", cfg.Port))
		}
	}

	// 2. Validate Core Storage
	if cfg.CoreDir == "" {
		errs = append(errs, errors.New("CORE_DIR cannot be empty"))
	}
	// Ensure core dir is not a web-served path (basic check)
	if strings.Contains(cfg.CoreDir, "static") || strings.Contains(cfg.CoreDir, "public") {
		errs = append(errs, errors.New("CORE_DIR should not be inside a web-served directory like 'static'"))
	}

	// 3. Validate Authentication
	if cfg.JWTSecret == "" {
		errs = append(errs, errors.New("JWT_SECRET is required for authentication"))
	} else if len(cfg.JWTSecret) < 16 {
		errs = append(errs, errors.New("JWT_SECRET must be at least 16 characters long"))
	}

	if cfg.OIDCEnabled {
		if cfg.OIDCIssuer == "" {
			errs = append(errs, errors.New("OIDC_ISSUER is required when OIDC is enabled"))
		}
		if cfg.OIDCClientID == "" {
			errs = append(errs, errors.New("OIDC_CLIENT_ID is required when OIDC is enabled"))
		}
		if cfg.OIDCClientSecret == "" {
			errs = append(errs, errors.New("OIDC_CLIENT_SECRET is required when OIDC is enabled"))
		}
	}

	// 4. Validate Paths
	if cfg.StaticDir == "" {
		errs = append(errs, errors.New("STATIC_DIR cannot be empty"))
	}
	if cfg.TemplateDir == "" {
		errs = append(errs, errors.New("TEMPLATE_DIR cannot be empty"))
	}
	if cfg.FunctionsDir == "" {
		errs = append(errs, errors.New("FUNCTIONS_DIR cannot be empty"))
	}
	if cfg.DataDir == "" {
		errs = append(errs, errors.New("DATA_DIR cannot be empty"))
	}

	// 5. Validate Security Limits
	if cfg.FunctionTimeout <= 0 {
		errs = append(errs, errors.New("MAX_FUNCTION_TIMEOUT must be greater than 0"))
	}
	if cfg.MaxMemoryMB <= 0 {
		errs = append(errs, errors.New("MAX_MEMORY_MB must be greater than 0"))
	}
	if cfg.MaxCPUPercent <= 0 || cfg.MaxCPUPercent > 100 {
		errs = append(errs, errors.New("MAX_CPU_PERCENT must be between 1 and 100"))
	}

	// 6. Validate Database Configuration
	if cfg.DBEnabled {
		// Validate Default DB
		if cfg.DefaultDB.Host == "" {
			errs = append(errs, errors.New("DB_DEFAULT_HOST is required when DB_ENABLED is true"))
		}
		if cfg.DefaultDB.User == "" {
			errs = append(errs, errors.New("DB_DEFAULT_USER is required when DB_ENABLED is true"))
		}
		if cfg.DefaultDB.Name == "" {
			errs = append(errs, errors.New("DB_DEFAULT_NAME is required when DB_ENABLED is true"))
		}

		// Validate Additional DBs
		for name, dbCfg := range cfg.AdditionalDBs {
			if dbCfg.Host == "" {
				errs = append(errs, fmt.Errorf("DB_%s_HOST is required for additional database '%s'", strings.ToUpper(name), name))
			}
			if dbCfg.User == "" && dbCfg.Type != "sqlite" {
				// SQLite doesn't require a user
				errs = append(errs, fmt.Errorf("DB_%s_USER is required for additional database '%s'", strings.ToUpper(name), name))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%w: %v", ErrInvalidConfig, errs)
	}

	return nil
}
