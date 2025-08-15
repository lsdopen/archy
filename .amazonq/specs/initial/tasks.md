# Archy - Test-Driven Implementation Tasks

## Phase 1: Project Setup and Test Infrastructure

### Task 1.1: Initialize Go Module and Test Framework
- [x] Create `go.mod` with module name and test dependencies
- [x] Create basic project structure directories including test directories
- [x] Create `Makefile` with test, coverage, lint, and build targets
- [x] Set up test utilities and mocking framework
- [x] Configure code coverage reporting (minimum 100%)
- [x] Create `.golangci.yml` with strict linting configuration
- [x] Set up pre-commit hooks for linting and testing

### Task 1.2: Configuration Tests (Write Tests First)
- [x] Create `internal/config/config_test.go` with comprehensive test cases:
  - [x] Test missing required environment variables
  - [x] Test invalid data type conversions
  - [x] Test boundary values (empty strings, max integers)
  - [x] Test malformed URLs and invalid formats
  - [x] Test concurrent configuration loading
  - [x] Test configuration validation edge cases
- [x] Create `internal/config/config.go` to make tests pass
- [x] Verify 100% test coverage for configuration package

### Task 1.3: HTTP Server Tests (Write Tests First)
- [x] Create `cmd/webhook/main_test.go` with server tests:
  - [x] Test server startup with invalid TLS certificates
  - [x] Test graceful shutdown with active connections
  - [x] Test health endpoints under load
  - [x] Test server behavior with no available ports
  - [x] Test signal handling (SIGTERM, SIGINT)
  - [x] Test server panic recovery
- [x] Create `cmd/webhook/main.go` to make tests pass
- [x] Verify server handles all tested failure scenarios

## Phase 2: Webhook Framework Tests (Test-First)

### Task 2.1: TLS and HTTP Server Tests (Write Tests First)
- [x] Create `internal/webhook/server_test.go` with exhaustive tests:
  - [x] Test TLS certificate loading from various sources
  - [x] Test expired/invalid certificate handling
  - [x] Test certificate rotation during runtime
  - [x] Test HTTP timeout scenarios (read, write, idle)
  - [x] Test middleware chain execution order
  - [x] Test server shutdown with pending requests
  - [x] Test concurrent connection handling
  - [x] Test memory leaks with long-running connections
- [x] Create `internal/webhook/server.go` to make tests pass

### Task 2.2: Admission Handler Tests (Write Tests First)
- [x] Create `internal/webhook/handler_test.go` with comprehensive tests:
  - [x] Test malformed AdmissionReview JSON
  - [x] Test missing required fields in admission request
  - [x] Test invalid Kubernetes API versions
  - [x] Test oversized request payloads
  - [x] Test concurrent request processing
  - [x] Test request timeout handling
  - [x] Test admission response serialization errors
  - [x] Test webhook failure policy enforcement
  - [x] Test request tracing and correlation IDs
- [x] Create `internal/webhook/handler.go` to make tests pass

### Task 2.3: Pod Mutation Tests (Write Tests First)
- [x] Create `internal/webhook/mutator_test.go` with edge case tests:
  - [x] Test pods with no containers
  - [x] Test pods with init containers only
  - [x] Test pods with existing architecture selectors
  - [x] Test pods with conflicting node selectors
  - [x] Test pods with invalid image references
  - [x] Test pods with empty image names
  - [x] Test mutation of system pods (kube-system)
  - [x] Test concurrent mutation requests
  - [x] Test mutation rollback scenarios
- [x] Create `internal/webhook/mutator.go` to make tests pass

## Phase 3: Architecture Detection Tests (Test-First)

### Task 3.1: Registry Client Interface Tests (Write Tests First)
- [x] Create `pkg/types/types_test.go` with interface compliance tests:
  - [x] Test interface method signatures
  - [x] Test error handling contracts
  - [x] Test timeout behavior requirements
  - [x] Test concurrent access patterns
- [x] Create `internal/registry/client_test.go` with factory tests:
  - [x] Test client factory with invalid registry URLs
  - [x] Test client factory with unsupported registry types
  - [x] Test client factory with network failures
  - [x] Test client factory with authentication failures
- [x] Create interfaces and factory to make tests pass

