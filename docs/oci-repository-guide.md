# OCI Helm Repository Guide

[中文](oci-repository-guide.md) | English

This document provides complete examples for using OCI (Open Container Initiative) Helm repositories with Helm Operator.

## Overview

As of v0.3.0, Helm Operator supports OCI-format Helm repositories. OCI repositories use the Docker Registry protocol and offer better bandwidth usage and version management.

## OCI vs Traditional Helm Repositories

| Feature | Traditional Helm | OCI Registry |
| --- | --- | --- |
| **Protocol** | HTTP/HTTPS | OCI (Docker Registry) |
| **URL format** | `https://charts.example.com` | `oci://registry.example.com/charts` |
| **Index file** | `index.yaml` | Not required |
| **Chart listing** | Full list available | Listing not supported |
| **Authentication** | Basic Auth, TLS | Docker Registry Auth |
| **Bandwidth** | Full index download | Pull on demand |

## Example 1: Public GitHub Container Registry (GHCR)

### 1.1 Register OCI Repository

```yaml
apiVersion: helm-operator.ketches.cn/v1alpha1
kind: HelmRepository
metadata:
  name: ghcr-public
  namespace: default
spec:
  url: "oci://ghcr.io/myorg/charts"
  type: "oci"
  interval: "1h"
  timeout: "10m"
  suspend: false
```

### 1.2 Use OCI Chart

```yaml
apiVersion: helm-operator.ketches.cn/v1alpha1
kind: HelmRelease
metadata:
  name: nginx-oci
  namespace: default
spec:
  chart:
    name: nginx
    version: "1.0.0"
    ociRepository: "oci://ghcr.io/myorg/charts/nginx"
  release:
    name: nginx-oci
    namespace: default
    createNamespace: true
  values: |
    replicaCount: 2
    service:
      type: ClusterIP
      port: 80
  install:
    timeout: "10m"
    wait: true
  upgrade:
    timeout: "10m"
    wait: true
```

## Example 2: Private Azure Container Registry (ACR)

### 2.1 Create Authentication Secret

```bash
# Get ACR credentials
az acr login --name myregistry
ACR_USERNAME=$(az acr credential show --name myregistry --query username -o tsv)
ACR_PASSWORD=$(az acr credential show --name myregistry --query passwords[0].value -o tsv)

# Create Kubernetes Secret
kubectl create secret generic acr-auth \
  --from-literal=username=$ACR_USERNAME \
  --from-literal=password=$ACR_PASSWORD \
  --namespace default
```

Or with YAML:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: acr-auth
  namespace: default
type: Opaque
data:
  username: <base64-encoded-username>
  password: <base64-encoded-password>
```

### 2.2 Register Private OCI Repository

```yaml
apiVersion: helm-operator.ketches.cn/v1alpha1
kind: HelmRepository
metadata:
  name: acr-private
  namespace: default
spec:
  url: "oci://myregistry.azurecr.io/helm"
  type: "oci"
  interval: "2h"
  timeout: "15m"
  auth:
    basic:
      secretRef:
        name: acr-auth
        namespace: default
  suspend: false
```

### 2.3 Deploy Chart

```yaml
apiVersion: helm-operator.ketches.cn/v1alpha1
kind: HelmRelease
metadata:
  name: myapp-acr
  namespace: production
spec:
  chart:
    name: myapp
    version: "2.1.0"
    ociRepository: "oci://myregistry.azurecr.io/helm/myapp"
  release:
    name: myapp
    namespace: production
    createNamespace: true
  values: |
    image:
      repository: myregistry.azurecr.io/apps/myapp
      tag: "2.1.0"
      pullPolicy: IfNotPresent
    
    replicaCount: 3
    
    resources:
      limits:
        cpu: 500m
        memory: 512Mi
      requests:
        cpu: 100m
        memory: 128Mi
    
    ingress:
      enabled: true
      className: nginx
      hosts:
        - host: myapp.example.com
          paths:
            - path: /
              pathType: Prefix
  install:
    timeout: "15m"
    wait: true
    waitForJobs: true
  upgrade:
    timeout: "15m"
    wait: true
    cleanupOnFail: true
