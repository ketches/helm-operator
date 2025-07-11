# Helm Operator

[中文文档](README_zh.md) | English

A Kubernetes Operator for managing Helm repositories and releases through Custom Resource Definitions (CRDs).

## Overview

Helm Operator provides a declarative way to manage Helm repositories and releases in Kubernetes clusters. It extends Kubernetes with custom resources that allow you to:

- **Manage Helm Repositories**: Automatically sync Helm repositories and track available charts
- **Manage Helm Releases**: Declaratively install, upgrade, and manage Helm releases
- **Authentication Support**: Support for private repositories with Basic Auth and TLS
- **Status Tracking**: Real-time status updates and chart information
- **Event Recording**: Comprehensive event logging for operations

## Features

### 🏪 HelmRepository Management

- Automatic repository synchronization
- Chart discovery and version tracking
- Authentication support (Basic Auth, TLS)
- Status reporting with chart information
- Configurable sync intervals

### 🚀 HelmRelease Management

- Declarative release management
- YAML-based values configuration
- Automatic upgrades on configuration changes
- Dependency management between releases
- Rollback and history tracking

### 🔐 Security & Authentication

- Private repository support
- TLS certificate management
- Kubernetes Secret integration
- RBAC permissions

### 📊 Observability

- Real-time status conditions
- Event recording
- Metrics and monitoring ready
- Comprehensive logging

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

#### Install with manifests

1. **Install CRDs:**

```bash
kubectl apply -f https://raw.githubusercontent.com/ketches/helm-operator/master/deploy/crds/helm-operator.ketches.cn_helmrepositories.yaml
kubectl apply -f https://raw.githubusercontent.com/ketches/helm-operator/master/deploy/crds/helm-operator.ketches.cn_helmreleases.yaml
```

2. **Deploy the Operator:**

```bash
kubectl create namespace ketches
kubectl apply -f https://raw.githubusercontent.com/ketches/helm-operator/master/deploy/manifests.yaml
```

#### Install with Helm

1. **Add Helm repository:**

```bash
helm repo add helm-operator https://ketches.github.io/helm-operator
helm repo update
```

2. **Install the operator:**

```bash
helm install helm-operator helm-operator/helm-operator -n ketches --create-namespace
```

3. **Verify Installation:**

```bash
kubectl get pods -n ketches
```

### Basic Usage

#### 1. Create a Helm Repository

```bash
kubectl apply -f https://raw.githubusercontent.com/ketches/helm-operator/master/samples/helm_repository.yaml
```

#### 2. Create a Helm Release

```bash
kubectl apply -f https://raw.githubusercontent.com/ketches/helm-operator/master/samples/helm_release.yaml
```

#### 3. Check Status

```bash
# Check repository status
kubectl get helmrepository helm-operator-charts

# Check release status
kubectl get helmrelease nginx
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

## Configuration

### HelmRepository Configuration

```yaml
apiVersion: helm-operator.ketches.cn/v1alpha1
kind: HelmRepository
metadata:
  name: private-repo
spec:
  url: "https://private.charts.example.com"
  interval: "1h"
  auth:
    basic:
      secretRef:
        name: repo-credentials
        namespace: default
  timeout: "10m"
```

### HelmRelease Configuration

```yaml
apiVersion: helm-operator.ketches.cn/v1alpha1
kind: HelmRelease
metadata:
  name: my-app
spec:
  chart:
    name: my-app
    version: "1.0.0"
    repository:
      name: my-repo
      namespace: default
  release:
    name: my-app-release
    namespace: production
    createNamespace: true
  values: |
    image:
      tag: "v1.0.0"
    resources:
      requests:
        cpu: "100m"
        memory: "128Mi"
  install:
    timeout: "10m"
    wait: true
  upgrade:
    timeout: "10m"
    wait: true
```

## Examples

### Private Repository with Authentication

```yaml
# Create authentication secret
apiVersion: v1
kind: Secret
metadata:
  name: private-repo-auth
type: Opaque
data:
  username: dXNlcm5hbWU=  # base64 encoded
  password: cGFzc3dvcmQ=  # base64 encoded
---
# Private repository with authentication
apiVersion: helm-operator.ketches.cn/v1alpha1
kind: HelmRepository
metadata:
  name: private-repo
spec:
  url: "https://private.charts.example.com"
  auth:
    basic:
      secretRef:
        name: private-repo-auth
```

### Complex Release Configuration

```yaml
apiVersion: helm-operator.ketches.cn/v1alpha1
kind: HelmRelease
metadata:
  name: complex-app
spec:
  chart:
    name: my-app
    version: "2.0.0"
    repository:
      name: my-repo
  release:
    name: complex-app
    namespace: production
    createNamespace: true
  values: |
    # Application configuration
    app:
      name: "complex-app"
      version: "2.0.0"
    
    # Replica count
    replicaCount: 3
    
    # Image configuration
    image:
      repository: "my-registry/my-app"
      tag: "v2.0.0"
      pullPolicy: "IfNotPresent"
    
    # Service configuration
    service:
      type: "ClusterIP"
      port: 8080
      targetPort: 8080
    
    # Ingress configuration
    ingress:
      enabled: true
      className: "nginx"
      hosts:
        - host: "app.example.com"
          paths:
            - path: "/"
              pathType: "Prefix"
      tls:
        - secretName: "app-tls"
          hosts:
            - "app.example.com"
    
    # Resource limits
    resources:
      limits:
        cpu: "1000m"
        memory: "1Gi"
      requests:
        cpu: "500m"
        memory: "512Mi"
    
    # Environment variables
    env:
      - name: "APP_ENV"
        value: "production"
      - name: "DB_HOST"
        value: "postgres.database.svc.cluster.local"
  
  # Install configuration
  install:
    timeout: "15m"
    wait: true
    waitForJobs: true
  
  # Upgrade configuration
  upgrade:
    timeout: "15m"
    wait: true
    cleanupOnFail: true
  
  # Dependencies
  dependsOn:
    - name: "postgres"
      namespace: "database"
```

## API Reference

For detailed API documentation, see:

- [HelmRepository API](.dev/api-reference.md#helmrepository)
- [HelmRelease API](.dev/api-reference.md#helmrelease)

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

## Roadmap

- [x] HelmRepository management
- [x] HelmRelease management
- [ ] OCI repository support
- [ ] Webhook validation

---

**Note**: This project is under active development. APIs may change before v1.0.0 release.
