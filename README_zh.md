# Helm Operator

[English](README.md) | 中文文档

> **版本**: v0.3.0

一个生产级的 Kubernetes Operator，通过自定义资源定义（CRD）管理 Helm 仓库和发布，具有智能自动化和高级功能。

## 概述

Helm Operator 提供了一种声明式的方式来管理 Kubernetes 集群中的 Helm 仓库和发布。它通过自定义资源扩展 Kubernetes，让你能够：

- **🏪 管理 Helm 仓库**: 智能错误处理和重试的自动同步
- **🚀 管理 Helm 发布**: 声明式安装、升级，支持自动回滚
- **📦 OCI 仓库支持**: 完整支持基于 OCI 的 Helm Charts（推荐）
- **🔄 自动回滚**: 从失败的升级自动恢复
- **📊 语义化版本**: SemVer 约束实现灵活的版本管理
- **🔐 认证支持**: 支持使用 Basic Auth 和 TLS 的私有仓库
- **📈 完整可观测性**: Prometheus 指标和全面的日志

## 功能特性

### 🏪 HelmRepository 管理

- **OCI 仓库支持**（生产环境推荐）
- 自动仓库同步
- Chart 发现和版本跟踪
- 认证支持（Basic Auth、TLS、OCI 认证）
- 带有 Chart 信息的状态报告
- 智能重试的可配置同步间隔
- ConfigMap 策略（disabled/on-demand/lazy）

### 🚀 HelmRelease 管理

- 声明式发布管理
- 基于 YAML 的 values 配置
- **升级失败自动回滚** 🆕
- **SemVer 版本约束** 🆕
- 发布间的依赖管理
- 回滚和历史跟踪
- Health check 集成

### 🔐 安全与认证

- 私有仓库支持
- OCI 仓库认证
- TLS 证书管理
- Kubernetes Secret 集成
- RBAC 权限

### 📊 可观测性

- **Prometheus 指标**（15+ 指标）🆕
- 实时状态条件
- 事件记录
- 结构化输出的全面日志
- Grafana dashboard 就绪

## 架构

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

## 快速开始

### 前置条件

- Kubernetes 集群 v1.25+
- 配置好的 kubectl 访问集群
- Go 1.21+（用于开发）
- Docker（用于构建镜像）

### 安装

#### 通过 Helm 安装（推荐）

```bash
# 添加 Helm 仓库
helm repo add helm-operator https://ketches.github.io/helm-operator
helm repo update

# 安装 operator
helm install helm-operator helm-operator/helm-operator \
  -n ketches --create-namespace

# 验证安装
kubectl get pods -n ketches
```

#### 通过 manifests 安装

```bash
# 安装 CRDs
kubectl apply -f https://raw.githubusercontent.com/ketches/helm-operator/master/deploy/crds/

# 部署 Operator
kubectl create namespace ketches
kubectl apply -f https://raw.githubusercontent.com/ketches/helm-operator/master/deploy/manifests.yaml
```

### 基本使用

#### 方法 1: 使用 OCI 仓库（推荐）

OCI 仓库提供更好的性能、安全性，是 Helm Chart 分发的未来。

步骤 1: **创建基于 OCI 的 HelmRepository**

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
  # 优化 ConfigMap 使用
  valuesConfigMapPolicy: disabled  # 推荐
```

步骤 2: **从 OCI 部署 Release**

```yaml
apiVersion: helm-operator.ketches.cn/v1alpha1
kind: HelmRelease
metadata:
  name: my-app
  namespace: default
spec:
  chart:
    name: myapp
    version: "^1.0.0"  # SemVer 约束
    ociRepository: "oci://ghcr.io/myorg/charts/myapp"
  
  release:
    name: my-app
    namespace: production
    createNamespace: true
  
  values: |
    replicaCount: 3
    image:
      tag: "v1.0.0"
  
  # 启用自动回滚以确保安全
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

**应用资源:**

```bash
kubectl apply -f helmrepository-oci.yaml
kubectl apply -f helmrelease-oci.yaml

# 检查状态
kubectl get helmrepository ghcr-charts
kubectl get helmrelease my-app
```

#### 方法 2: 使用传统 HTTP 仓库

步骤 1: **创建传统 HelmRepository**

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
  valuesConfigMapPolicy: disabled  # 推荐
```

步骤 2: **部署 Release**

```yaml
apiVersion: helm-operator.ketches.cn/v1alpha1
kind: HelmRelease
metadata:
  name: nginx
  namespace: default
spec:
  chart:
    name: nginx
    version: "~15.0.0"  # 跟踪 minor 版本
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

