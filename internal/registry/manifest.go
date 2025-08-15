package registry

import (
	"encoding/json"
	"fmt"
)

// ManifestParser handles parsing of container manifests
type ManifestParser struct{}

// ManifestList represents a Docker manifest list
type ManifestList struct {
	SchemaVersion int `json:"schemaVersion"`
	Manifests     []struct {
		Platform struct {
			Architecture string `json:"architecture"`
			OS           string `json:"os"`
		} `json:"platform"`
	} `json:"manifests"`
}

// NewManifestParser creates a new manifest parser
func NewManifestParser() *ManifestParser {
	return &ManifestParser{}
}

// ParseArchitectures extracts supported architectures from a manifest
func (p *ManifestParser) ParseArchitectures(manifestData []byte) ([]string, error) {
	if len(manifestData) == 0 {
		return nil, fmt.Errorf("manifest data is empty")
	}

	var manifest ManifestList
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Check for null manifest
	if string(manifestData) == "null" {
		return nil, fmt.Errorf("manifest is null")
	}

	// Validate schema version
	if manifest.SchemaVersion == 0 {
		return nil, fmt.Errorf("missing schema version")
	}
	if manifest.SchemaVersion != 2 {
		return nil, fmt.Errorf("unsupported schema version: %d", manifest.SchemaVersion)
	}

	// Check if manifests exist
	if len(manifest.Manifests) == 0 {
		return nil, fmt.Errorf("no manifests found")
	}

	// Extract architectures
	var architectures []string
	seen := make(map[string]bool)
	validPlatforms := false

	for _, m := range manifest.Manifests {
		if m.Platform.Architecture == "" {
			continue
		}
		validPlatforms = true
		
		arch := m.Platform.Architecture
		if !seen[arch] {
			architectures = append(architectures, arch)
			seen[arch] = true
		}
	}

	if !validPlatforms {
		return nil, fmt.Errorf("no valid platforms found")
	}

	if len(architectures) == 0 {
		return nil, fmt.Errorf("no valid architectures found")
	}

	return architectures, nil
}