# Helm Operator

[中文文档](README_zh.md) | English

> **Version**: v0.3.0

A production-grade Kubernetes Operator for managing Helm repositories and releases through Custom Resource Definitions (CRDs), with intelligent automation and advanced features.

## Overview

Helm Operator provides a declarative way to manage Helm repositories and releases in Kubernetes clusters. It extends Kubernetes with custom resources that allow you to:

- **🏪 Manage Helm Repositories**: Automatic sync with intelligent error handling and retry
- **🚀 Manage Helm Releases**: Declarative install, upgrade with automatic rollback
- **📦 OCI Registry Support**: Full support for OCI-based Helm charts (Recommended)
- **🔄 Automatic Rollback**: Auto-recover from failed upgrades
- **📊 Semantic Versioning**: SemVer constraints for flexible version management
- **🔐 Authentication Support**: Private repositories with Basic Auth and TLS
- **📈 Full Observability**: Prometheus metrics and comprehensive logging

## Features

### 🏪 HelmRepository Management

- **OCI Registry Support** (Recommended for production)
- Automatic repository synchronization
- Chart discovery and version tracking
- Authentication support (Basic Auth, TLS, OCI auth)
- Status reporting with chart information
- Configurable sync intervals with smart retry
- ConfigMap policy (disabled/on-demand/lazy)

### 🚀 HelmRelease Management

- Declarative release management
- YAML-based values configuration
- **Automatic rollback on upgrade failure** 🆕
- **SemVer version constraints** 🆕
- Dependency management between releases
- Rollback and history tracking
- Health check integration

### 🔐 Security & Authentication

- Private repository support
- OCI registry authentication
- TLS certificate management
- Kubernetes Secret integration
- RBAC permissions

### 📊 Observability

- **Prometheus metrics** (15+ metrics) 🆕
- Real-time status conditions
- Event recording
- Comprehensive logging with structured output
- Grafana dashboard ready

## Architecture

```txt
┌────────────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster                          │
│                                                                │
│         ┌─────────────────┐    ┌─────────────────┐             │
│         │  HelmRepository │    │   HelmRelease   │             │
│         │       CRD       │    │      CRD        │             │
│         └─────────────────┘    └─────────────────┘             │
│                  │                      │                      │
│                  V                      V                      │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │              Helm Operator                              │   │
│  │                                                         │   │
│  │      ┌─────────────────┐    ┌─────────────────┐         │   │
│  │      │  Repository     │    │   Release       │         │   │
│  │      │  Controller     │    │  Controller     │         │   │
│  │      └─────────────────┘    └─────────────────┘         │   │
│  │               │                      │                  │   │
│  │               V                      V                  │   │
│  │  ┌──────────────────────────────────────────────────┐   │   │
│  │  │               Helm Client Library                │   │   │
│  │  └──────────────────────────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              |                                 │
└──────────────────────────────┼─────────────────────────────────┘
                               V
                      ┌─────────────────┐
                      │  External Helm  │
                      │  Repositories   │
                      └─────────────────┘
```

## Quick Start

### Prerequisites

- Kubernetes cluster v1.25+
- kubectl configured to access your cluster
- Go 1.21+ (for development)
- Docker (for building images)

### Installation

#### Install with Helm (Recommended)

```bash
# Add Helm repository
helm repo add helm-operator https://ketches.github.io/helm-operator
helm repo update

# Install the operator
helm install helm-operator helm-operator/helm-operator \
  -n ketches --create-namespace

# Verify installation
kubectl get pods -n ketches
```

#### Install with manifests

```bash
# Install CRDs
kubectl apply -f https://raw.githubusercontent.com/ketches/helm-operator/master/deploy/crds/

# Deploy the Operator
kubectl create namespace ketches
kubectl apply -f https://raw.githubusercontent.com/ketches/helm-operator/master/deploy/manifests.yaml
```

### Basic Usage

#### Method 1: Using OCI Registry (Recommended)

OCI registries offer better performance, security, and are the future of Helm chart distribution.

Step 1: **Create an OCI-based HelmRepository**

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
  # Optimize ConfigMap usage
  valuesConfigMapPolicy: disabled  # Recommended
