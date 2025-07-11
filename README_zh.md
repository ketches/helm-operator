# Helm Operator

[English](README.md) | 中文文档

一个通过自定义资源定义（CRD）管理 Helm 仓库和发布的 Kubernetes Operator。

## 概述

Helm Operator 提供了一种声明式的方式来管理 Kubernetes 集群中的 Helm 仓库和发布。它通过自定义资源扩展 Kubernetes，让你能够：

- **管理 Helm 仓库**: 自动同步 Helm 仓库并跟踪可用的 Charts
- **管理 Helm 发布**: 声明式地安装、升级和管理 Helm 发布
- **认证支持**: 支持使用 Basic Auth 和 TLS 的私有仓库
- **状态跟踪**: 实时状态更新和 Chart 信息
- **事件记录**: 全面的操作事件日志

## 功能特性

### 🏪 HelmRepository 管理

- 自动仓库同步
- Chart 发现和版本跟踪
- 认证支持（Basic Auth、TLS）
- 带有 Chart 信息的状态报告
- 可配置的同步间隔

### 🚀 HelmRelease 管理

- 声明式发布管理
- 基于 YAML 的 values 配置
- 配置变更时自动升级
- 发布间的依赖管理
- 回滚和历史跟踪

### 🔐 安全与认证

- 私有仓库支持
- TLS 证书管理
- Kubernetes Secret 集成
- RBAC 权限

### 📊 可观测性

- 实时状态条件
- 事件记录
- 监控和指标就绪
- 全面的日志记录

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

#### 通过 manifests 安装

1. **安装 CRDs:**

```bash
kubectl apply -f https://raw.githubusercontent.com/ketches/helm-operator/master/deploy/crds/helm-operator.ketches.cn_helmrepositories.yaml
kubectl apply -f https://raw.githubusercontent.com/ketches/helm-operator/master/deploy/crds/helm-operator.ketches.cn_helmreleases.yaml
```

2. **部署 Operator:**

```bash
kubectl create namespace ketches
kubectl apply -f https://raw.githubusercontent.com/ketches/helm-operator/master/deploy/manifests.yaml
```

#### 通过 Helm 安装

1. **添加 Helm 仓库:**

```bash
helm repo add helm-operator https://ketches.github.io/helm-operator
helm repo update
```

2. **安装 operator:**

```bash
helm install helm-operator helm-operator/helm-operator -n ketches --create-namespace
```

3. **验证安装:**

```bash
kubectl get pods -n ketches
```

### 基本使用

#### 1. 创建 Helm 仓库

```bash
kubectl apply -f https://raw.githubusercontent.com/ketches/helm-operator/master/samples/helm_repository.yaml
```

#### 2. 创建 Helm 发布

```bash
kubectl apply -f https://raw.githubusercontent.com/ketches/helm-operator/master/samples/helm_release.yaml
```

#### 3. 检查状态

```bash
# 检查仓库状态
kubectl get helmrepository helm-operator-charts

# 检查发布状态
kubectl get helmrelease nginx
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
make docker-build-local IMG=helm-operator TAG=dev
```

### 部署到集群

```bash
make deploy
```

## 配置

### HelmRepository 配置

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

### HelmRelease 配置

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

## API 参考

详细的 API 文档，请参见：

- [HelmRepository API](.dev/api-reference.md#helmrepository)
- [HelmRelease API](.dev/api-reference.md#helmrelease)

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

## 路线图

- [x] HelmRepository 管理
- [x] HelmRelease 管理
- [ ] OCI 仓库支持
- [ ] Webhook 验证

## 示例

### 私有仓库认证

```yaml
# 创建认证 Secret
apiVersion: v1
kind: Secret
metadata:
  name: private-repo-auth
type: Opaque
data:
  username: dXNlcm5hbWU=  # base64 encoded
  password: cGFzc3dvcmQ=  # base64 encoded
---
# 使用认证的私有仓库
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

### 复杂的 Release 配置

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
    # 应用配置
    app:
      name: "complex-app"
      version: "2.0.0"
    
    # 副本数
    replicaCount: 3
    
    # 镜像配置
    image:
      repository: "my-registry/my-app"
      tag: "v2.0.0"
      pullPolicy: "IfNotPresent"
    
    # 服务配置
    service:
      type: "ClusterIP"
      port: 8080
      targetPort: 8080
    
    # Ingress 配置
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
    
    # 资源限制
    resources:
      limits:
        cpu: "1000m"
        memory: "1Gi"
      requests:
        cpu: "500m"
        memory: "512Mi"
    
    # 环境变量
    env:
      - name: "APP_ENV"
        value: "production"
      - name: "DB_HOST"
        value: "postgres.database.svc.cluster.local"
  
  # 安装配置
  install:
    timeout: "15m"
    wait: true
    waitForJobs: true
  
  # 升级配置
  upgrade:
    timeout: "15m"
    wait: true
    cleanupOnFail: true
  
  # 依赖关系
  dependsOn:
    - name: "postgres"
      namespace: "database"
```

---

**注意**: 本项目正在积极开发中。在 v1.0.0 发布之前，API 可能会发生变化。
