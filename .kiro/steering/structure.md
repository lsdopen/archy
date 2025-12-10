# Project Structure

## Directory Layout

```
archy/
├── cmd/webhook/           # Application entry point
│   └── main.go           # HTTP server setup, signal handling, dependency injection
├── pkg/                  # Core business logic packages
│   ├── inspector/        # Container registry inspection
│   │   ├── inspector.go  # Platform detection interface and implementation
│   │   └── auth.go       # Kubernetes authentication for private registries
│   └── webhook/          # Admission webhook logic
│       ├── handler.go    # HTTP handler, AdmissionReview processing, Pod mutation
│       └── handler_test.go # Unit tests with mock inspector
├── deploy/               # Kubernetes manifests
│   ├── deployment.yaml   # Webhook deployment and service
│   └── webhook-config.yaml # MutatingWebhookConfiguration
├── certs/                # TLS certificates for webhook
├── scripts/              # Build and setup scripts
└── bin/                  # Build output directory
```

## Code Organization Patterns

### Package Structure
- **cmd/**: Application entry points only, minimal logic
- **pkg/**: Reusable packages with clear interfaces
- **deploy/**: Infrastructure as code, Kubernetes manifests

### Interface Design
- `Inspector` interface in `pkg/inspector` enables testing and future extensibility
- Dependency injection pattern in main.go for clean separation

### Error Handling
- Fail-safe approach: reject Pods when architecture cannot be determined
- Structured error messages in AdmissionResponse
- Context propagation for request timeouts

### Testing Patterns
- Mock implementations for external dependencies (registries, Kubernetes API)
- Table-driven tests covering edge cases
- Unit tests focus on business logic, not HTTP plumbing

### Configuration
- Command-line flags for server configuration (port, TLS certs)
- Environment-based configuration for Kubernetes client (in-cluster config)
- Kubernetes-native configuration via MutatingWebhookConfiguration

### Naming Conventions
- Go standard naming (PascalCase for exported, camelCase for unexported)
- Package names are lowercase, single word when possible
- Interface names end with -er suffix (Inspector)
- Test files use `_test.go` suffix with same package name