```

## Example 3: Amazon Elastic Container Registry (ECR)

### 3.1 Configure ECR Authentication

```bash
# Get ECR login token
aws ecr get-login-password --region us-east-1 | \
  docker login --username AWS --password-stdin 123456789012.dkr.ecr.us-east-1.amazonaws.com

# Create Secret (note: ECR token expires; consider IAM roles for production)
kubectl create secret generic ecr-auth \
  --from-literal=username=AWS \
  --from-literal=password=$(aws ecr get-login-password --region us-east-1) \
  --namespace default
```

**Recommended**: Use [external-secrets-operator](https://external-secrets.io/) to refresh ECR tokens automatically:

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: ecr-auth
  namespace: default
spec:
  refreshInterval: 1h  # ECR token valid for 12 hours
  secretStoreRef:
    name: aws-secrets-manager
    kind: SecretStore
  target:
    name: ecr-auth
  data:
  - secretKey: username
    remoteRef:
      key: helm/ecr-credentials
      property: username
  - secretKey: password
    remoteRef:
      key: helm/ecr-credentials
      property: password
```

### 3.2 Register ECR OCI Repository

```yaml
apiVersion: helm-operator.ketches.cn/v1alpha1
kind: HelmRepository
metadata:
  name: ecr-helm
  namespace: default
spec:
  url: "oci://123456789012.dkr.ecr.us-east-1.amazonaws.com/helm"
  type: "oci"
  interval: "1h"
  timeout: "10m"
  auth:
    basic:
      secretRef:
        name: ecr-auth
        namespace: default
```

## Example 4: Google Artifact Registry (GAR)

### 4.1 Configure GAR Authentication

```bash
# Use gcloud to generate access token
gcloud auth print-access-token | \
  docker login -u oauth2accesstoken --password-stdin https://us-central1-docker.pkg.dev

# Create Service Account key (recommended)
gcloud iam service-accounts create helm-operator \
  --display-name="Helm Operator Service Account"

gcloud iam service-accounts keys create key.json \
  --iam-account=helm-operator@PROJECT_ID.iam.gserviceaccount.com

# Grant Artifact Registry Reader role
gcloud projects add-iam-policy-binding PROJECT_ID \
  --member="serviceAccount:helm-operator@PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/artifactregistry.reader"

# Create Secret
kubectl create secret generic gar-auth \
  --from-literal=username=_json_key \
  --from-file=password=key.json \
  --namespace default
```

### 4.2 Register GAR Repository

```yaml
apiVersion: helm-operator.ketches.cn/v1alpha1
kind: HelmRepository
metadata:
  name: gar-helm
  namespace: default
spec:
  url: "oci://us-central1-docker.pkg.dev/PROJECT_ID/helm-charts"
  type: "oci"
  interval: "1h"
  timeout: "10m"
  auth:
    basic:
      secretRef:
        name: gar-auth
        namespace: default
```

## Example 5: Harbor OCI Registry

### 5.1 Create Harbor Project and User

1. Log in to Harbor Web UI
2. Create project `helm-charts`
3. Create a Robot Account or use user credentials

### 5.2 Configure Harbor Authentication

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: harbor-auth
  namespace: default
type: Opaque
stringData:
  username: "robot$helm-operator"
  password: "your-robot-token"
```

### 5.3 Register Harbor OCI Repository

```yaml
apiVersion: helm-operator.ketches.cn/v1alpha1
kind: HelmRepository
metadata:
  name: harbor-helm
  namespace: default
spec:
  url: "oci://harbor.example.com/helm-charts"
  type: "oci"
  interval: "30m"
  timeout: "10m"
  auth:
    basic:
      secretRef:
        name: harbor-auth
        namespace: default
    tls:
      insecureSkipVerify: false  # Use valid certs in production
```

### 5.4 Use Harbor Chart

```yaml
apiVersion: helm-operator.ketches.cn/v1alpha1
kind: HelmRelease
metadata:
  name: webapp-harbor
  namespace: apps
