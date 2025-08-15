# Archy

A Kubernetes mutating webhook that automatically adds node selectors based on container image architectures.

## Overview

Archy analyzes container images in pod specifications and adds appropriate `kubernetes.io/arch` node selectors to ensure pods are scheduled on compatible nodes.

## Installation

### Using Helm (Recommended)

```bash
helm install archy oci://ghcr.io/lsdopen/charts/archy
```

### Configuration

Configure via Helm values:

```yaml
config:
  defaultArch: amd64
  logLevel: info
  cacheTimeout: 300s

tls:
  certManager: true
  issuer: selfsigned-issuer

monitoring:
  serviceMonitor:
    enabled: true
  prometheusRule:
    enabled: true
```

## Development

### Prerequisites

- Go 1.21+
- golangci-lint
- make
- Helm 3.12+

### Setup

```bash
git clone <repository>
cd archy
go mod tidy
```

### Testing

Run all tests with coverage:
```bash
make test-coverage
```

### Linting

```bash
make lint
```

### Building

```bash
make build
```

### Helm Chart Development

```bash
# Lint chart
helm lint chart/

# Test installation
helm install archy-test chart/ --dry-run
```

## Configuration

Configure via environment variables:

- `PORT`: Server port (required)
- `TLS_CERT_PATH`: TLS certificate path (required)  
- `TLS_KEY_PATH`: TLS private key path (required)
- `DEFAULT_ARCH`: Default architecture fallback (default: amd64)
- `LOG_LEVEL`: Log level (default: info)
- `CACHE_TIMEOUT`: Cache timeout in seconds (default: 300)

## Development

### Testing Philosophy
- **Test-Driven Development**: Write tests before implementation
- **100% Code Coverage**: All functions must have corresponding tests
- **Comprehensive Edge Cases**: Test all failure scenarios and boundary conditions
- **Concurrent Testing**: Verify thread safety and race conditions
- **Integration Testing**: End-to-end webhook functionality validation

### Running Tests
```bash
# Run all tests with coverage
make test-coverage

# Run tests with race detection
make test-race

# Run specific package tests
go test -v ./internal/credentials/...
```

### Linting Requirements
```bash
# Run all linters (must pass with zero warnings)
make lint

# Fix auto-fixable issues
make lint-fix
```

### Build Process
```bash
# Build binary
make build

# Build container image
make container

# Build multi-arch images
make container-multiarch
```

## Contributing

### Development Workflow
1. **Fork and Clone**: Fork the repository and clone locally
2. **Create Branch**: Use descriptive branch names (feat/feature-name, fix/bug-name)
3. **Write Tests First**: Follow TDD approach - write failing tests first
4. **Implement Code**: Write minimal code to make tests pass
5. **Verify Coverage**: Ensure 100% test coverage with `make test-coverage`
6. **Run Linting**: All linting rules must pass with `make lint`
7. **Commit Changes**: Use conventional commits for semantic versioning
8. **Create PR**: Submit pull request with comprehensive description

### Code Quality Standards
- **100% Test Coverage**: No exceptions - all code must be tested
- **Zero Linting Warnings**: All golangci-lint rules must pass
- **Conventional Commits**: Required for automatic semantic versioning
- **Security First**: All security scans must pass
- **Performance**: Benchmarks must meet SLA requirements

### Conventional Commit Format
```
type(scope): description

[optional body]

[optional footer]
```

**Types**: feat, fix, docs, style, refactor, test, chore
**Examples**:
- `feat(webhook): add architecture detection for private registries`
- `fix(cache): resolve race condition in TTL expiration`
- `docs(readme): update installation instructions`

### Code Review Checklist
- [ ] Tests written before implementation (TDD)
- [ ] 100% code coverage maintained
- [ ] All linting rules pass with zero warnings
- [ ] Security scans pass without issues
- [ ] Performance benchmarks meet requirements
- [ ] Conventional commit format used
- [ ] Documentation updated if needed
- [ ] Edge cases and failure scenarios tested
- [ ] Concurrent access patterns tested
- [ ] Integration tests pass