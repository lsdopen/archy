# Contributing to Archy

Thank you for your interest in contributing to Archy! We welcome contributions from the community to help make this project better.

## Code of Conduct

Please note that this project is released with a Contributor Code of Conduct. By participating in this project you agree to abide by its terms.

## How to Contribute

### Reporting Bugs

If you find a bug, please open an issue on GitHub. Include as much detail as possible:
- Steps to reproduce
- Expected behavior
- Actual behavior
- Logs or error messages
- Kubernetes version and environment details

### Suggesting Enhancements

We welcome ideas for new features or improvements. Please open an issue to discuss your idea before submitting a Pull Request.

### Submitting Pull Requests

1. Fork the repository.
2. Create a new branch for your feature or fix (`git checkout -b feature/amazing-feature`).
3. Make your changes.
4. Run tests to ensure everything is working (`go test ./...`).
5. Commit your changes (`git commit -m 'feat: add amazing feature'`).
6. Push to the branch (`git push origin feature/amazing-feature`).
7. Open a Pull Request.

## Development

### Prerequisites

- Go 1.21+
- Docker
- Kubernetes cluster (Kind, Minikube, or remote)
- [Tilt](https://tilt.dev/) (recommended for local development)

### Local Development

We use Tilt for rapid local development. Simply run:

```bash
tilt up
```

This will build the webhook, deploy it to your current Kubernetes context, and stream logs.

### Building Binaries

To build the binary for your local machine:

```bash
make build
```

To cross-compile:

```bash
make build-amd64
make build-arm64
```
