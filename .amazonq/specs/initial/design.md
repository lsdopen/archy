# Archy - Design Document

## Project Structure

```
archy/
├── cmd/
│   └── webhook/
│       └── main.go              # Application entry point
├── internal/
│   ├── webhook/
│   │   ├── server.go            # HTTP server and TLS setup
│   │   ├── mutator.go           # Pod mutation logic
│   │   └── handler.go           # Admission webhook handler
│   ├── registry/
│   │   ├── client.go            # Registry client interface
│   │   ├── dockerhub.go         # Docker Hub implementation
│   │   ├── ecr.go               # ECR implementation
│   │   └── manifest.go          # Manifest parsing
│   ├── cache/
│   │   └── memory.go            # In-memory architecture cache
│   ├── config/
│   │   └── config.go            # Configuration management
│   └── metrics/
│       └── prometheus.go        # Metrics collection
├── pkg/
│   └── types/
│       └── types.go             # Shared types and interfaces
├── chart/
│   ├── Chart.yaml               # Helm chart metadata
│   ├── values.yaml              # Default configuration values
│   └── templates/
│       ├── deployment.yaml      # Webhook deployment template
│       ├── service.yaml         # Service template
│       ├── rbac.yaml            # RBAC templates
│       ├── configmap.yaml       # Configuration template
│       ├── secret.yaml          # TLS certificate template
│       ├── webhook.yaml         # MutatingWebhookConfiguration
│       └── _helpers.tpl         # Template helpers
├── .github/
│   └── workflows/
│       ├── build.yaml           # Build and test workflow
│       ├── release.yaml         # Release workflow
│       └── helm.yaml            # Helm chart workflow
├── Containerfile               # Multi-arch container build
├── go.mod                      # Go module definition
└── Makefile                    # Build automation
```

## Implementation Strategy

### Phase 1: Core Webhook Framework
1. **HTTP Server**: TLS-enabled server with health endpoints
2. **Admission Handler**: Basic webhook request/response handling
3. **Configuration**: Environment-based configuration loading
4. **Deployment**: Helm chart templates only

### Phase 2: Architecture Detection
1. **Registry Interface**: Pluggable registry client design
2. **Docker Hub Client**: Manifest API integration
3. **Manifest Parser**: Extract architecture information
4. **Caching Layer**: In-memory cache with TTL

### Phase 3: Pod Mutation Logic
1. **Image Extraction**: Parse pod spec for container images
2. **Architecture Resolution**: Query registries for supported archs
3. **Node Selector Logic**: Add appropriate kubernetes.io/arch selector
4. **Error Handling**: Graceful fallbacks and logging

### Phase 4: Extended Features
1. **Multi-Registry Support**: ECR, GCR, private registries
2. **Authentication**: Registry credential management
3. **Observability**: Prometheus metrics and structured logging
4. **Performance**: Concurrent processing and optimization

### Phase 5: Helm Chart and Deployment
1. **Helm Templates**: Parameterized Kubernetes manifests
2. **Chart Packaging**: OCI-compliant Helm chart
3. **GitHub Registry**: Automated chart publishing
4. **Installation Guide**: Documentation for chart deployment

## Key Design Decisions

### Minimal Dependencies
- Use Go standard library where possible
- Only essential external dependencies:
  - `k8s.io/api` for Kubernetes types
  - `k8s.io/apimachinery` for admission review
  - `prometheus/client_golang` for metrics

### Registry Client Architecture
```go
type RegistryClient interface {
    GetSupportedArchitectures(image string) ([]string, error)
}
```

### Caching Strategy
- In-memory cache with configurable TTL
- Cache key: image reference
- Cache value: supported architectures list
- LRU eviction for memory management

### Error Handling Philosophy
- Always fail open to prevent pod creation blocking
- Log all errors for observability
- Use fallback architecture (amd64) when detection fails
- Implement circuit breaker for registry failures

### Multi-Architecture Build Process
1. **Go Cross-Compilation**: Build static binaries for amd64/arm64
2. **Docker Buildx**: Create multi-arch container images
3. **Manifest Lists**: Single image reference supporting multiple archs
4. **GitHub Actions**: Automated build and push to GHCR

### Test-Driven Development Strategy
- **Test-First Approach**: Write comprehensive tests before implementation
- **100% Code Coverage**: All functions must have corresponding tests
- **Edge Case Testing**: Exhaustive boundary condition testing
- **Failure Mode Testing**: Every possible failure scenario must be tested
- **Property-Based Testing**: Use fuzzing for input validation
- **Mutation Testing**: Verify test quality by introducing code mutations
- **Performance Testing**: Load testing with realistic failure injection
- **Security Testing**: Penetration testing and vulnerability scanning
- **Chaos Testing**: Random failure injection during integration tests

## 12-Factor App Compliance
- **Codebase**: Single repository with multiple deployments
- **Dependencies**: Explicit dependency declaration via go.mod
- **Config**: Environment variables only, no config files
- **Backing Services**: Registry APIs as attached resources
- **Build/Release/Run**: Strict separation via CI/CD pipeline
- **Processes**: Stateless, share-nothing architecture
- **Port Binding**: Self-contained HTTP service on configurable port
- **Concurrency**: Horizontal scaling via Kubernetes replicas
- **Disposability**: Fast startup, graceful shutdown
- **Dev/Prod Parity**: Identical environments via containers
- **Logs**: Structured logging to stdout/stderr
- **Admin Processes**: No admin processes required

