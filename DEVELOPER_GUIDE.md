# Helm Operator Developer Guide

> **Version**: v0.3.0

## Overview

Helm Operator is a Kubernetes controller for managing the lifecycle of Helm releases. It provides declarative Helm release management capabilities through Kubernetes Custom Resource Definitions (CRDs) with intelligent automation and advanced features.

## Architecture Overview

### Core Components

```txt
helm-operator/
├── api/v1alpha1/           # CRD definitions
│   ├── helmrelease_types.go    # HelmRelease resource definition
│   └── helmrepository_types.go # HelmRepository resource definition
├── internal/
│   ├── controller/         # Controller logic
│   │   ├── helmrelease_controller.go
│   │   └── helmrepository_controller.go
│   ├── helm/              # Helm client wrapper
│   │   ├── client.go
│   │   ├── release.go
│   │   └── repository.go
│   └── utils/             # Utility functions
├── deploy/                # Deployment configurations
├── samples/               # Sample resources
└── cmd/main.go            # Program entry point
```

### Resource Types

1. **HelmRepository**: Defines Helm repository configuration
2. **HelmRelease**: Defines Helm release configuration

## Development Environment Setup

### Prerequisites

- Go 1.21+
- Docker
- Kubernetes cluster (local or remote)
- kubectl
- Helm 3.x
- kubebuilder

### Install Dependencies

```bash
# Clone the project
git clone https://github.com/ketches/helm-operator.git
cd helm-operator

# Install dependencies
go mod download
```

### Build Project

```bash
# Build binary
make build

# Build Docker image
make docker-build

# Run tests
make test
```

## Core Concepts

### HelmRepository

HelmRepository defines the configuration information for Helm repositories and supports multiple types of repositories:

#### 1. OCI Registry (Recommended)

OCI (Open Container Initiative) registries are the recommended way to distribute Helm charts due to better performance, security, and standardization.

**Public OCI Registry:**

```yaml
apiVersion: helm-operator.ketches.cn/v1alpha1
kind: HelmRepository
metadata:
  name: ghcr-charts
  namespace: default
spec:
  url: "oci://ghcr.io/myorg/charts"
  type: "oci"
  interval: "1h"
  timeout: "10m"
  # Optimize resource usage
  valuesConfigMapPolicy: disabled  # Recommended
```

**Private OCI Registry with Authentication:**

```yaml
# Create Docker registry secret
apiVersion: v1
kind: Secret
metadata:
  name: oci-registry-auth
  namespace: default
type: kubernetes.io/dockerconfigjson
data:
  .dockerconfigjson: <base64-encoded-docker-config>
---
apiVersion: helm-operator.ketches.cn/v1alpha1
kind: HelmRepository
metadata:
  name: acr-private
  namespace: default
spec:
  url: "oci://myregistry.azurecr.io/helm"
  type: "oci"
  auth:
    secretRef:
      name: oci-registry-auth
  interval: "1h"
  valuesConfigMapPolicy: disabled
```

**OCI Registry Examples:**

```yaml
# GitHub Container Registry (GHCR)
spec:
  url: "oci://ghcr.io/myorg/charts"

# Azure Container Registry (ACR)
spec:
  url: "oci://myregistry.azurecr.io/helm"

# Google Artifact Registry (GAR)
spec:
  url: "oci://us-docker.pkg.dev/project-id/helm-charts"

# Amazon Elastic Container Registry (ECR)
spec:
  url: "oci://123456789012.dkr.ecr.us-east-1.amazonaws.com/helm-charts"

# Harbor
spec:
  url: "oci://harbor.example.com/library"
```

#### 2. Public HTTPS Repository

```yaml
apiVersion: helm-operator.ketches.cn/v1alpha1
kind: HelmRepository
metadata:
  name: bitnami
  namespace: default
spec:
  url: "https://charts.bitnami.com/bitnami"
  type: "helm"
  interval: "30m"
  timeout: "5m"
  suspend: false
  valuesConfigMapPolicy: disabled  # Recommended
```

#### 3. Private HTTPS Repository (with authentication)

```yaml
apiVersion: helm-operator.ketches.cn/v1alpha1
kind: HelmRepository
metadata:
  name: private-repo
  namespace: default
spec:
  url: "https://private.charts.example.com"
  type: "helm"
  interval: "1h"
  timeout: "10m"
  auth:
    basic:
      secretRef:
        name: "private-repo-auth"
        namespace: "default"
  suspend: false
```

