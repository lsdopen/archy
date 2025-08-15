package registry

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/lsdopen/archy/pkg/types"
)

// NewClient creates a new registry client based on the registry URL
func NewClient(registryURL string) (types.RegistryClient, error) {
	if registryURL == "" {
		return nil, fmt.Errorf("registry URL cannot be empty")
	}

	// Handle Docker Hub special cases
	if registryURL == "docker.io" || registryURL == "index.docker.io" {
		registryURL = "https://registry-1.docker.io"
	}

	// Parse URL if it doesn't have a scheme
	if !strings.Contains(registryURL, "://") {
		registryURL = "https://" + registryURL
	}

	parsedURL, err := url.Parse(registryURL)
	if err != nil {
		return nil, fmt.Errorf("invalid registry URL: %w", err)
	}

	if parsedURL.Scheme != "https" && parsedURL.Scheme != "http" {
		return nil, fmt.Errorf("unsupported URL scheme: %s", parsedURL.Scheme)
	}

	// Determine registry type based on hostname
	hostname := parsedURL.Hostname()
	switch {
	case strings.Contains(hostname, "docker.io") || strings.Contains(hostname, "registry-1.docker.io"):
		return NewDockerHubClient(), nil
	default:
		return nil, fmt.Errorf("unsupported registry: %s", hostname)
	}
}