```

Step 2: **Deploy a Release from OCI**

```yaml
apiVersion: helm-operator.ketches.cn/v1alpha1
kind: HelmRelease
metadata:
  name: my-app
  namespace: default
spec:
  chart:
    name: myapp
    version: "^1.0.0"  # SemVer constraint
    ociRepository: "oci://ghcr.io/myorg/charts/myapp"
  
  release:
    name: my-app
    namespace: production
    createNamespace: true
  
  values: |
    replicaCount: 3
    image:
      tag: "v1.0.0"
  
  # Enable automatic rollback for safety
  rollback:
    enabled: true
    timeout: "5m"
  
  install:
    timeout: "10m"
    wait: true
  
  upgrade:
    timeout: "10m"
    wait: true
```

**Apply the resources:**

```bash
kubectl apply -f helmrepository-oci.yaml
kubectl apply -f helmrelease-oci.yaml

# Check status
kubectl get helmrepository ghcr-charts
kubectl get helmrelease my-app
```

#### Method 2: Using Traditional HTTP Repository

Step 1: **Create a traditional HelmRepository**

```yaml
apiVersion: helm-operator.ketches.cn/v1alpha1
kind: HelmRepository
metadata:
  name: bitnami
  namespace: default
spec:
  url: "https://charts.bitnami.com/bitnami"
  type: "helm"
  interval: "1h"
  valuesConfigMapPolicy: disabled  # Recommended
```

Step 2: **Deploy a Release**

```yaml
apiVersion: helm-operator.ketches.cn/v1alpha1
kind: HelmRelease
metadata:
  name: nginx
  namespace: default
spec:
  chart:
    name: nginx
    version: "~15.0.0"  # Track minor versions
    repository:
      name: bitnami
      namespace: default
  
  release:
    name: nginx
    namespace: default
  
  values: |
    service:
      type: LoadBalancer
  
  rollback:
    enabled: true
  
  upgrade:
    wait: true
```

#### Check Status and Events

```bash
# Check repository sync status
kubectl describe helmrepository ghcr-charts

# Check release status
kubectl get helmrelease my-app -o yaml

# View events
kubectl get events --field-selector involvedObject.name=my-app

# Check Prometheus metrics (if configured)
kubectl port-forward -n ketches svc/helm-operator-metrics 8080:8080
curl http://localhost:8080/metrics | grep helm_
```

## Development

### Local Development Setup

1. **Clone the repository:**

```bash
git clone https://github.com/ketches/helm-operator.git
cd helm-operator
```

2. **Install dependencies:**

```bash
make generate
make manifests
```

3. **Run locally:**

```bash
make install  # Install CRDs
make run      # Run controller locally
```

4. **Build and test:**

```bash
make build    # Build binary
make test     # Run tests
```

### Building Docker Image Locally

```bash
make docker-build-local IMG=helm-operator VERSION=dev
```

### Deploying to Cluster

```bash
make deploy
```

## Configuration Examples

### OCI Registry with Authentication (Recommended)

#### Public OCI Registry (GitHub Container Registry)

```yaml
apiVersion: helm-operator.ketches.cn/v1alpha1
kind: HelmRepository
metadata:
  name: ghcr-public
spec:
  url: "oci://ghcr.io/myorg/charts"
  type: "oci"
  interval: "1h"
  valuesConfigMapPolicy: disabled
```

#### Private OCI Registry with Authentication

```yaml
# Create authentication secret
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
spec:
  url: "oci://myregistry.azurecr.io/helm"
  type: "oci"
  auth:
    secretRef:
      name: oci-registry-auth
  interval: "1h"
  valuesConfigMapPolicy: disabled
```

### Production-Ready Release with Auto-Rollback

```yaml
apiVersion: helm-operator.ketches.cn/v1alpha1
kind: HelmRelease
metadata:
  name: production-app
  namespace: production