#### 3. Private HTTP Repository (internal network)

```yaml
apiVersion: helm-operator.ketches.cn/v1alpha1
kind: HelmRepository
metadata:
  name: internal-repo
  namespace: default
spec:
  url: "http://internal-charts.company.local:8080"
  type: "helm"
  interval: "30m"
  timeout: "10m"
  auth:
    basic:
      secretRef:
        name: "internal-repo-auth"
        namespace: "default"
    tls:
      insecureSkipVerify: true
  suspend: false
```

**Key Field Descriptions:**

- `url`: Helm repository URL, supports `https://` and `http://` protocols
- `type`: Repository type (currently only supports `helm`, may add `oci` support in the future)
- `interval`: Synchronization interval
- `timeout`: Operation timeout
- `auth`: Authentication configuration (optional)
  - `basic`: Basic authentication (username/password)
  - `tls`: TLS configuration
- `suspend`: Whether to suspend synchronization

### HelmRelease

HelmRelease defines the configuration for Helm releases:

```yaml
apiVersion: helm-operator.ketches.cn/v1alpha1
kind: HelmRelease
metadata:
  name: nginx
  namespace: default
spec:
  chart:
    name: "nginx-charts"
    version: "0.1.0"
    repository:
      name: "nginx-charts"
      namespace: "default"
  release:
    name: "nginx"
    namespace: "default"
    createNamespace: true
  values: |
    replicaCount: 2
  install:
    timeout: "10m"
    wait: true
  upgrade:
    timeout: "10m"
    wait: true
  interval: "1h"
```

**Key Field Descriptions:**

- `chart`: Chart configuration (name, version, repository)
- `release`: Release configuration (name, namespace)
- `values`: Helm values configuration
- `install/upgrade`: Install/upgrade options
- `interval`: Reconciliation interval

## Development Guide

### Adding New Features

1. **Modify CRD definitions**

   ```bash
   # Edit API type definitions
   vim api/v1alpha1/helmrelease_types.go
   
   # Regenerate code
   make generate
   
   # Update CRDs
   make manifests
   ```

2. **Update controller logic**

   ```bash
   # Edit controller
   vim internal/controller/helmrelease_controller.go
   
   # Add business logic
   # Update reconciliation loop
   ```

3. **Add tests**

   ```bash
   # Add unit tests
   vim internal/controller/helmrelease_controller_test.go
   
   # Run tests
   make test
   ```

### Controller Development Pattern

#### Reconciliation Loop

```go
func (r *HelmReleaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // 1. Get resource
    release := &helmoperatorv1alpha1.HelmRelease{}
    if err := r.Get(ctx, req.NamespacedName, release); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // 2. Handle deletion logic
    if !release.DeletionTimestamp.IsZero() {
        return r.reconcileDelete(ctx, release)
    }

    // 3. Add Finalizer
    if !controllerutil.ContainsFinalizer(release, utils.HelmReleaseFinalizer) {
        controllerutil.AddFinalizer(release, utils.HelmReleaseFinalizer)
        return ctrl.Result{}, r.Update(ctx, release)
    }

    // 4. Execute main logic
    return r.reconcileNormal(ctx, release)
}
```

#### Status Management

```go
// Update status condition
condition := utils.NewReleaseReadyCondition(metav1.ConditionTrue, "InstallCompleted", "Release is ready")
meta.SetStatusCondition(&release.Status.Conditions, condition)

// Update status
return r.Status().Update(ctx, release)
```

### Using Helm Client

#### Create Helm Client

```go
helmClient, err := helm.NewClient("default")
if err != nil {
    return fmt.Errorf("failed to create helm client: %w", err)
}
```

#### Install Release

```go
installReq := &helm.InstallRequest{
    Name:            "my-release",
    Namespace:       "default",
    Chart:           "bitnami/nginx",
    Version:         "1.0.0",
    Values:          "replicaCount: 2",
    CreateNamespace: true,
    Wait:            true,
    Timeout:         10 * time.Minute,
}

releaseInfo, err := helmClient.InstallRelease(ctx, installReq)
```

#### Chart Reference Formats

Chart references support multiple formats:

1. **Repository reference**: `repository_name/chart_name`

   ```go
   chartRef := fmt.Sprintf("%s/%s", repoName, chartName)
   // Example: "bitnami/nginx", "private-repo/myapp"
   ```

2. **Direct URL**: Use `repositoryURL` field

   ```yaml
   spec:
     chart:
       name: "nginx"
       repositoryURL: "https://charts.bitnami.com/bitnami"
   ```

