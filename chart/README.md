# Archy Helm Chart

A Kubernetes mutating admission webhook that automatically ensures Pods are scheduled on nodes with compatible architectures in multi-architecture clusters.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+

## Installing the Chart

### From GitHub Container Registry (Recommended)

Install the latest version:

```bash
helm install archy oci://ghcr.io/lsdopen/archy/charts/archy
```

Install a specific version:

```bash
helm install archy oci://ghcr.io/lsdopen/archy/charts/archy --version 0.1.0
```

Install with custom values:

```bash
helm install archy oci://ghcr.io/lsdopen/archy/charts/archy --values values-production.yaml
```

### From Source

To install the chart from source with the release name `archy`:

```bash
helm install archy ./chart
```

Or with custom values:

```bash
helm install archy ./chart -f values-production.yaml
```

## Uninstalling the Chart

To uninstall/delete the `archy` deployment:

```bash
helm delete archy
```

## Configuration

The chart comes with sensible defaults and requires no configuration for basic deployment. All parameters are optional and can be customized as needed.

### Default Configuration

The chart automatically configures:
- **Image**: `ghcr.io/lsdopen/archy:1.0.0` with `IfNotPresent` pull policy
- **Service**: ClusterIP on port 443
- **Webhook**: 5-second timeout with "Fail" policy
- **Certificates**: Helm-generated self-signed certificates (1-year validity)

### Certificate-Specific Required Parameters

#### When using `certificates.method: "helm"` (default)
| Parameter | Description | Default | Type |
|-----------|-------------|---------|------|
| `certificates.helm.duration` | Certificate validity duration | `"8760h"` | `string` |
| `certificates.helm.subject.organizationName` | Certificate organization name | `"Archy Webhook"` | `string` |

#### When using `certificates.method: "cert-manager"`
| Parameter | Description | Type |
|-----------|-------------|------|
| `certificates.certManager.issuer.name` | cert-manager issuer name | `string` |
| `certificates.certManager.issuer.kind` | cert-manager issuer kind (Issuer/ClusterIssuer) | `string` |

#### When using `certificates.method: "external"`
| Parameter | Description | Type |
|-----------|-------------|------|
| `certificates.external.secretName` | Secret containing TLS certificates | `string` |
| `certificates.external.certFile` | Certificate file name in secret | `string` |
| `certificates.external.keyFile` | Private key file name in secret | `string` |
| `certificates.external.caBundle` | Base64 encoded CA bundle | `string` |

### Optional Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas | `1` |
| `imagePullSecrets` | Image pull secrets | `[]` |
| `serviceAccount.create` | Create service account | `true` |
| `serviceAccount.annotations` | Service account annotations | `{}` |
| `serviceAccount.name` | Service account name | `""` |
| `podAnnotations` | Pod annotations | `{}` |
| `podSecurityContext` | Pod security context | `{}` |
| `securityContext` | Container security context | `{}` |
| `resources` | Resource limits and requests | `{}` |
| `nodeSelector` | Node selector | `{}` |
| `tolerations` | Tolerations | `[]` |
| `affinity` | Affinity rules | `{}` |
| `topologySpreadConstraints` | Topology spread constraints | `[]` |
| `certificates.helm.subject.organizationalUnit` | Certificate organizational unit | `""` |
| `certificates.helm.subject.country` | Certificate country code | `""` |
| `certificates.helm.subject.province` | Certificate province/state | `""` |
| `certificates.helm.subject.locality` | Certificate city/locality | `""` |
| `certificates.certManager.duration` | Certificate duration (cert-manager) | `""` |
| `certificates.certManager.renewBefore` | Certificate renewal time (cert-manager) | `""` |
| `webhook.objectSelector` | Additional object selector expressions | `{}` |
| `webhook.namespaceSelector` | Additional namespace selector expressions | `{}` |
| `labels` | Additional labels for all resources | `{}` |
| `annotations` | Additional annotations for all resources | `{}` |

## Certificate Management

The Archy webhook requires TLS certificates to function properly. The chart supports three certificate management methods:

### 1. Helm-Generated Certificates (Recommended for Development)

Helm automatically generates self-signed certificates during installation:

```yaml
certificates:
  method: "helm"
  helm:
    duration: "8760h" # 1 year
    subject:
      organizationName: "Your Organization"
```

### 2. cert-manager Integration (Recommended for Production)

Use cert-manager to automatically provision and renew certificates:

```yaml
certificates:
  method: "cert-manager"
  certManager:
    issuer:
      name: "letsencrypt-prod"
      kind: "ClusterIssuer"
    duration: "2160h" # 90 days
    renewBefore: "720h" # 30 days
```

### 3. External Certificate Management

Bring your own certificates by creating a secret manually:

```bash
# Generate certificates
./scripts/gen-certs.sh

# Create secret
kubectl create secret tls archy-webhook-certs \
  --cert=certs/tls.crt \
  --key=certs/tls.key

# Configure values
certificates:
  method: "external"
  external:
    secretName: "archy-webhook-certs"
    certFile: "tls.crt"
    keyFile: "tls.key"
    caBundle: "$(cat certs/ca.crt | base64 | tr -d '\n')"
```

## Example Configurations

### Basic Configuration

No configuration required! Install with defaults:

```bash
helm install archy ./chart
```

Or customize as needed:

```yaml
# Override image (optional)
image:
  repository: "your-registry/archy-webhook"
  tag: "v2.0.0"

# Customize webhook behavior (optional)
webhook:
  timeoutSeconds: 10
  failurePolicy: "Ignore"
```

### High Availability Configuration

```yaml
replicaCount: 3

resources:
  limits:
    cpu: "200m"
    memory: "256Mi"
  requests:
    cpu: "100m"
    memory: "128Mi"

affinity:
  podAntiAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
    - labelSelector:
        matchExpressions:
        - key: app.kubernetes.io/name
          operator: In
          values:
          - archy
      topologyKey: kubernetes.io/hostname

topologySpreadConstraints:
  - maxSkew: 1
    topologyKey: topology.kubernetes.io/zone
    whenUnsatisfiable: DoNotSchedule
    labelSelector:
      matchLabels:
        app.kubernetes.io/name: archy
```

## Troubleshooting

### Common Issues

1. **Webhook not intercepting pods**: Check that the MutatingWebhookConfiguration is properly configured and the service is accessible.

2. **Certificate errors**: Ensure the TLS certificates are valid and the CA bundle matches the certificate authority.

3. **Permission errors**: Verify the service account has the necessary RBAC permissions to access secrets in target namespaces.

### Debugging Commands

```bash
# Check webhook configuration
kubectl get mutatingwebhookconfiguration archy

# Check webhook pods
kubectl get pods -l app.kubernetes.io/name=archy

# View webhook logs
kubectl logs -l app.kubernetes.io/name=archy -f

# Test webhook connectivity
kubectl port-forward svc/archy 8443:443
curl -k https://localhost:8443/healthz
```

## Contributing

Please read the main project's CONTRIBUTING.md for details on our code of conduct and the process for submitting pull requests.