### Task 3.2: Docker Hub Client Tests (Write Tests First)
- [x] Create `internal/registry/dockerhub_test.go` with failure tests:
  - [x] Test API rate limiting responses (429)
  - [x] Test network timeouts and retries
  - [x] Test malformed JSON responses
  - [x] Test authentication token expiry
  - [x] Test private repository access denied
  - [x] Test non-existent image handling
  - [x] Test registry API version changes
  - [x] Test concurrent API calls
  - [x] Test memory usage with large manifests
- [x] Create `internal/registry/dockerhub.go` to make tests pass

### Task 3.3: Manifest Parsing Tests (Write Tests First)
- [x] Create `internal/registry/manifest_test.go` with parsing tests:
  - [x] Test invalid JSON manifest structures
  - [x] Test missing required manifest fields
  - [x] Test unsupported manifest schema versions
  - [x] Test manifest with no architecture information
  - [x] Test manifest with unknown architectures
  - [x] Test manifest list with mixed schema versions
  - [x] Test extremely large manifest files
  - [x] Test manifest with circular references
  - [x] Test concurrent manifest parsing
- [x] Create `internal/registry/manifest.go` to make tests pass

### Task 3.4: Cache Tests (Write Tests First)
- [x] Create `internal/cache/memory_test.go` with stress tests:
  - [x] Test cache under memory pressure
  - [x] Test concurrent read/write operations (race conditions)
  - [x] Test TTL expiration edge cases
  - [x] Test LRU eviction with rapid insertions
  - [x] Test cache behavior during garbage collection
  - [x] Test cache statistics accuracy
  - [x] Test cache with zero/negative TTL values
  - [x] Test cache key collision scenarios
  - [x] Test cache persistence across restarts
- [x] Create `internal/cache/memory.go` to make tests pass

## Phase 4: Enhanced Features

### Task 4.1: Registry Credential Management (Write Tests First)
- [ ] Create `internal/credentials/resolver_test.go` with credential resolution tests:
  - [ ] Test pod imagePullSecrets extraction and parsing
  - [ ] Test service account imagePullSecrets fallback
  - [ ] Test dockerconfigjson secret format parsing
  - [ ] Test credential caching and TTL expiration
  - [ ] Test concurrent credential resolution
  - [ ] Test missing/invalid secret handling
  - [ ] Test registry URL matching logic
- [ ] Create `internal/credentials/resolver.go` to make tests pass
- [ ] Update RBAC templates to include secrets and serviceaccounts permissions
- [ ] Integrate credential resolver into registry clients

### Task 4.2: Advanced Mutation Logic with Credentials
- [x] Update mutator with architecture detection
- [x] Implement multi-arch selection strategy
- [x] Add fallback mechanisms
- [x] Handle edge cases and errors
- [ ] Integrate credential resolver into mutator
- [ ] Add credential-aware registry client selection
- [ ] Test private registry image architecture detection

### Task 4.3: Observability
- [x] Create `internal/metrics/prometheus.go`
- [x] Add mutation counters and timing metrics
- [x] Implement structured logging
- [x] Add request tracing
- [x] Integrate metrics into mutator and webhook server
- [x] Add architecture detection with cache integration

## Phase 5: Kubernetes Deployment

### Task 5.1: Helm Chart Only Deployment
- [x] Removed raw Kubernetes manifests (deploy folder)
- [x] Use Helm chart as single deployment method
- [x] Ensure no raw manifests exist to prevent confusion

### Task 5.2: Helm Chart
- [x] Create `chart/Chart.yaml` with metadata
- [x] Create `chart/values.yaml` with defaults
- [x] Create `chart/templates/deployment.yaml`
- [x] Create `chart/templates/service.yaml`
- [x] Create `chart/templates/rbac.yaml`
- [x] Create `chart/templates/secret.yaml` (cert-manager integration)
- [x] Create `chart/templates/webhook.yaml`
- [x] Create `chart/templates/servicemonitor.yaml` (Prometheus monitoring)
- [x] Create `chart/templates/prometheusrule.yaml` (alerting rules)
- [x] Create `chart/templates/_helpers.tpl`

## Phase 6: Build and CI/CD