3. **Local Chart**: Use chart name directly

   ```yaml
   spec:
     chart:
       name: "./local-chart"
   ```

#### Supported Repository Types

| Repository Type | URL Format | Example | Use Case |
|----------------|------------|---------|----------|
| Public HTTPS | `https://` | `https://charts.bitnami.com/bitnami` | Public Helm repositories |
| Private HTTPS | `https://` | `https://private.charts.example.com` | Enterprise private repositories |
| Internal HTTP | `http://` | `http://charts.internal:8080` | Internal private repositories |

#### Authentication Configuration

For private repositories, multiple authentication methods are supported:

1. **Basic Authentication (username/password)**

   ```yaml
   auth:
     basic:
       secretRef:
         name: "repo-credentials"
         namespace: "default"
   ```

2. **TLS Configuration**

   ```yaml
   auth:
     tls:
       insecureSkipVerify: true  # Skip certificate verification (for HTTP repositories)
       secretRef:
         name: "tls-config"
         namespace: "default"
   ```

3. **Authentication Secret Format**

   ```yaml
   apiVersion: v1
   kind: Secret
   metadata:
     name: repo-credentials
   type: Opaque
   data:
     username: <base64-encoded-username>
     password: <base64-encoded-password>
   ```

## Testing Guide

### Unit Tests

```bash
# Run all tests
make test

# Run tests for specific packages
go test ./internal/controller/...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Integration Tests

```bash
# Start test environment
make test-integration

# Run end-to-end tests
make test-e2e
```

### Local Debugging

1. **Run controller**

   ```bash
   # Set KUBECONFIG
   export KUBECONFIG=~/.kube/config
   
   # Run controller
   go run cmd/main.go
   ```

2. **Apply test resources**

   ```bash
   # Apply HelmRepository
   kubectl apply -f samples/helm_repository.yaml
   
   # Apply HelmRelease
   kubectl apply -f samples/helm_release.yaml
   ```

3. **View logs**

   ```bash
   # View controller logs
   kubectl logs -f deployment/helm-operator -n ketches
   ```

## Troubleshooting

### Common Issues

1. **Chart reference error**

   ```txt
   Error: non-absolute URLs should be in form of repo_name/path_to_chart, got: myapp
   Solution: Ensure correct chart reference format "repository_name/chart_name"
   ```

2. **Kubernetes client initialization failure**

   ```txt
   Error: kubernetes client not initialized in Helm configuration
   Solution: Check KUBECONFIG settings and cluster connection
   ```

3. **Insufficient permissions**

   ```txt
   Error: forbidden: User cannot create resource
   Solution: Check RBAC configuration and service account permissions
   ```

### Debugging Tips

1. **Enable verbose logging**

   ```bash
   # Set log level
   export LOG_LEVEL=debug
   go run cmd/main.go
   ```

2. **Check resource status**

   ```bash
   # Check HelmRelease status
   kubectl describe helmrelease myapp-sample
   
   # View events
   kubectl get events --sort-by=.metadata.creationTimestamp
   ```

3. **Verify using Helm CLI**

   ```bash
   # List releases
   helm list -A
   
   # View release details
   helm get all <release-name> -n <namespace>
   ```

## Contributing Guide

### Code Standards

1. **Go code style**
   - Follow `gofmt` formatting
   - Use `golint` for code quality checks
   - Add appropriate comments and documentation

2. **Commit message format**

   ```txt
   <type>(<scope>): <subject>
   
   <body>
   
   <footer>
   ```

   Example:

   ```txt
   feat(controller): add support for OCI repositories
   
   - Add OCI repository type support
   - Update chart reference handling
   - Add integration tests
   
   Fixes #123
   ```

3. **Testing requirements**
   - New features must include unit tests
   - Test coverage should not be below 80%
   - Integration tests to verify end-to-end functionality

### Release Process

1. **Version tagging**

   ```bash
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin v1.0.0
   ```

2. **Build release**

   ```bash
   make release VERSION=v1.0.0
   ```

## References

- [Kubernetes Controller Runtime](https://github.com/kubernetes-sigs/controller-runtime)
- [Helm Go SDK](https://helm.sh/docs/topics/advanced/#go-sdk)
- [Kubebuilder Book](https://book.kubebuilder.io/)
- [Operator Pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)

## License

This project is licensed under the Apache 2.0 License. See the [LICENSE](LICENSE) file for details.
