# OCI Helm Repository Example

This example demonstrates how to use OCI-based Helm repositories with Helm Operator.

## Prerequisites

- Helm Operator v0.2.3+
- Access to an OCI-compatible registry (GHCR, ACR, ECR, Harbor, etc.)

## Example 1: Public OCI Repository

```yaml
# oci-public-repository.yaml
apiVersion: helm-operator.ketches.cn/v1alpha1
kind: HelmRepository
metadata:
  name: ghcr-public
  namespace: default
spec:
  url: "oci://ghcr.io/helm/charts"
  type: "oci"
  interval: "1h"
  timeout: "10m"
  suspend: false
```

## Example 2: Private OCI Repository with Authentication

### Step 1: Create Authentication Secret

```bash
kubectl create secret generic oci-registry-auth \
  --from-literal=username=your-username \
  --from-literal=password=your-token \
  --namespace default
```

### Step 2: Create HelmRepository

```yaml
# oci-private-repository.yaml
apiVersion: helm-operator.ketches.cn/v1alpha1
kind: HelmRepository
metadata:
  name: private-oci-repo
  namespace: default
spec:
  url: "oci://registry.example.com/charts"
  type: "oci"
  interval: "2h"
  timeout: "15m"
  auth:
    basic:
      secretRef:
        name: oci-registry-auth
        namespace: default
  suspend: false
```

## Example 3: Deploy Chart from OCI Repository

```yaml
# oci-helm-release.yaml
apiVersion: helm-operator.ketches.cn/v1alpha1
kind: HelmRelease
metadata:
  name: nginx-oci
  namespace: default
spec:
  chart:
    name: nginx
    version: "1.0.0"
    # Direct OCI reference
    ociRepository: "oci://ghcr.io/helm/charts/nginx"
  release:
    name: nginx-oci
    namespace: default
    createNamespace: true
  values: |
    replicaCount: 2
    service:
      type: ClusterIP
      port: 80
    resources:
      limits:
        cpu: 200m
        memory: 256Mi
      requests:
        cpu: 100m
        memory: 128Mi
  install:
    timeout: "10m"
    wait: true
    waitForJobs: true
  upgrade:
    timeout: "10m"
    wait: true
    cleanupOnFail: true
```

## Apply the Examples

```bash
# Create the repository
kubectl apply -f oci-public-repository.yaml

# Wait for repository to be ready
kubectl wait --for=condition=Ready helmrepository/ghcr-public --timeout=300s

# Deploy the release
kubectl apply -f oci-helm-release.yaml

# Check status
kubectl get helmrepository
kubectl get helmrelease
kubectl describe helmrelease nginx-oci
```

## OCI Repository Providers

### GitHub Container Registry (GHCR)

```yaml
spec:
  url: "oci://ghcr.io/organization/charts"
  type: "oci"
```

### Azure Container Registry (ACR)

```yaml
spec:
  url: "oci://myregistry.azurecr.io/helm"
  type: "oci"
```

### Amazon ECR

```yaml
spec:
  url: "oci://123456789012.dkr.ecr.us-east-1.amazonaws.com/helm"
  type: "oci"
```

### Google Artifact Registry (GAR)

```yaml
spec:
  url: "oci://us-central1-docker.pkg.dev/project-id/helm-charts"
  type: "oci"
```

### Harbor

```yaml
spec:
  url: "oci://harbor.example.com/helm-charts"
  type: "oci"
```

## Differences from Traditional Helm Repositories

| Feature | Traditional Helm | OCI |
|---------|------------------|-----|
| **URL Format** | `https://charts.example.com` | `oci://registry.example.com/charts` |
| **Index File** | Required (`index.yaml`) | Not used |
| **Chart List** | Can list all charts | Cannot list (must know chart name) |
| **Authentication** | Basic Auth, TLS | Docker Registry Auth |
| **Bandwidth** | Downloads full index | Pull only needed charts |

## Notes

- OCI repositories don't support listing all available charts
- You must know the exact chart name and version
- Chart values ConfigMaps are not automatically generated for OCI repos
- Use `ociRepository` field in HelmRelease for direct OCI references

## Troubleshooting

### Authentication Issues

```bash
# Test OCI registry login manually
helm registry login oci://registry.example.com

# Verify secret contents
kubectl get secret oci-registry-auth -o yaml
```

### Chart Pull Failures

```bash
# Test chart pull manually
helm pull oci://registry.example.com/charts/mychart --version 1.0.0

# Check HelmRelease status
kubectl describe helmrelease nginx-oci
kubectl get events --sort-by='.lastTimestamp'
```

## Additional Resources

- [OCI Repository Guide](../docs/oci-repository-guide.md)
- [Helm OCI Documentation](https://helm.sh/docs/topics/registries/)

---

**Version**: v0.2.3  
**Last Updated**: 2026-02-11