spec:
  chart:
    name: webapp
    version: "3.2.1"
    ociRepository: "oci://harbor.example.com/helm-charts/webapp"
  release:
    name: webapp
    namespace: apps
    createNamespace: true
  values: |
    image:
      repository: harbor.example.com/library/webapp
      tag: "3.2.1"
    
    ingress:
      enabled: true
      hosts:
        - host: webapp.example.com
  install:
    timeout: "10m"
    wait: true
```

## Pushing Charts to an OCI Registry

### Using Helm CLI

```bash
# 1. Package chart
helm package ./my-chart

# 2. Log in to OCI registry
helm registry login oci://registry.example.com

# 3. Push chart
helm push my-chart-1.0.0.tgz oci://registry.example.com/charts

# 4. Verify
helm show chart oci://registry.example.com/charts/my-chart --version 1.0.0
```

### Using Docker-style Commands

```bash
# 1. Save chart as OCI image
helm chart save ./my-chart registry.example.com/charts/my-chart:1.0.0

# 2. Push image
helm chart push registry.example.com/charts/my-chart:1.0.0
```

## CI/CD Integration Examples

### GitHub Actions

```yaml
name: Push Helm Chart to GHCR

on:
  push:
    tags:
      - 'v*'

jobs:
  push-chart:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Helm
      uses: azure/setup-helm@v3
      with:
        version: v3.12.0
    
    - name: Login to GHCR
      run: |
        echo ${{ secrets.GITHUB_TOKEN }} | \
          helm registry login ghcr.io -u ${{ github.actor }} --password-stdin
    
    - name: Package and Push Chart
      run: |
        VERSION=${GITHUB_REF#refs/tags/v}
        helm package charts/my-chart --version $VERSION
        helm push my-chart-${VERSION}.tgz oci://ghcr.io/${{ github.repository_owner }}/charts
```

### GitLab CI

```yaml
push-chart:
  stage: deploy
  image: alpine/helm:latest
  script:
    - helm registry login $CI_REGISTRY -u $CI_REGISTRY_USER -p $CI_REGISTRY_PASSWORD
    - helm package charts/my-chart --version $CI_COMMIT_TAG
    - helm push my-chart-${CI_COMMIT_TAG}.tgz oci://$CI_REGISTRY/$CI_PROJECT_PATH/charts
  only:
    - tags
```

## Troubleshooting

### 1. Authentication Failures

```bash
# Test OCI registry connection
helm registry login oci://registry.example.com

# Inspect Secret contents
kubectl get secret oci-auth -o jsonpath='{.data.username}' | base64 -d
kubectl get secret oci-auth -o jsonpath='{.data.password}' | base64 -d
```

### 2. Chart Pull Failures

```bash
# Test chart pull manually
helm pull oci://registry.example.com/charts/my-chart --version 1.0.0

# Check HelmRelease status
kubectl describe helmrelease my-app
kubectl get events --sort-by='.lastTimestamp' | grep HelmRelease
```

### 3. Version Not Found

```bash
# List available tags (if registry supports it)
curl -u username:password https://registry.example.com/v2/charts/my-chart/tags/list

# Using Helm CLI
helm show chart oci://registry.example.com/charts/my-chart --version 1.0.0
```

## Best Practices

### 1. Versioning

- Use semantic versioning (SemVer)
- Create Git tags for each release
- Keep chart version and app version in sync

### 2. Security

- Use dedicated Service Accounts or Robot Accounts
- Rotate credentials regularly
- Use secret managers (Vault, External Secrets Operator)
- Enable TLS/HTTPS

### 3. Performance

- Set a reasonable `interval` (e.g. 1–2 hours)
- Use registry caching where possible
- Deploy registry and Kubernetes cluster in the same region when feasible

### 4. Monitoring and Alerts

- Monitor HelmRepository sync status
- Alert on chart pull failures
- Track credential expiration

## References

- [Helm OCI Support](https://helm.sh/docs/topics/registries/)
- [OCI Artifacts Specification](https://github.com/opencontainers/artifacts)
- [GHCR Documentation](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
- [Harbor Documentation](https://goharbor.io/docs/)

---

**Last updated**: 2026-02-11  
**Version**: v0.3.0-dev