### Task 6.1: Linting Configuration (Write Tests First)
- [x] Create `.golangci.yml` with comprehensive linter configuration:
  - [x] Enable all security linters (gosec, gas)
  - [x] Enable performance linters (ineffassign, prealloc)
  - [x] Enable style linters (gofmt, goimports, misspell)
  - [x] Enable complexity linters (gocyclo, gocognit)
  - [x] Configure custom rules for 12-factor compliance
- [x] Create linting tests to verify configuration works
- [x] Add Makefile targets for linting

### Task 6.2: Container Build Tests (Write Tests First)
- [x] Create container security tests:
  - [x] Test container for known vulnerabilities
  - [x] Test container runs as non-root user
  - [x] Test container has no shell access
  - [x] Test container filesystem is read-only
  - [x] Test container resource limits enforcement
- [x] Create multi-arch build tests:
  - [x] Test binary compatibility across architectures
  - [x] Test container startup on different platforms
- [x] Create `Containerfile` to pass security tests

### Task 6.3: GitHub Actions - PR Workflow
- [x] Create `.github/workflows/pr.yaml` with comprehensive checks:
  - [x] Run golangci-lint with zero tolerance
  - [x] Execute full test suite with coverage reporting
  - [x] Perform security scanning with gosec
  - [x] Validate Helm chart templates
  - [x] Check conventional commit format
  - [x] Block merge if any check fails

### Task 6.4: GitHub Actions - Main Branch Workflow
- [x] Create `.github/workflows/main.yaml` for releases:
  - [x] Run complete test suite including chaos tests
  - [x] Build and test multi-arch container images
  - [x] Use semantic-release for automatic versioning
  - [x] Generate changelog from conventional commits
  - [x] Publish container images to GHCR
  - [x] Publish Helm chart to OCI registry
  - [x] Create GitHub release with assets

### Task 6.5: Branch Protection and Semantic Release
- [x] Configure GitHub branch protection rules:
  - [x] Require PR reviews and status checks
  - [x] Dismiss stale reviews on new commits
  - [x] Restrict direct pushes to main branch
- [x] Set up semantic-release configuration:
  - [x] Configure conventional commit parsing
  - [x] Set up automatic version bumping
  - [x] Configure changelog generation
  - [x] Set up asset publishing

## Phase 7: Testing

### Task 7.1: Unit Tests
- [ ] Add tests for `internal/config/`
- [ ] Add tests for `internal/registry/`
- [ ] Add tests for `internal/cache/`
- [ ] Add tests for `internal/webhook/mutator.go`

### Task 7.2: Integration Tests
- [ ] Create webhook server integration tests
- [ ] Add registry client integration tests
- [ ] Test end-to-end mutation flow
- [ ] Add performance benchmarks

### Task 7.3: Failure Scenario Tests
- [ ] Test registry timeout scenarios
- [ ] Test invalid manifest responses
- [ ] Test network failure handling
- [ ] Test malformed admission requests

### Task 7.4: Security Tests
- [ ] Test TLS certificate validation
- [ ] Test RBAC permissions
- [ ] Test input validation
- [ ] Test credential handling

## Phase 8: Documentation

### Task 8.1: Code Documentation
- [ ] Add Go doc comments to all public functions
- [ ] Create package-level documentation
- [ ] Add usage examples

### Task 8.2: Deployment Documentation
- [ ] Create installation guide
- [ ] Document Helm chart values
- [ ] Add troubleshooting guide
- [ ] Create configuration reference

### Task 8.3: Final System Validation
- [ ] Run complete end-to-end test suite
- [ ] Verify 100% code coverage across all packages
- [ ] Validate all linting rules pass with zero warnings
- [ ] Confirm all security scans pass
- [ ] Validate all security requirements are tested
- [ ] Confirm all failure scenarios are covered
- [ ] Execute full chaos engineering test suite
- [ ] Validate performance meets all SLA requirements
- [ ] Test semantic release workflow
- [ ] Validate branch protection rules work correctly

### Task 8.4: Development Documentation
- [ ] Update README.md with comprehensive testing and linting approach
- [ ] Add contributing guidelines emphasizing test-first development
- [ ] Document build process including linting and test execution
- [ ] Add development setup guide with linting requirements
- [ ] Create code review checklist focusing on test quality and linting
- [ ] Document conventional commit format requirements
- [ ] Add semantic release workflow documentation