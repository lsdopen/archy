# Archy - Architecture-Aware Mutating Webhook Requirements

## Overview
A Kubernetes mutating webhook that automatically adds node selectors to pods based on the supported architectures of their container images.

## Core Requirements

### 1. Webhook Functionality
- **Admission Controller**: Implement a mutating admission webhook
- **Trigger**: Activate on pod creation/update events
- **Mutation**: Add `kubernetes.io/arch` node selector to pod spec
- **Architecture Detection**: Query container registries to determine supported architectures

### 2. Image Architecture Detection
- **Registry Support**: Docker Hub, ECR, GCR, private registries
- **Multi-arch Images**: Detect all supported architectures (amd64, arm64, etc.)
- **Fallback Strategy**: Default to amd64 if architecture cannot be determined
- **Caching**: Cache architecture information to reduce registry calls

### 3. Node Selector Logic
- **Single Architecture**: Add specific arch selector (e.g., `kubernetes.io/arch: amd64`)
- **Multi-arch Images**: Prefer cluster's most common architecture or configurable default
- **Existing Selectors**: Preserve existing node selectors, only add if arch selector missing
- **Override Prevention**: Skip mutation if pod already has architecture selector

### 4. Configuration
- **Registry Credentials**: Support for private registry authentication via pod imagePullSecrets
- **Credential Resolution**: Automatic discovery from pod and service account imagePullSecrets
- **Default Architecture**: Configurable fallback architecture
- **Webhook Settings**: TLS certificates, port, namespace filtering
- **Logging Level**: Configurable verbosity

### 5. Deployment Requirements
- **Kubernetes Version**: Support 1.20+
- **RBAC**: Minimal required permissions
- **TLS**: Self-signed or cert-manager integration
- **High Availability**: Support multiple replicas
- **Resource Limits**: Defined CPU/memory constraints

### 6. Error Handling
- **Registry Failures**: Graceful degradation with fallback
- **Network Issues**: Retry logic with exponential backoff
- **Invalid Images**: Log warnings, apply default selector
- **Webhook Failures**: Always fail open to allow pod creation to continue

### 7. Observability
- **Metrics**: Prometheus metrics for mutations, errors, cache hits
- **Logging**: Structured logging with request tracing
- **Health Checks**: Readiness and liveness probes

## Development Requirements

### Language and Dependencies
- **Language**: Go (latest stable version)
- **Dependencies**: Minimal external dependencies, prefer standard library
- **Container Image**: Scratch-based multi-architecture image with static binary
- **Build**: Single static binary with no runtime dependencies
- **Multi-arch Support**: Build for amd64, arm64 architectures

### CI/CD Requirements
- **GitHub Actions**: Automated build, test, and deployment pipeline
- **Multi-arch Build**: Cross-compile Go binaries for multiple architectures
- **Container Registry**: GitHub Container Registry (ghcr.io)
- **Multi-arch Images**: Single manifest supporting amd64 and arm64
- **Automated Releases**: Semantic versioning with semantic-release action
- **Helm Chart**: Package and deploy to GitHub OCI registry
- **Chart Deployment**: Automated Helm chart publishing on releases
- **Code Linting**: golangci-lint with strict configuration
- **PR Validation**: All PRs must pass tests and linting
- **Main Branch Protection**: Merges to main trigger automatic releases

### Application Architecture
- **12-Factor App**: Full compliance with 12-factor app principles
- **Configuration**: Environment variables only, no config files
- **Stateless**: No local state, horizontally scalable
- **Process Isolation**: Single responsibility per process
- **Port Binding**: Self-contained HTTP service

### Testing Requirements
- **Unit Tests**: 100% coverage for all business logic
- **Integration Tests**: End-to-end webhook functionality
- **Failure Scenarios**: Network failures, registry timeouts, invalid manifests
- **Performance Tests**: Load testing for throughput requirements
- **Security Tests**: TLS configuration and certificate validation

## Technical Specifications

### Architecture Detection Flow
1. Extract image references from pod spec
2. Query registry manifest for each image
3. Parse manifest to identify supported architectures
4. Apply selection logic for node selector
5. Mutate pod spec with appropriate selector

### Webhook Configuration
- **Path**: `/mutate`
- **Port**: 8443 (HTTPS)
- **Failure Policy**: Ignore (fail open)
- **Namespace Selector**: Configurable inclusion/exclusion

### Performance Requirements
- **Response Time**: <100ms for cached results, <500ms for registry queries
- **Throughput**: Handle 100+ pod creations per second
- **Memory Usage**: <256MB per replica
- **CPU Usage**: <100m per replica

## Implementation Priorities
1. Basic webhook framework with TLS
2. Image architecture detection for Docker Hub
3. Node selector mutation logic
4. Configuration and deployment manifests
5. Extended registry support and caching
6. Observability and error handling