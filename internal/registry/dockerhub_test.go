package registry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDockerHubClient_APIRateLimiting(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"errors":[{"code":"TOOMANYREQUESTS","message":"Too Many Requests"}]}`))
	}))
	defer server.Close()

	client := &DockerHubClient{
		baseURL:    server.URL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	ctx := context.Background()
	archs, err := client.GetSupportedArchitectures(ctx, "nginx:latest")
	
	require.Error(t, err)
	assert.Nil(t, archs)
	assert.Contains(t, err.Error(), "rate limit")
}

func TestDockerHubClient_NetworkTimeouts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond) // Simulate slow response
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &DockerHubClient{
		baseURL:    server.URL,
		httpClient: &http.Client{Timeout: 10 * time.Millisecond}, // Very short timeout
	}

	ctx := context.Background()
	archs, err := client.GetSupportedArchitectures(ctx, "nginx:latest")
	
	require.Error(t, err)
	assert.Nil(t, archs)
	assert.Contains(t, err.Error(), "timeout")
}

func TestDockerHubClient_MalformedJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{invalid json}`))
	}))
	defer server.Close()

	client := &DockerHubClient{
		baseURL:    server.URL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	ctx := context.Background()
	archs, err := client.GetSupportedArchitectures(ctx, "nginx:latest")
	
	require.Error(t, err)
	assert.Nil(t, archs)
	assert.Contains(t, err.Error(), "invalid character")
}

func TestDockerHubClient_AuthenticationTokenExpiry(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			// First call - token expired
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"errors":[{"code":"UNAUTHORIZED","message":"authentication required"}]}`))
		} else {
			// Second call - success after token refresh
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"manifests":[{"platform":{"architecture":"amd64"}}]}`))
		}
	}))
	defer server.Close()

	client := &DockerHubClient{
		baseURL:    server.URL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	ctx := context.Background()
	archs, err := client.GetSupportedArchitectures(ctx, "nginx:latest")
	
	// Should retry and succeed
	require.NoError(t, err)
	assert.Equal(t, []string{"amd64"}, archs)
	assert.Equal(t, 2, callCount)
}

func TestDockerHubClient_PrivateRepositoryAccessDenied(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"errors":[{"code":"DENIED","message":"requested access to the resource is denied"}]}`))
	}))
	defer server.Close()

	client := &DockerHubClient{
		baseURL:    server.URL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	ctx := context.Background()
	archs, err := client.GetSupportedArchitectures(ctx, "private/repo:latest")
	
	require.Error(t, err)
	assert.Nil(t, archs)
	assert.Contains(t, err.Error(), "access denied")
}

func TestDockerHubClient_NonExistentImage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"errors":[{"code":"NAME_UNKNOWN","message":"repository name not known to registry"}]}`))
	}))
	defer server.Close()

	client := &DockerHubClient{
		baseURL:    server.URL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	ctx := context.Background()
	archs, err := client.GetSupportedArchitectures(ctx, "nonexistent/image:latest")
	
	require.Error(t, err)
	assert.Nil(t, archs)
	assert.Contains(t, err.Error(), "not found")
}

func TestDockerHubClient_RegistryAPIVersionChanges(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate API version change
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"errors":[{"code":"UNSUPPORTED","message":"API version not supported"}]}`))
	}))
	defer server.Close()

	client := &DockerHubClient{
		baseURL:    server.URL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	ctx := context.Background()
	archs, err := client.GetSupportedArchitectures(ctx, "nginx:latest")
	
	require.Error(t, err)
	assert.Nil(t, archs)
	assert.Contains(t, err.Error(), "API version")
}

func TestDockerHubClient_ConcurrentAPICalls(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"manifests":[{"platform":{"architecture":"amd64"}}]}`))
	}))
	defer server.Close()

	client := &DockerHubClient{
		baseURL:    server.URL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	done := make(chan bool, 10)
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			
			ctx := context.Background()
			_, err := client.GetSupportedArchitectures(ctx, "nginx:latest")
			if err != nil {
				errors <- err
			}
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
	close(errors)

	// Check no errors occurred
	for err := range errors {
		t.Errorf("Concurrent API call failed: %v", err)
	}

	assert.Equal(t, 10, callCount)
}

func TestDockerHubClient_LargeManifestHandling(t *testing.T) {
	// Create a large manifest response (>1MB)
	largeManifest := `{"manifests":[`
	for i := 0; i < 10000; i++ {
		if i > 0 {
			largeManifest += ","
		}
		largeManifest += `{"platform":{"architecture":"amd64","os":"linux"}}`
	}
	largeManifest += `]}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(largeManifest))
	}))
	defer server.Close()

	client := &DockerHubClient{
		baseURL:    server.URL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	ctx := context.Background()
	archs, err := client.GetSupportedArchitectures(ctx, "nginx:latest")
	
	require.NoError(t, err)
	assert.Equal(t, []string{"amd64"}, archs)
}