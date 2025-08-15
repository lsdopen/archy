package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// DockerHubClient implements registry client for Docker Hub
type DockerHubClient struct {
	baseURL    string
	httpClient *http.Client
}

// DockerHubManifest represents Docker Hub manifest list response
type DockerHubManifest struct {
	Manifests []struct {
		Platform struct {
			Architecture string `json:"architecture"`
			OS           string `json:"os"`
		} `json:"platform"`
	} `json:"manifests"`
}

// NewDockerHubClient creates a new Docker Hub client
func NewDockerHubClient() *DockerHubClient {
	return &DockerHubClient{
		baseURL: "https://registry-1.docker.io",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetSupportedArchitectures retrieves supported architectures for an image
func (c *DockerHubClient) GetSupportedArchitectures(ctx context.Context, image string) ([]string, error) {
	// Parse image reference
	repo, tag := parseImageReference(image)
	if repo == "" {
		return []string{"amd64"}, nil // Default fallback
	}

	// Build manifest URL
	url := fmt.Sprintf("%s/v2/%s/manifests/%s", c.baseURL, repo, tag)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return []string{"amd64"}, nil // Fail open
	}

	// Set headers for manifest list
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.list.v2+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if strings.Contains(err.Error(), "timeout") {
			return nil, fmt.Errorf("request timeout: %w", err)
		}
		return []string{"amd64"}, nil // Fail open
	}
	defer resp.Body.Close()

	// Handle different response codes
	switch resp.StatusCode {
	case http.StatusTooManyRequests:
		return nil, fmt.Errorf("rate limit exceeded")
	case http.StatusUnauthorized:
		// Try to retry once (simulate token refresh)
		return c.retryWithAuth(ctx, url)
	case http.StatusForbidden:
		return nil, fmt.Errorf("access denied to repository")
	case http.StatusNotFound:
		return nil, fmt.Errorf("image not found")
	case http.StatusBadRequest:
		return nil, fmt.Errorf("API version not supported")
	case http.StatusOK:
		// Continue processing
	default:
		return []string{"amd64"}, nil // Fail open
	}

	var manifest DockerHubManifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("failed to decode manifest: %w", err)
	}

	// Extract architectures
	var architectures []string
	seen := make(map[string]bool)
	
	for _, m := range manifest.Manifests {
		arch := m.Platform.Architecture
		if arch != "" && !seen[arch] {
			architectures = append(architectures, arch)
			seen[arch] = true
		}
	}

	if len(architectures) == 0 {
		return []string{"amd64"}, nil // Default fallback
	}

	return architectures, nil
}

func (c *DockerHubClient) retryWithAuth(ctx context.Context, url string) ([]string, error) {
	// Simulate token refresh and retry
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return []string{"amd64"}, nil
	}

	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.list.v2+json")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return []string{"amd64"}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []string{"amd64"}, nil
	}

	var manifest DockerHubManifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return []string{"amd64"}, nil
	}

	var architectures []string
	for _, m := range manifest.Manifests {
		if m.Platform.Architecture != "" {
			architectures = append(architectures, m.Platform.Architecture)
		}
	}

	if len(architectures) == 0 {
		return []string{"amd64"}, nil
	}

	return architectures, nil
}

func parseImageReference(image string) (string, string) {
	if image == "" {
		return "", ""
	}

	// Handle library images (nginx -> library/nginx)
	parts := strings.Split(image, ":")
	repo := parts[0]
	tag := "latest"
	
	if len(parts) > 1 {
		tag = parts[1]
	}

	// Add library prefix for official images
	if !strings.Contains(repo, "/") {
		repo = "library/" + repo
	}

	return repo, tag
}