## Helm Chart Design

### Chart Structure
- **Chart.yaml**: Metadata with semantic versioning
- **values.yaml**: Configurable parameters with sensible defaults
- **Templates**: Parameterized Kubernetes manifests
- **Helpers**: Reusable template functions

### Key Templates
- **Deployment**: Webhook pod with configurable resources
- **Service**: ClusterIP service for webhook endpoint
- **RBAC**: ServiceAccount, ClusterRole, ClusterRoleBinding
- **ConfigMap**: Environment-based configuration
- **Secret**: TLS certificates (cert-manager integration)
- **MutatingWebhookConfiguration**: Admission webhook registration

### Configurable Values
```yaml
image:
  repository: ghcr.io/owner/archy
  tag: latest
  pullPolicy: IfNotPresent

replicas: 2

resources:
  requests:
    cpu: 100m
    memory: 128Mi
  limits:
    cpu: 500m
    memory: 256Mi

config:
  defaultArch: amd64
  logLevel: info
  cacheTimeout: 300s

tls:
  certManager: true
  issuer: selfsigned-issuer
```

### GitHub Actions Integration
- **Chart Linting**: Helm lint validation
- **Chart Testing**: Install/upgrade/rollback tests
- **OCI Publishing**: Push to GitHub Container Registry
- **Release Automation**: Tag-based chart versioning

## Comprehensive Testing Requirements

### Unit Test Categories
- **Configuration Tests**: Invalid env vars, missing required fields, type conversions
- **Registry Client Tests**: Network failures, malformed responses, authentication errors
- **Manifest Parser Tests**: Invalid JSON, missing fields, unsupported schemas
- **Cache Tests**: Concurrent access, TTL expiration, memory pressure, eviction policies
- **Mutation Logic Tests**: Complex pod specs, existing selectors, invalid architectures
- **HTTP Handler Tests**: Malformed requests, invalid certificates, timeout scenarios

### Integration Test Scenarios
- **End-to-End Webhook**: Real Kubernetes API server with admission controller
- **Registry Integration**: Live registry calls with rate limiting and failures
- **TLS Certificate Rotation**: Dynamic certificate updates during operation
- **Multi-Registry Fallback**: Primary registry failure with secondary success
- **Concurrent Request Handling**: 1000+ simultaneous webhook requests
- **Memory Leak Detection**: Long-running tests with heap profiling

### Chaos Engineering Tests
- **Network Partitions**: Registry unreachable during pod creation
- **DNS Failures**: Registry hostname resolution failures
- **Certificate Expiry**: TLS certificate expiration during operation
- **Memory Exhaustion**: Cache growth beyond available memory
- **CPU Starvation**: High load with limited CPU resources
- **Disk Full**: Log file growth consuming all disk space

### Security Penetration Tests
- **TLS Downgrade Attacks**: Force HTTP instead of HTTPS
- **Certificate Validation Bypass**: Invalid/expired certificate acceptance
- **Injection Attacks**: Malicious image names and registry URLs
- **Resource Exhaustion**: DoS via excessive webhook requests
- **Privilege Escalation**: Attempt to access unauthorized Kubernetes resources
- **Data Exfiltration**: Attempt to extract sensitive configuration data

### Property-Based Testing
- **Image Name Fuzzing**: Random valid/invalid image reference formats
- **Manifest Fuzzing**: Random JSON structures for registry responses
- **Pod Spec Fuzzing**: Random Kubernetes pod specifications
- **Configuration Fuzzing**: Random environment variable combinations

## Code Quality and CI/CD Pipeline

### Linting Configuration
- **golangci-lint**: Comprehensive Go linting with strict rules
- **Enabled Linters**: gofmt, goimports, govet, staticcheck, gosec, errcheck, ineffassign, misspell
- **Custom Rules**: Enforce 12-factor app patterns, security best practices
- **Performance Linters**: Check for memory leaks, inefficient algorithms
- **Documentation Linters**: Ensure all public functions have comments

### GitHub Actions Workflows
- **PR Workflow**: Triggered on pull requests
  - Run all tests with coverage reporting
  - Execute linting with zero tolerance for warnings
  - Perform security scanning
  - Validate Helm charts
  - Block merge if any check fails
- **Main Branch Workflow**: Triggered on merge to main
  - Run full test suite including chaos tests
  - Build multi-arch container images
  - Generate semantic version using conventional commits
  - Create GitHub release with changelog
  - Publish container images and Helm charts
  - Deploy to staging environment for validation

### Branch Protection Rules
- **Required Status Checks**: All CI checks must pass
- **Required Reviews**: At least one code review required
- **Dismiss Stale Reviews**: Re-review required after new commits
- **Restrict Push**: Only allow merges through PRs
- **Linear History**: Require merge commits or squash merging

### Semantic Release Configuration
- **Conventional Commits**: Enforce commit message format
- **Automatic Versioning**: Generate versions based on commit types
- **Changelog Generation**: Auto-generate release notes
- **Asset Publishing**: Attach binaries and charts to releases

## Security Considerations
- TLS-only communication with proper certificate validation
- Minimal RBAC permissions (only pod read access)
- No persistent storage of sensitive data
- Registry credentials via Kubernetes secrets
- Fail-open design prevents availability impact