# Technology Stack

## Language & Runtime
- **Go 1.25.0** - Primary language
- **Kubernetes API** - Built for Kubernetes admission webhook pattern

## Key Dependencies
- `github.com/google/go-containerregistry` - Container registry inspection and authentication
- `k8s.io/api` & `k8s.io/apimachinery` - Kubernetes API types and utilities
- `k8s.io/client-go` - Kubernetes client library for accessing secrets/ServiceAccounts

## Architecture Pattern
- **Admission Webhook**: HTTP server that receives AdmissionReview requests from Kubernetes API server
- **Interface-based Design**: `Inspector` interface allows for testing with mocks
- **Graceful Shutdown**: Proper signal handling and server shutdown

## Build System

### Makefile Targets
```bash
# Build for current platform
make build

# Cross-compile for specific architectures
make build-amd64    # Linux AMD64
make build-arm64    # Linux ARM64

# Clean build artifacts
make clean
```

### Build Configuration
- Binary output: `bin/webhook`
- CGO disabled for static binaries
- Linux target for container deployment

## Development & Testing

### Running Tests
```bash
go test ./...                    # Run all tests
go test ./pkg/webhook -v         # Run webhook tests with verbose output
```

### Local Development
- Uses Tilt for local Kubernetes development
- TLS certificates generated via `scripts/gen-certs.sh`
- Health check endpoint at `/healthz`

## Deployment
- **Container-based**: Containerfile for building images
- **Kubernetes native**: Deployed as Deployment + Service + MutatingWebhookConfiguration
- **TLS required**: Admission webhooks must use HTTPS
- **RBAC**: Requires access to secrets in target namespaces for private registry authentication