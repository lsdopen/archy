package registry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientFactory_InvalidRegistryURLs(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr string
	}{
		{
			name:    "empty URL",
			url:     "",
			wantErr: "registry URL cannot be empty",
		},
		{
			name:    "invalid URL format",
			url:     ":::invalid:::url:::",
			wantErr: "invalid registry URL",
		},
		{
			name:    "unsupported scheme",
			url:     "ftp://registry.example.com",
			wantErr: "unsupported URL scheme",
		},
		{
			name:    "malformed URL",
			url:     "http://[::1:80",
			wantErr: "invalid registry URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.url)
			require.Error(t, err)
			assert.Nil(t, client)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestClientFactory_UnsupportedRegistryTypes(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr string
	}{
		{
			name:    "unsupported registry",
			url:     "https://unsupported-registry.com",
			wantErr: "unsupported registry",
		},
		{
			name:    "unknown registry domain",
			url:     "https://unknown.registry.example.com",
			wantErr: "unsupported registry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.url)
			require.Error(t, err)
			assert.Nil(t, client)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestClientFactory_NetworkFailures(t *testing.T) {
	// Test with unreachable registry
	client, err := NewClient("https://127.0.0.1:1") // Invalid port
	
	// Factory should succeed, but actual calls should fail
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestClientFactory_AuthenticationFailures(t *testing.T) {
	// Test with registry requiring authentication
	client, err := NewClient("https://registry-1.docker.io")
	
	// Factory should succeed even without auth
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestClientFactory_SupportedRegistries(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{
			name: "Docker Hub",
			url:  "https://registry-1.docker.io",
		},
		{
			name: "Docker Hub index",
			url:  "https://index.docker.io",
		},
		{
			name: "Docker Hub default",
			url:  "docker.io",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.url)
			require.NoError(t, err)
			assert.NotNil(t, client)
		})
	}
}