spec:
  chart:
    name: myapp
    version: "^2.0.0"  # Auto-update to compatible versions
    ociRepository: "oci://ghcr.io/company/charts/myapp"
  
  release:
    name: production-app
    namespace: production
    createNamespace: true
  
  values: |
    replicaCount: 5
    
    image:
      repository: company.azurecr.io/myapp
      tag: "2.1.5"
      pullPolicy: IfNotPresent
    
    resources:
      limits:
        cpu: 2000m
        memory: 2Gi
      requests:
        cpu: 500m
        memory: 512Mi
    
    autoscaling:
      enabled: true
      minReplicas: 5
      maxReplicas: 20
      targetCPUUtilizationPercentage: 70
    
    ingress:
      enabled: true
      className: nginx
      hosts:
        - host: app.example.com
          paths:
            - path: /
              pathType: Prefix
      tls:
        - secretName: app-tls
          hosts:
            - app.example.com
  
  # Critical: Enable automatic rollback
  rollback:
    enabled: true
    toRevision: 0       # Rollback to previous version
    timeout: "5m"
    wait: true
    cleanupOnFail: true
  
  install:
    timeout: "15m"
    wait: true
    waitForJobs: true
  
  upgrade:
    timeout: "15m"
    wait: true          # Required for auto-rollback detection
    waitForJobs: true
    cleanupOnFail: true
  
  # Check for updates every 12 hours
  interval: "12h"
```

### Traditional HTTP Repository Configuration

```yaml
apiVersion: helm-operator.ketches.cn/v1alpha1
kind: HelmRepository
metadata:
  name: bitnami
spec:
  url: "https://charts.bitnami.com/bitnami"
  type: "helm"
  interval: "1h"
  timeout: "10m"
  
  # ConfigMap optimization (recommended)
  valuesConfigMapPolicy: disabled
  # Or use lazy policy for latest versions only
  # valuesConfigMapPolicy: lazy
  # valuesConfigMapRetention: "168h"  # 7 days
```

### Private Repository with Basic Auth

```yaml
# Create auth secret
apiVersion: v1
kind: Secret
metadata:
  name: private-repo-auth
type: Opaque
data:
  username: dXNlcm5hbWU=  # base64: username
  password: cGFzc3dvcmQ=  # base64: password
---
apiVersion: helm-operator.ketches.cn/v1alpha1
kind: HelmRepository
metadata:
  name: private-repo
spec:
  url: "https://charts.company.com"
  auth:
    basic:
      secretRef:
        name: private-repo-auth
  timeout: "10m"
  valuesConfigMapPolicy: disabled
```

### Version Constraint Examples

```yaml
apiVersion: helm-operator.ketches.cn/v1alpha1
kind: HelmRelease
metadata:
  name: app-with-constraints
spec:
  chart:
    name: myapp
    # Semantic version constraints
    version: "^1.2.0"     # >= 1.2.0, < 2.0.0 (recommended for production)
    # version: "~1.2.0"   # >= 1.2.0, < 1.3.0 (conservative)
    # version: ">=1.0.0, <2.0.0"  # Range
    # version: "1.2.3"    # Exact version (most stable)
    # version: "latest"   # Always latest (dev only)
    ociRepository: "oci://ghcr.io/charts/myapp"
  
  release:
    name: my-app
    namespace: default
  
  # Rollback protection
  rollback:
    enabled: true
```

## Advanced Features

### Prometheus Metrics

The operator exports comprehensive metrics for monitoring:

```yaml
# ServiceMonitor for Prometheus Operator
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: helm-operator
  namespace: ketches
spec:
  selector:
    matchLabels:
      app: helm-operator
  endpoints:
  - port: metrics
    interval: 30s
```

**Available Metrics:**

- `helm_repository_sync_duration_seconds` - Repository sync latency
- `helm_repository_sync_total` - Repository sync count by status
- `helm_release_operation_duration_seconds` - Release operation latency
- `helm_release_rollbacks_total` - Automatic rollback count
- `helm_operator_reconcile_duration_seconds` - Controller performance

### Automatic Rollback Scenarios

```yaml
# Scenario 1: Rollback on timeout
spec:
  rollback:
    enabled: true
  upgrade:
    timeout: "5m"
    wait: true  # Pod doesn't become ready in 5m → auto rollback

# Scenario 2: Rollback on health check failure
spec:
  rollback:
    enabled: true
  upgrade:
    wait: true
    waitForJobs: true  # Job fails → auto rollback

# Scenario 3: Rollback to specific revision
spec:
  rollback:
    enabled: true
    toRevision: 3  # Rollback to revision 3 if upgrade fails