#### 检查状态和事件

```bash
# 检查仓库同步状态
kubectl describe helmrepository ghcr-charts

# 检查发布状态
kubectl get helmrelease my-app -o yaml

# 查看事件
kubectl get events --field-selector involvedObject.name=my-app

# 检查 Prometheus 指标（如果已配置）
kubectl port-forward -n ketches svc/helm-operator-metrics 8080:8080
curl http://localhost:8080/metrics | grep helm_
```

## 开发

### 本地开发环境搭建

1. **克隆仓库:**

```bash
git clone https://github.com/ketches/helm-operator.git
cd helm-operator
```

2. **安装依赖:**

```bash
make generate
make manifests
```

3. **本地运行:**

```bash
make install  # 安装 CRDs
make run      # 本地运行控制器
```

4. **构建和测试:**

```bash
make build    # 构建二进制文件
make test     # 运行测试
```

### 构建本地 Docker 镜像

```bash
make docker-build-local IMG=helm-operator VERSION=dev
```

### 部署到集群

```bash
make deploy
```

## 配置示例

### OCI 仓库认证（推荐）

#### 公共 OCI 仓库（GitHub Container Registry）

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

#### 私有 OCI 仓库认证

```yaml
# 创建认证 secret
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

### 生产就绪的自动回滚配置

```yaml
apiVersion: helm-operator.ketches.cn/v1alpha1
kind: HelmRelease
metadata:
  name: production-app
  namespace: production
spec:
  chart:
    name: myapp
    version: "^2.0.0"  # 自动更新到兼容版本
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
  
  # 关键: 启用自动回滚
  rollback:
    enabled: true
    toRevision: 0       # 回滚到前一个版本
    timeout: "5m"
    wait: true
    cleanupOnFail: true
  
  install:
    timeout: "15m"
    wait: true
    waitForJobs: true
  
  upgrade:
    timeout: "15m"
    wait: true          # 自动回滚检测需要
    waitForJobs: true
    cleanupOnFail: true
  
  # 每 12 小时检查更新
  interval: "12h"
```

### 版本约束示例

```yaml
apiVersion: helm-operator.ketches.cn/v1alpha1
kind: HelmRelease
metadata:
  name: app-with-constraints
spec:
  chart:
    name: myapp
    # 语义化版本约束
    version: "^1.2.0"     # >= 1.2.0, < 2.0.0 (生产推荐)
    # version: "~1.2.0"   # >= 1.2.0, < 1.3.0 (保守策略)
    # version: ">=1.0.0, <2.0.0"  # 范围
    # version: "1.2.3"    # 精确版本 (最稳定)
    # version: "latest"   # 始终最新 (仅开发环境)
    ociRepository: "oci://ghcr.io/charts/myapp"
  
  release:
    name: my-app
    namespace: default
  
  # 回滚保护
  rollback:
    enabled: true
```

## 多云 OCI 示例

### GitHub Container Registry (GHCR)

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

### Azure Container Registry (ACR)

```yaml
# 创建 ACR 认证 secret
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
```

### Google Artifact Registry (GAR)

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
      name: gar-auth  # GCP 服务账号密钥
```

## 监控与可观测性

### Prometheus 指标

```promql
# 仓库同步成功率
sum(rate(helm_repository_sync_total{status="success"}[5m])) 
  / 
sum(rate(helm_repository_sync_total[5m]))

# Release 操作 P95 延迟
histogram_quantile(0.95, 
  sum(rate(helm_release_operation_duration_seconds_bucket[5m])) 
  by (le, operation))

# 自动回滚频率
sum(increase(helm_release_rollbacks_total[1h])) by (release, status)
```

### 告警规则

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

## API 参考

详细的 API 文档，请参见：

- [HelmRepository API](docs/api-reference.md#helmrepository)
- [HelmRelease API](docs/api-reference.md#helmrelease)

## 贡献

我们欢迎贡献！请查看我们的[贡献指南](./CONTRIBUTING_zh.md)和[开发者指南](./DEVELOPER_GUIDE_zh.md)了解详情。

### 开发工作流

1. Fork 仓库
2. 创建功能分支
3. 进行修改
4. 添加测试
5. 运行 `make test lint`
6. 提交 Pull Request

## 许可证

本项目使用 Apache License 2.0 许可证 - 详见 [LICENSE](LICENSE) 文件。

## 支持

- 📖 [文档](docs/)
- 🐛 [问题跟踪](https://github.com/ketches/helm-operator/issues)
- 💬 [讨论](https://github.com/ketches/helm-operator/discussions)
