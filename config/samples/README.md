# Helm Operator Samples

This directory contains various examples demonstrating how to use the Helm Operator.

## Quick Start

To deploy the basic samples:

```bash
kubectl apply -k config/samples/
```

## Available Examples

### Basic Examples

#### 1. Basic HelmRepository

- **File**: `helm-operator_v1alpha1_helmrepository.yaml`
- **Description**: Simple public repository configuration
- **Repository**: Bitnami charts repository

#### 2. Basic HelmRelease

- **File**: `helm-operator_v1alpha1_helmrelease.yaml`
- **Description**: Simple nginx deployment
- **Dependencies**: Requires the basic HelmRepository

### Authentication Examples

#### 3. Private Repository with Basic Auth

- **File**: `private-repository-example.yaml`
- **Description**: Repository with username/password authentication
- **Includes**: Secret for credentials

#### 4. Repository with TLS Authentication

- **File**: `tls-repository-example.yaml`
- **Description**: Repository with client certificate authentication
- **Includes**: Secret for TLS certificates

### Advanced Deployment Examples

#### 5. Complex Application

- **File**: `complex-release-example.yaml`
- **Description**: Production-ready application with full configuration
- **Features**:
  - Autoscaling
  - Ingress with TLS
  - Resource limits
  - Health checks
  - Security contexts
  - Dependencies

#### 6. Multi-Environment Deployment

- **File**: `multi-environment-example.yaml`
- **Description**: Same application deployed to dev, staging, and production
- **Features**:
  - Environment-specific configurations
  - Different resource allocations
  - Progressive deployment strategy

#### 7. Canary Deployment

- **File**: `canary-deployment-example.yaml`
- **Description**: Blue-green deployment with traffic splitting
- **Features**:
  - Stable and canary versions
  - Nginx ingress traffic splitting
  - Dependency management

### Microservices Examples

#### 8. Microservices Stack

- **File**: `microservices-example.yaml`
- **Description**: Complete microservices deployment
- **Components**:
  - PostgreSQL database
  - Redis cache
  - User service
  - Order service
  - API Gateway
- **Features**: Service dependencies and inter-service communication

### Monitoring Examples

#### 9. Monitoring Stack

- **File**: `monitoring-stack-example.yaml`
- **Description**: Complete observability stack
- **Components**:
  - Prometheus + Grafana
  - Loki for logs
  - Jaeger for tracing
- **Features**: Persistent storage and ingress configuration

## Usage Instructions

### Deploy Basic Examples

1. **Deploy basic repository and release**:

   ```bash
   kubectl apply -f config/samples/helm-operator_v1alpha1_helmrepository.yaml
   kubectl apply -f config/samples/helm-operator_v1alpha1_helmrelease.yaml
   ```

2. **Check status**:

   ```bash
   kubectl get helmrepository
   kubectl get helmrelease
   ```

### Deploy Advanced Examples

1. **Multi-environment deployment**:

   ```bash
   # Create namespaces first
   kubectl create namespace development
   kubectl create namespace staging
   kubectl create namespace production
   
   # Deploy the releases
   kubectl apply -f config/samples/multi-environment-example.yaml
   ```

2. **Microservices stack**:

   ```bash
   # Create namespaces
   kubectl create namespace database
   kubectl create namespace cache
   kubectl create namespace microservices
   kubectl create namespace gateway
   
   # Deploy the stack
   kubectl apply -f config/samples/microservices-example.yaml
   ```

3. **Monitoring stack**:

   ```bash
   # Create monitoring namespace
   kubectl create namespace monitoring
   
   # Add required repositories first
   kubectl apply -f - <<EOF
   apiVersion: helm-operator.ketches.cn/v1alpha1
   kind: HelmRepository
   metadata:
     name: prometheus-community
     namespace: helm-system
   spec:
     url: "https://prometheus-community.github.io/helm-charts"
   ---
   apiVersion: helm-operator.ketches.cn/v1alpha1
   kind: HelmRepository
   metadata:
     name: grafana
     namespace: helm-system
   spec:
     url: "https://grafana.github.io/helm-charts"
   ---
   apiVersion: helm-operator.ketches.cn/v1alpha1
   kind: HelmRepository
   metadata:
     name: jaegertracing
     namespace: helm-system
   spec:
     url: "https://jaegertracing.github.io/helm-charts"
   EOF
   
   # Deploy monitoring stack
   kubectl apply -f config/samples/monitoring-stack-example.yaml
   ```

### Deploy with Authentication

1. **Private repository**:

   ```bash
   # Update the secret with real credentials
   kubectl apply -f config/samples/private-repository-example.yaml
   ```

2. **TLS repository**:

   ```bash
   # Update the secret with real certificates
   kubectl apply -f config/samples/tls-repository-example.yaml
   ```

## Customization

### Modify Examples

1. **Copy an example**:

   ```bash
   cp config/samples/helm-operator_v1alpha1_helmrelease.yaml my-release.yaml
   ```

2. **Edit the configuration**:
   - Change metadata (name, namespace)
   - Update chart name and version
   - Modify values as needed
   - Adjust resource requirements

3. **Apply your customized version**:

   ```bash
   kubectl apply -f my-release.yaml
   ```

### Using Kustomize

1. **Create your own kustomization**:

   ```yaml
   # my-kustomization.yaml
   apiVersion: kustomize.config.k8s.io/v1beta1
   kind: Kustomization
   
   resources:
   - config/samples/helm-operator_v1alpha1_helmrepository.yaml
   - config/samples/complex-release-example.yaml
   
   namespace: my-namespace
   
   patchesStrategicMerge:
   - my-patches.yaml
   ```

2. **Apply with kustomize**:

   ```bash
   kubectl apply -k .
   ```

## Monitoring and Troubleshooting

### Check Resource Status

```bash
# Check repositories
kubectl get helmrepository -A
kubectl describe helmrepository <name>

# Check releases
kubectl get helmrelease -A
kubectl describe helmrelease <name>

# Check events
kubectl get events --field-selector involvedObject.kind=HelmRepository
kubectl get events --field-selector involvedObject.kind=HelmRelease
```

### View Logs

```bash
# Controller logs
kubectl logs -n helm-operator-system deployment/helm-operator-controller-manager

# Follow logs
kubectl logs -n helm-operator-system deployment/helm-operator-controller-manager -f
```

### Common Issues

1. **Repository not ready**: Check URL and authentication
2. **Release failed**: Check chart name, version, and values
3. **Dependencies not met**: Ensure required repositories are ready
4. **Resource conflicts**: Check for naming conflicts

## Best Practices

1. **Use specific chart versions** in production
2. **Set resource limits** for all deployments
3. **Use secrets** for sensitive configuration
4. **Implement health checks** for applications
5. **Set up monitoring** for production deployments
6. **Use dependencies** to ensure proper deployment order
7. **Test in development** before deploying to production

## Contributing

To add new examples:

1. Create a new YAML file with descriptive name
2. Add comprehensive comments
3. Include realistic configurations
4. Update this README
5. Test the example thoroughly
