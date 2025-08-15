package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the webhook
type Config struct {
	Port         int
	TLSCertPath  string
	TLSKeyPath   string
	DefaultArch  string
	LogLevel     string
	CacheTimeout time.Duration
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{}

	// Required fields
	portStr := os.Getenv("PORT")
	if portStr == "" {
		return nil, fmt.Errorf("PORT is required")
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid PORT: %w", err)
	}
	if port < 1 || port > 65535 {
		return nil, fmt.Errorf("PORT must be between 1 and 65535")
	}
	cfg.Port = port

	cfg.TLSCertPath = os.Getenv("TLS_CERT_PATH")
	if cfg.TLSCertPath == "" {
		return nil, fmt.Errorf("TLS_CERT_PATH is required")
	}
	if cfg.TLSCertPath == "" {
		return nil, fmt.Errorf("TLS_CERT_PATH cannot be empty")
	}

	cfg.TLSKeyPath = os.Getenv("TLS_KEY_PATH")
	if cfg.TLSKeyPath == "" {
		return nil, fmt.Errorf("TLS_KEY_PATH is required")
	}

	// Optional fields with defaults
	cfg.DefaultArch = getEnvWithDefault("DEFAULT_ARCH", "amd64")
	cfg.LogLevel = getEnvWithDefault("LOG_LEVEL", "info")

	cacheTimeoutStr := getEnvWithDefault("CACHE_TIMEOUT", "300")
	cacheTimeoutSecs, err := strconv.Atoi(cacheTimeoutStr)
	if err != nil {
		return nil, fmt.Errorf("invalid CACHE_TIMEOUT: %w", err)
	}
	cfg.CacheTimeout = time.Duration(cacheTimeoutSecs) * time.Second

	return cfg, nil
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}