# Contributing to Archy

## Development Philosophy

Archy follows a **Test-Driven Development (TDD)** approach with strict quality standards:

- **Tests First**: Always write failing tests before implementing functionality
- **100% Coverage**: Every line of code must be covered by tests
- **Zero Warnings**: All linting rules must pass without exceptions
- **Security First**: All security scans must pass
- **Fail-Safe Design**: Always fail open to prevent blocking pod creation

## Getting Started

### Prerequisites

- Go 1.21+
- golangci-lint
- make
- Helm 3.12+
- Docker (for container builds)

### Setup Development Environment

```bash
# Clone the repository
git clone https://github.com/lsdopen/archy.git
cd archy

# Install dependencies
go mod tidy

# Verify setup
make test
make lint
```

## Development Workflow

### 1. Create Feature Branch

```bash
git checkout -b feat/your-feature-name
```

### 2. Write Tests First (TDD)

Before implementing any functionality, write comprehensive tests:

```bash
# Create test file first
touch internal/yourpackage/yourfile_test.go

# Write failing tests that describe expected behavior
# Include edge cases, error conditions, and concurrent scenarios
```

### 3. Implement Code

Write minimal code to make tests pass:

```bash
# Create implementation file
touch internal/yourpackage/yourfile.go

# Implement functionality to satisfy tests
```

### 4. Verify Quality Standards

```bash
# Run tests with coverage (must be 100%)
make test-coverage

# Run linting (must pass with zero warnings)
make lint

# Run security scans
make security-scan

# Build and test container
make container
```

### 5. Commit Changes

Use conventional commit format for semantic versioning:

```bash
git add .
git commit -m "feat(scope): add new functionality

Detailed description of changes made.

Closes #123"
```

## Code Quality Requirements

### Test Coverage

- **100% Required**: No exceptions for production code
- **Edge Cases**: Test all boundary conditions and error scenarios
- **Concurrency**: Test race conditions and concurrent access
- **Integration**: Test end-to-end functionality
- **Performance**: Include benchmarks for critical paths

### Linting Standards

All golangci-lint rules must pass:

```bash
# Check linting
make lint

# Auto-fix issues where possible
make lint-fix
```

Key linting categories:
- **Security**: gosec, gas
- **Performance**: ineffassign, prealloc
- **Style**: gofmt, goimports, misspell
- **Complexity**: gocyclo, gocognit
- **Correctness**: govet, staticcheck

### Security Requirements

- **No Hardcoded Secrets**: Use environment variables or Kubernetes secrets
- **Input Validation**: Validate all external inputs
- **Fail-Safe Design**: Always fail open to prevent availability impact
- **Minimal Permissions**: Use least-privilege RBAC
- **TLS Only**: All communication must use TLS

## Testing Guidelines

### Unit Tests

```go
func TestFunction_EdgeCase(t *testing.T) {
    // Arrange
    input := createTestInput()
    
    // Act
    result, err := Function(input)
    
    // Assert
    require.NoError(t, err)
    assert.Equal(t, expected, result)
}
```

### Integration Tests

```go
func TestWebhook_EndToEnd(t *testing.T) {
    // Setup test server
    server := setupTestServer()
    defer server.Close()
    
    // Test complete webhook flow
    response := sendAdmissionRequest(server, testPod)
    
    // Verify mutation applied correctly
    assertNodeSelectorAdded(t, response)
}
```

### Failure Scenario Tests

```go
func TestFunction_NetworkFailure(t *testing.T) {
    // Simulate network failure
    client := &failingClient{}
    
    // Function should handle gracefully
    result, err := Function(client)
    
    // Should fail open with default behavior
    assert.NoError(t, err)
    assert.Equal(t, defaultValue, result)
}
```

## Conventional Commits

### Format

```
type(scope): description

[optional body]

[optional footer]
```

### Types

- **feat**: New feature
- **fix**: Bug fix
- **docs**: Documentation changes
- **style**: Code style changes (formatting, etc.)
- **refactor**: Code refactoring
- **test**: Adding or updating tests
- **chore**: Maintenance tasks

### Examples

```bash
# New feature
git commit -m "feat(webhook): add support for private registry credentials"

# Bug fix
git commit -m "fix(cache): resolve race condition in TTL cleanup"

# Documentation
git commit -m "docs(readme): update installation instructions"

# Breaking change
git commit -m "feat(api): change webhook endpoint path

BREAKING CHANGE: webhook endpoint moved from /webhook to /mutate"
```

## Pull Request Process

### 1. Pre-PR Checklist

- [ ] All tests pass with 100% coverage
- [ ] All linting rules pass with zero warnings
- [ ] Security scans pass without issues
- [ ] Documentation updated if needed
- [ ] Conventional commit format used
- [ ] Branch is up to date with main

### 2. PR Description Template

```markdown
## Description
Brief description of changes made.

## Type of Change
- [ ] Bug fix (non-breaking change which fixes an issue)
- [ ] New feature (non-breaking change which adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update

## Testing
- [ ] Unit tests added/updated
- [ ] Integration tests added/updated
- [ ] All tests pass with 100% coverage
- [ ] Manual testing completed

## Checklist
- [ ] Code follows TDD approach (tests written first)
- [ ] Self-review completed
- [ ] Code is well-commented
- [ ] Documentation updated
- [ ] No breaking changes (or clearly documented)
```

### 3. Review Process

- **Required Reviews**: At least one maintainer approval
- **Automated Checks**: All CI checks must pass
- **Security Review**: For security-related changes
- **Performance Review**: For performance-critical changes

## Architecture Guidelines

### Package Structure

```
internal/
├── config/          # Configuration management
├── credentials/     # Registry credential resolution
├── cache/          # Architecture caching
├── metrics/        # Prometheus metrics
├── registry/       # Registry client implementations
└── webhook/        # Webhook server and handlers
```

### Design Principles

- **Single Responsibility**: Each package has one clear purpose
- **Dependency Injection**: Use interfaces for testability
- **Fail-Safe**: Always fail open to prevent blocking pods
- **12-Factor App**: Follow 12-factor app principles
- **Cloud Native**: Design for Kubernetes environments

### Error Handling

```go
// Always fail open for webhook operations
func (m *Mutator) detectArchitecture(image string) string {
    arch, err := m.registryClient.GetArchitecture(image)
    if err != nil {
        // Log error but don't fail the mutation
        log.Warnf("Failed to detect architecture for %s: %v", image, err)
        return m.defaultArch // Fail open with default
    }
    return arch
}
```

## Release Process

### Semantic Versioning

Versions are automatically generated based on conventional commits:

- **PATCH**: fix commits
- **MINOR**: feat commits  
- **MAJOR**: commits with BREAKING CHANGE footer

### Release Workflow

1. **Merge to Main**: PR merged triggers release workflow
2. **Version Calculation**: semantic-release analyzes commits
3. **Changelog Generation**: Automatic changelog from commits
4. **Container Build**: Multi-arch container images built
5. **Helm Chart**: Chart packaged and published
6. **GitHub Release**: Release created with assets

## Getting Help

- **Issues**: Create GitHub issue for bugs or feature requests
- **Discussions**: Use GitHub discussions for questions
- **Security**: Email security@lsdopen.com for security issues
- **Documentation**: Check README.md and inline code comments

## Code of Conduct

This project follows the [Contributor Covenant Code of Conduct](https://www.contributor-covenant.org/version/2/1/code_of_conduct/).