# Product Overview

Archy is a Kubernetes mutating admission webhook that automatically ensures Pods are scheduled on nodes with compatible architectures in multi-architecture clusters.

## Core Functionality

- **Architecture Detection**: Inspects container image manifests to determine supported platforms (amd64, arm64, etc.)
- **Automatic Pod Mutation**: Adds `kubernetes.io/arch` nodeSelector to Pods when a single common architecture is found
- **Multi-Arch Support**: Allows Kubernetes scheduler to handle placement when images support multiple architectures
- **Private Registry Support**: Authenticates with private registries using Pod's imagePullSecrets and ServiceAccount credentials
- **Safety First**: Rejects Pods when images have no common supported architecture

## Key Behaviors

- Skips Pods that already have a nodeSelector defined
- Fails closed (rejects Pod) if architecture inspection fails or no common platform exists
- Allows scheduler flexibility when multiple common architectures are available
- Excludes system namespaces (kube-system, kube-public) and self (archy-webhook) from processing