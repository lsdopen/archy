package types

import "context"

// RegistryClient defines the interface for container registry clients
type RegistryClient interface {
	// GetSupportedArchitectures returns the list of architectures supported by the given image
	GetSupportedArchitectures(ctx context.Context, image string) ([]string, error)
}