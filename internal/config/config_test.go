package config

import (
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_MissingRequiredEnvVars(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr string
	}{
		{
			name:    "missing port",
			envVars: map[string]string{},
			wantErr: "PORT is required",
		},
		{
			name: "missing cert path",
			envVars: map[string]string{
				"PORT": "8443",
			},
			wantErr: "TLS_CERT_PATH is required",
		},
		{
			name: "missing key path",
			envVars: map[string]string{
				"PORT":          "8443",
				"TLS_CERT_PATH": "/tmp/cert.pem",
			},
			wantErr: "TLS_KEY_PATH is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv()
			setEnvVars(tt.envVars)

			_, err := Load()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestLoad_InvalidDataTypes(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr string
	}{
		{
			name: "invalid port",
			envVars: map[string]string{
				"PORT":          "invalid",
				"TLS_CERT_PATH": "/tmp/cert.pem",
				"TLS_KEY_PATH":  "/tmp/key.pem",
			},
			wantErr: "invalid PORT",
		},
		{
			name: "invalid cache timeout",
			envVars: map[string]string{
				"PORT":           "8443",
				"TLS_CERT_PATH":  "/tmp/cert.pem",
				"TLS_KEY_PATH":   "/tmp/key.pem",
				"CACHE_TIMEOUT":  "invalid",
			},
			wantErr: "invalid CACHE_TIMEOUT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv()
			setEnvVars(tt.envVars)

			_, err := Load()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestLoad_BoundaryValues(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr string
	}{
		{
			name: "port too low",
			envVars: map[string]string{
				"PORT":          "0",
				"TLS_CERT_PATH": "/tmp/cert.pem",
				"TLS_KEY_PATH":  "/tmp/key.pem",
			},
			wantErr: "PORT must be between 1 and 65535",
		},
		{
			name: "port too high",
			envVars: map[string]string{
				"PORT":          "65536",
				"TLS_CERT_PATH": "/tmp/cert.pem",
				"TLS_KEY_PATH":  "/tmp/key.pem",
			},
			wantErr: "PORT must be between 1 and 65535",
		},
		{
			name: "empty cert path",
			envVars: map[string]string{
				"PORT":          "8443",
				"TLS_CERT_PATH": "",
				"TLS_KEY_PATH":  "/tmp/key.pem",
			},
			wantErr: "TLS_CERT_PATH cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv()
			setEnvVars(tt.envVars)

			_, err := Load()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestLoad_ValidConfiguration(t *testing.T) {
	clearEnv()
	setEnvVars(map[string]string{
		"PORT":           "8443",
		"TLS_CERT_PATH":  "/tmp/cert.pem",
		"TLS_KEY_PATH":   "/tmp/key.pem",
		"DEFAULT_ARCH":   "arm64",
		"LOG_LEVEL":      "debug",
		"CACHE_TIMEOUT":  "600",
	})

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, 8443, cfg.Port)
	assert.Equal(t, "/tmp/cert.pem", cfg.TLSCertPath)
	assert.Equal(t, "/tmp/key.pem", cfg.TLSKeyPath)
	assert.Equal(t, "arm64", cfg.DefaultArch)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, 600*time.Second, cfg.CacheTimeout)
}

func TestLoad_DefaultValues(t *testing.T) {
	clearEnv()
	setEnvVars(map[string]string{
		"PORT":          "8443",
		"TLS_CERT_PATH": "/tmp/cert.pem",
		"TLS_KEY_PATH":  "/tmp/key.pem",
	})

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "amd64", cfg.DefaultArch)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, 300*time.Second, cfg.CacheTimeout)
}

func TestLoad_ConcurrentAccess(t *testing.T) {
	clearEnv()
	setEnvVars(map[string]string{
		"PORT":          "8443",
		"TLS_CERT_PATH": "/tmp/cert.pem",
		"TLS_KEY_PATH":  "/tmp/key.pem",
	})

	var wg sync.WaitGroup
	results := make(chan *Config, 10)
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cfg, err := Load()
			if err != nil {
				errors <- err
				return
			}
			results <- cfg
		}()
	}

	wg.Wait()
	close(results)
	close(errors)

	// Check no errors occurred
	for err := range errors {
		t.Errorf("Unexpected error: %v", err)
	}

	// Check all configs are identical
	var configs []*Config
	for cfg := range results {
		configs = append(configs, cfg)
	}

	require.Len(t, configs, 10)
	for i := 1; i < len(configs); i++ {
		assert.Equal(t, configs[0], configs[i])
	}
}

func clearEnv() {
	envVars := []string{
		"PORT", "TLS_CERT_PATH", "TLS_KEY_PATH", "DEFAULT_ARCH",
		"LOG_LEVEL", "CACHE_TIMEOUT",
	}
	for _, env := range envVars {
		os.Unsetenv(env)
	}
}

func setEnvVars(vars map[string]string) {
	for k, v := range vars {
		os.Setenv(k, v)
	}
}