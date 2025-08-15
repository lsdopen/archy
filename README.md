# Archy

A Kubernetes mutating webhook that automatically adds node selectors based on container image architectures.

## Overview

Archy analyzes container images in pod specifications and adds appropriate `kubernetes.io/arch` node selectors to ensure pods are scheduled on compatible nodes.

## Development

### Prerequisites

- Go 1.21+
- golangci-lint
- make

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

## Configuration

Configure via environment variables:

- `PORT`: Server port (required)
- `TLS_CERT_PATH`: TLS certificate path (required)  
- `TLS_KEY_PATH`: TLS private key path (required)
- `DEFAULT_ARCH`: Default architecture fallback (default: amd64)
- `LOG_LEVEL`: Log level (default: info)
- `CACHE_TIMEOUT`: Cache timeout in seconds (default: 300)

## Contributing

1. All code must have 100% test coverage
2. All linting rules must pass
3. Use conventional commits for semantic versioning
4. Write tests before implementation (TDD)