```

### 📦 Multi-Cloud OCI Examples

#### GitHub Container Registry (GHCR)

```yaml
apiVersion: helm-operator.ketches.cn/v1alpha1
kind: HelmRepository
metadata:
  name: ghcr-charts
spec:
  url: "oci://ghcr.io/myorg/charts"
  type: "oci"
  valuesConfigMapPolicy: disabled
---
apiVersion: helm-operator.ketches.cn/v1alpha1
kind: HelmRelease
metadata:
  name: my-app
spec:
  chart:
    name: myapp
    version: "^1.0.0"
    ociRepository: "oci://ghcr.io/myorg/charts/myapp"
  rollback:
    enabled: true
```

#### Azure Container Registry (ACR)

```yaml
# Create ACR auth secret
kubectl create secret docker-registry acr-auth \
  --docker-server=myregistry.azurecr.io \
  --docker-username=<username> \
  --docker-password=<password>

---
apiVersion: helm-operator.ketches.cn/v1alpha1
kind: HelmRepository
metadata:
  name: acr-charts
spec:
  url: "oci://myregistry.azurecr.io/helm"
  type: "oci"
  auth:
    secretRef:
      name: acr-auth
---
apiVersion: helm-operator.ketches.cn/v1alpha1
kind: HelmRelease
metadata:
  name: acr-app
spec:
  chart:
    name: myapp
    version: "~2.0.0"
    ociRepository: "oci://myregistry.azurecr.io/helm/myapp"
  rollback:
    enabled: true
```

#### Google Artifact Registry (GAR)

```yaml
apiVersion: helm-operator.ketches.cn/v1alpha1
kind: HelmRepository
metadata:
  name: gar-charts
spec:
  url: "oci://us-docker.pkg.dev/project-id/helm-charts"
  type: "oci"
  auth:
    secretRef:
      name: gar-auth  # GCP service account key
---
apiVersion: helm-operator.ketches.cn/v1alpha1
kind: HelmRelease
metadata:
  name: gar-app
spec:
  chart:
    name: myapp
    ociRepository: "oci://us-docker.pkg.dev/project-id/helm-charts/myapp"
  rollback:
    enabled: true
```

## Monitoring & Observability

### Grafana Dashboard

Example queries for monitoring:

```promql
# Repository sync success rate
sum(rate(helm_repository_sync_total{status="success"}[5m])) 
  / 
sum(rate(helm_repository_sync_total[5m]))

# Release operation P95 latency
histogram_quantile(0.95, 
  sum(rate(helm_release_operation_duration_seconds_bucket[5m])) 
  by (le, operation))

# Automatic rollback frequency
sum(increase(helm_release_rollbacks_total[1h])) by (release, status)

# Active releases
count(helm_release_info{status="deployed"})
```

### Alert Rules

```yaml
groups:
- name: helm-operator
  rules:
  - alert: HelmRepositorySyncFailed
    expr: rate(helm_repository_sync_errors_total[5m]) > 0
    for: 5m
    annotations:
      summary: "Repository {{ $labels.repository }} sync failing"
  
  - alert: HelmReleaseOperationFailed
    expr: rate(helm_release_operation_errors_total[5m]) > 0
    for: 2m
    annotations:
      summary: "Release {{ $labels.release }} operation failing"
  
  - alert: FrequentRollbacks
    expr: sum(increase(helm_release_rollbacks_total[1h])) by (release) > 3
    annotations:
      summary: "Release {{ $labels.release }} has frequent rollbacks"
```

## API Reference

For detailed API documentation, see:

- [HelmRepository API](docs/api-reference.md#helmrepository)
- [HelmRelease API](docs/api-reference.md#helmrelease)

## Contributing

We welcome contributions! Please see our [Contributing Guide](./CONTRIBUTING.md) and [Developer Guide](./DEVELOPER_GUIDE.md) for details.

### Development Workflow

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Run `make test lint`
6. Submit a pull request

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Support

- 📖 [Documentation](docs/)
- 🐛 [Issue Tracker](https://github.com/ketches/helm-operator/issues)
- 💬 [Discussions](https://github.com/ketches/helm-operator/discussions)
