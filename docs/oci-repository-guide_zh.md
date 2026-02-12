# OCI Helm 仓库指南

[English](oci-repository-guide-en.md) | 中文

本文档提供使用 OCI (Open Container Initiative) Helm 仓库的完整示例。

## 概述

从 v0.3.0 开始，Helm Operator 支持 OCI 格式的 Helm 仓库。OCI 仓库使用 Docker Registry 协议，提供了更好的带宽优化和版本管理能力。

## OCI vs 传统 Helm 仓库

| 特性 | 传统 Helm 仓库 | OCI 仓库 |
| --- | ------------- | ------- |
| **协议** | HTTP/HTTPS | OCI (Docker Registry) |
| **URL 格式** | `https://charts.example.com` | `oci://registry.example.com/charts` |
| **索引文件** | `index.yaml` | 不需要 |
| **Chart 列表** | 可获取完整列表 | 不支持列表操作 |
| **认证** | Basic Auth, TLS | Docker Registry Auth |
| **带宽优化** | 需要下载完整索引 | 按需拉取 |

## 示例 1: 公开 GitHub Container Registry (GHCR)

### 1.1 注册 OCI 仓库

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

### 1.2 使用 OCI Chart

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

## 示例 2: 私有 Azure Container Registry (ACR)

### 2.1 创建认证 Secret

```bash
# 获取 ACR 登录凭证
az acr login --name myregistry
ACR_USERNAME=$(az acr credential show --name myregistry --query username -o tsv)
ACR_PASSWORD=$(az acr credential show --name myregistry --query passwords[0].value -o tsv)

# 创建 Kubernetes Secret
kubectl create secret generic acr-auth \
  --from-literal=username=$ACR_USERNAME \
  --from-literal=password=$ACR_PASSWORD \
  --namespace default
```

或使用 YAML：

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

### 2.2 注册私有 OCI 仓库

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

### 2.3 部署 Chart

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

## 示例 3: Amazon Elastic Container Registry (ECR)

### 3.1 配置 ECR 认证

```bash
# 获取 ECR 登录令牌
aws ecr get-login-password --region us-east-1 | \
  docker login --username AWS --password-stdin 123456789012.dkr.ecr.us-east-1.amazonaws.com

# 创建 Secret (注意: ECR token 有时效性，建议使用 IAM roles)
kubectl create secret generic ecr-auth \
  --from-literal=username=AWS \
  --from-literal=password=$(aws ecr get-login-password --region us-east-1) \
  --namespace default
```

**推荐方式**: 使用 [external-secrets-operator](https://external-secrets.io/) 自动刷新 ECR token：

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: ecr-auth
  namespace: default
spec:
  refreshInterval: 1h  # ECR token 有效期 12 小时
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

### 3.2 注册 ECR OCI 仓库

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

## 示例 4: Google Artifact Registry (GAR)

### 4.1 配置 GAR 认证

```bash
# 使用 gcloud 生成访问令牌
gcloud auth print-access-token | \
  docker login -u oauth2accesstoken --password-stdin https://us-central1-docker.pkg.dev

# 创建 Service Account 密钥 (推荐)
gcloud iam service-accounts create helm-operator \
  --display-name="Helm Operator Service Account"

gcloud iam service-accounts keys create key.json \
  --iam-account=helm-operator@PROJECT_ID.iam.gserviceaccount.com

# 授予 Artifact Registry Reader 权限
gcloud projects add-iam-policy-binding PROJECT_ID \
  --member="serviceAccount:helm-operator@PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/artifactregistry.reader"

# 创建 Secret
kubectl create secret generic gar-auth \
  --from-literal=username=_json_key \
  --from-file=password=key.json \
  --namespace default
```

### 4.2 注册 GAR 仓库

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

## 示例 5: Harbor OCI Registry

### 5.1 创建 Harbor 项目和用户

1. 登录 Harbor Web UI
2. 创建项目 `helm-charts`
3. 创建 Robot Account 或使用用户凭证

### 5.2 配置 Harbor 认证

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

### 5.3 注册 Harbor OCI 仓库

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
      insecureSkipVerify: false  # 生产环境建议使用有效证书
```

### 5.4 使用 Harbor Chart

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

## 推送 Chart 到 OCI Registry

### 使用 Helm CLI

```bash
# 1. 打包 chart
helm package ./my-chart

# 2. 登录 OCI registry
helm registry login oci://registry.example.com

# 3. 推送 chart
helm push my-chart-1.0.0.tgz oci://registry.example.com/charts

# 4. 验证
helm show chart oci://registry.example.com/charts/my-chart --version 1.0.0
```

### 使用 Docker CLI

```bash
# 1. 将 chart 转换为 OCI image
helm chart save ./my-chart registry.example.com/charts/my-chart:1.0.0

# 2. 推送 image
helm chart push registry.example.com/charts/my-chart:1.0.0
```

## CI/CD 集成示例

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

## 故障排查

### 1. 认证失败

```bash
# 测试 OCI registry 连接
helm registry login oci://registry.example.com

# 验证 Secret 内容
kubectl get secret oci-auth -o jsonpath='{.data.username}' | base64 -d
kubectl get secret oci-auth -o jsonpath='{.data.password}' | base64 -d
```

### 2. Chart 拉取失败

```bash
# 手动测试 chart 拉取
helm pull oci://registry.example.com/charts/my-chart --version 1.0.0

# 查看 HelmRelease 状态
kubectl describe helmrelease my-app
kubectl get events --sort-by='.lastTimestamp' | grep HelmRelease
```

### 3. 版本不存在

```bash
# 列出可用的 tags (需要 registry 支持)
curl -u username:password https://registry.example.com/v2/charts/my-chart/tags/list

# 使用 Helm CLI 查看
helm show chart oci://registry.example.com/charts/my-chart --version 1.0.0
```

## 最佳实践

### 1. 版本管理

- 使用语义化版本号 (SemVer)
- 为每个 release 创建 Git tag
- 保持 Chart 版本和 App 版本同步

### 2. 安全性

- 使用专用的 Service Account 或 Robot Account
- 定期轮换密钥
- 使用 Secret 管理工具 (Vault, External Secrets Operator)
- 启用 TLS/HTTPS

### 3. 性能优化

- 合理设置 `interval` (1-2 小时)
- 使用镜像缓存加速
- 在同一 region 部署 registry 和 Kubernetes 集群

### 4. 监控和告警

- 监控 HelmRepository sync 状态
- 设置 chart 拉取失败告警
- 跟踪认证过期时间

## 参考资源

- [Helm OCI Support](https://helm.sh/docs/topics/registries/)
- [OCI Artifacts Specification](https://github.com/opencontainers/artifacts)
- [GHCR Documentation](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
- [Harbor Documentation](https://goharbor.io/docs/)

---

**最后更新**: 2026-02-11  
**版本**: v0.2.3
