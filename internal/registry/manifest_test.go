package registry

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManifestParser_InvalidJSON(t *testing.T) {
	tests := []struct {
		name     string
		manifest string
		wantErr  string
	}{
		{
			name:     "completely invalid JSON",
			manifest: `{invalid json}`,
			wantErr:  "invalid character",
		},
		{
			name:     "empty manifest",
			manifest: ``,
			wantErr:  "unexpected end of JSON input",
		},
		{
			name:     "null manifest",
			manifest: `null`,
			wantErr:  "manifest is null",
		},
		{
			name:     "truncated JSON",
			manifest: `{"manifests":[{"platform":{"architecture":"amd64"`,
			wantErr:  "unexpected end of JSON input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewManifestParser()
			archs, err := parser.ParseArchitectures([]byte(tt.manifest))
			
			require.Error(t, err)
			assert.Nil(t, archs)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestManifestParser_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name     string
		manifest string
		wantErr  string
	}{
		{
			name:     "missing manifests field",
			manifest: `{"schemaVersion": 2}`,
			wantErr:  "no manifests found",
		},
		{
			name:     "empty manifests array",
			manifest: `{"manifests": []}`,
			wantErr:  "no manifests found",
		},
		{
			name:     "manifest without platform",
			manifest: `{"manifests": [{"digest": "sha256:abc123"}]}`,
			wantErr:  "no valid platforms found",
		},
		{
			name:     "platform without architecture",
			manifest: `{"manifests": [{"platform": {"os": "linux"}}]}`,
			wantErr:  "no valid architectures found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewManifestParser()
			archs, err := parser.ParseArchitectures([]byte(tt.manifest))
			
			require.Error(t, err)
			assert.Nil(t, archs)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestManifestParser_UnsupportedSchemaVersions(t *testing.T) {
	tests := []struct {
		name     string
		manifest string
		wantErr  string
	}{
		{
			name:     "schema version 1",
			manifest: `{"schemaVersion": 1, "manifests": [{"platform": {"architecture": "amd64"}}]}`,
			wantErr:  "unsupported schema version",
		},
		{
			name:     "schema version 3",
			manifest: `{"schemaVersion": 3, "manifests": [{"platform": {"architecture": "amd64"}}]}`,
			wantErr:  "unsupported schema version",
		},
		{
			name:     "missing schema version",
			manifest: `{"manifests": [{"platform": {"architecture": "amd64"}}]}`,
			wantErr:  "missing schema version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewManifestParser()
			archs, err := parser.ParseArchitectures([]byte(tt.manifest))
			
			require.Error(t, err)
			assert.Nil(t, archs)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestManifestParser_UnknownArchitectures(t *testing.T) {
	manifest := `{
		"schemaVersion": 2,
		"manifests": [
			{"platform": {"architecture": "unknown-arch", "os": "linux"}},
			{"platform": {"architecture": "future-arch", "os": "linux"}}
		]
	}`

	parser := NewManifestParser()
	archs, err := parser.ParseArchitectures([]byte(manifest))
	
	require.NoError(t, err)
	assert.Equal(t, []string{"unknown-arch", "future-arch"}, archs)
}

func TestManifestParser_MixedSchemaVersions(t *testing.T) {
	// This shouldn't happen in practice, but test robustness
	manifest := `{
		"schemaVersion": 2,
		"manifests": [
			{"platform": {"architecture": "amd64", "os": "linux"}},
			{"schemaVersion": 1, "platform": {"architecture": "arm64", "os": "linux"}}
		]
	}`

	parser := NewManifestParser()
	archs, err := parser.ParseArchitectures([]byte(manifest))
	
	require.NoError(t, err)
	assert.Contains(t, archs, "amd64")
	assert.Contains(t, archs, "arm64")
}

func TestManifestParser_ExtremelyLargeManifest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large manifest test in short mode")
	}

	// Create manifest with 10,000 entries
	manifest := `{"schemaVersion": 2, "manifests": [`
	for i := 0; i < 10000; i++ {
		if i > 0 {
			manifest += ","
		}
		manifest += `{"platform": {"architecture": "amd64", "os": "linux"}}`
	}
	manifest += `]}`

	parser := NewManifestParser()
	archs, err := parser.ParseArchitectures([]byte(manifest))
	
	require.NoError(t, err)
	assert.Equal(t, []string{"amd64"}, archs) // Should deduplicate
}

func TestManifestParser_CircularReferences(t *testing.T) {
	// Test manifest with potential circular reference structure
	manifest := `{
		"schemaVersion": 2,
		"manifests": [
			{
				"platform": {"architecture": "amd64", "os": "linux"},
				"digest": "sha256:abc123",
				"references": ["sha256:def456"]
			}
		]
	}`

	parser := NewManifestParser()
	archs, err := parser.ParseArchitectures([]byte(manifest))
	
	require.NoError(t, err)
	assert.Equal(t, []string{"amd64"}, archs)
}

func TestManifestParser_ConcurrentParsing(t *testing.T) {
	manifest := `{
		"schemaVersion": 2,
		"manifests": [
			{"platform": {"architecture": "amd64", "os": "linux"}},
			{"platform": {"architecture": "arm64", "os": "linux"}}
		]
	}`

	parser := NewManifestParser()
	done := make(chan bool, 10)
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			
			archs, err := parser.ParseArchitectures([]byte(manifest))
			if err != nil {
				errors <- err
				return
			}
			
			if len(archs) != 2 {
				errors <- fmt.Errorf("expected 2 architectures, got %d", len(archs))
			}
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent parsing failed: %v", err)
	}
}

func TestManifestParser_ValidManifests(t *testing.T) {
	tests := []struct {
		name     string
		manifest string
		expected []string
	}{
		{
			name: "single architecture",
			manifest: `{
				"schemaVersion": 2,
				"manifests": [
					{"platform": {"architecture": "amd64", "os": "linux"}}
				]
			}`,
			expected: []string{"amd64"},
		},
		{
			name: "multiple architectures",
			manifest: `{
				"schemaVersion": 2,
				"manifests": [
					{"platform": {"architecture": "amd64", "os": "linux"}},
					{"platform": {"architecture": "arm64", "os": "linux"}},
					{"platform": {"architecture": "arm", "os": "linux"}}
				]
			}`,
			expected: []string{"amd64", "arm64", "arm"},
		},
		{
			name: "duplicate architectures",
			manifest: `{
				"schemaVersion": 2,
				"manifests": [
					{"platform": {"architecture": "amd64", "os": "linux"}},
					{"platform": {"architecture": "amd64", "os": "windows"}}
				]
			}`,
			expected: []string{"amd64"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewManifestParser()
			archs, err := parser.ParseArchitectures([]byte(tt.manifest))
			
			require.NoError(t, err)
			assert.ElementsMatch(t, tt.expected, archs)
		})
	}
}