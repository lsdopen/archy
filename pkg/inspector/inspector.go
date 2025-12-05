package inspector

import (
	"context"
	"fmt"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

// Platform represents a supported OS/Architecture pair
type Platform struct {
	Architecture string
}

// Inspector defines the interface for inspecting images
type Inspector interface {
	GetSupportedPlatforms(ctx context.Context, image string, opts ...remote.Option) ([]Platform, error)
}

// RegistryInspector implements Inspector using a remote registry
type RegistryInspector struct {
	// We can add caching here later
}

// NewRegistryInspector creates a new RegistryInspector
func NewRegistryInspector() *RegistryInspector {
	return &RegistryInspector{}
}

// GetSupportedPlatforms fetches the manifest for the given image and returns supported platforms
func (r *RegistryInspector) GetSupportedPlatforms(ctx context.Context, imageRef string, opts ...remote.Option) ([]Platform, error) {
	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return nil, fmt.Errorf("parsing reference %q: %w", imageRef, err)
	}

	// Add context to the options
	opts = append(opts, remote.WithContext(ctx))

	// Fetch the index (manifest list) or manifest with authentication options
	desc, err := remote.Get(ref, opts...)
	if err != nil {
		return nil, fmt.Errorf("fetching image %q: %w", imageRef, err)
	}

	// If it's an image index (multi-arch), we iterate over manifests
	if desc.MediaType.IsIndex() {
		index, err := desc.ImageIndex()
		if err != nil {
			return nil, fmt.Errorf("getting image index: %w", err)
		}

		manifest, err := index.IndexManifest()
		if err != nil {
			return nil, fmt.Errorf("getting index manifest: %w", err)
		}

		var platforms []Platform
		for _, descriptor := range manifest.Manifests {
			if descriptor.Platform != nil {
				platforms = append(platforms, Platform{
					Architecture: descriptor.Platform.Architecture,
				})
			}
		}
		return platforms, nil
	}

	// If it's a single image manifest
	if desc.MediaType.IsImage() {
		img, err := desc.Image()
		if err != nil {
			return nil, fmt.Errorf("getting image: %w", err)
		}
		cfg, err := img.ConfigFile()
		if err != nil {
			return nil, fmt.Errorf("getting config file: %w", err)
		}
		return []Platform{{
			Architecture: cfg.Architecture,
		}}, nil
	}

	return nil, fmt.Errorf("unknown media type: %s", desc.MediaType)
}
