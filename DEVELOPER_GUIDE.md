# Helm Operator 开发者指南

## 概述

Helm Operator 是一个 Kubernetes 控制器，用于管理 Helm 发布的生命周期。它通过 Kubernetes 自定义资源（CRD）提供声明式的 Helm 发布管理功能。

## 架构概览

### 核心组件

```txt
helm-operator/
├── api/v1alpha1/           # CRD 定义
│   ├── helmrelease_types.go    # HelmRelease 资源定义
│   └── helmrepository_types.go # HelmRepository 资源定义
├── internal/
│   ├── controller/         # 控制器逻辑
│   │   ├── helmrelease_controller.go
│   │   └── helmrepository_controller.go
│   ├── helm/              # Helm 客户端封装
│   │   ├── client.go
│   │   ├── release.go
│   │   └── repository.go
│   └── utils/             # 工具函数
├── config/                # Kubernetes 配置
└── cmd/main.go           # 程序入口
```

### 资源类型

1. **HelmRepository**: 定义 Helm 仓库配置
2. **HelmRelease**: 定义 Helm 发布配置

## 开发环境设置

### 前置条件

- Go 1.21+
- Docker
- Kubernetes 集群（本地或远程）
- kubectl
- Helm 3.x
- kubebuilder

### 安装依赖

```bash
# 克隆项目
git clone https://github.com/ketches/helm-operator.git
cd helm-operator

# 安装依赖
go mod download
```

### 构建项目

```bash
# 构建二进制文件
make build

# 构建 Docker 镜像
make docker-build

# 运行测试
make test
```

## 核心概念

### HelmRepository

HelmRepository 定义了 Helm 仓库的配置信息，支持多种类型的仓库：

#### 1. 公共 HTTPS 仓库

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
```

#### 2. 私有 HTTPS 仓库（带认证）

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

#### 3. 私有 HTTP 仓库（内网环境）

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

**关键字段说明：**

- `url`: Helm 仓库 URL，支持 `https://`、`http://` 协议
- `type`: 仓库类型（当前只支持 `helm`，后续可能会添加对 `oci` 的支持）
- `interval`: 同步间隔
- `timeout`: 操作超时时间
- `auth`: 认证配置（可选）
  - `basic`: 基础认证（用户名/密码）
  - `tls`: TLS 配置
- `suspend`: 是否暂停同步

### HelmRelease

HelmRelease 定义了 Helm 发布的配置：

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

**关键字段说明：**

- `chart`: Chart 配置（名称、版本、仓库）
- `release`: 发布配置（名称、命名空间）
- `values`: Helm values 配置
- `install/upgrade`: 安装/升级选项
- `interval`: 协调间隔

## 开发指南

### 添加新功能

1. **修改 CRD 定义**

   ```bash
   # 编辑 API 类型定义
   vim api/v1alpha1/helmrelease_types.go
   
   # 重新生成代码
   make generate
   
   # 更新 CRD
   make manifests
   ```

2. **更新控制器逻辑**

   ```bash
   # 编辑控制器
   vim internal/controller/helmrelease_controller.go
   
   # 添加业务逻辑
   # 更新协调循环
   ```

3. **添加测试**

   ```bash
   # 添加单元测试
   vim internal/controller/helmrelease_controller_test.go
   
   # 运行测试
   make test
   ```

### 控制器开发模式

#### 协调循环（Reconciliation Loop）

```go
func (r *HelmReleaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // 1. 获取资源
    release := &helmoperatorv1alpha1.HelmRelease{}
    if err := r.Get(ctx, req.NamespacedName, release); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // 2. 处理删除逻辑
    if !release.DeletionTimestamp.IsZero() {
        return r.reconcileDelete(ctx, release)
    }

    // 3. 添加 Finalizer
    if !controllerutil.ContainsFinalizer(release, utils.HelmReleaseFinalizer) {
        controllerutil.AddFinalizer(release, utils.HelmReleaseFinalizer)
        return ctrl.Result{}, r.Update(ctx, release)
    }

    // 4. 执行主要逻辑
    return r.reconcileNormal(ctx, release)
}
```

#### 状态管理

```go
// 更新状态条件
condition := utils.NewReleaseReadyCondition(metav1.ConditionTrue, "InstallCompleted", "Release is ready")
meta.SetStatusCondition(&release.Status.Conditions, condition)

// 更新状态
return r.Status().Update(ctx, release)
```

### Helm 客户端使用

#### 创建 Helm 客户端

```go
helmClient, err := helm.NewClient("default")
if err != nil {
    return fmt.Errorf("failed to create helm client: %w", err)
}
```

#### 安装发布

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

#### Chart 引用格式

Chart 引用支持多种格式：

1. **仓库引用**: `repository_name/chart_name`

   ```go
   chartRef := fmt.Sprintf("%s/%s", repoName, chartName)
   // 例如: "bitnami/nginx", "private-repo/myapp"
   ```

2. **直接 URL**: 使用 `repositoryURL` 字段

   ```yaml
   spec:
     chart:
       name: "nginx"
       repositoryURL: "https://charts.bitnami.com/bitnami"
   ```

3. **本地 Chart**: 直接使用 chart 名称

   ```yaml
   spec:
     chart:
       name: "./local-chart"
   ```

#### 支持的仓库类型

| 仓库类型 | URL 格式 | 示例 | 用途 |
|---------|---------|------|------|
| 公共 HTTPS | `https://` | `https://charts.bitnami.com/bitnami` | 公共 Helm 仓库 |
| 私有 HTTPS | `https://` | `https://private.charts.example.com` | 企业私有仓库 |
| 内网 HTTP | `http://` | `http://charts.internal:8080` | 内网私有仓库 |

#### 认证配置

对于私有仓库，支持多种认证方式：

1. **基础认证（用户名/密码）**

   ```yaml
   auth:
     basic:
       secretRef:
         name: "repo-credentials"
         namespace: "default"
   ```

2. **TLS 配置**

   ```yaml
   auth:
     tls:
       insecureSkipVerify: true  # 跳过证书验证（HTTP 仓库）
       secretRef:
         name: "tls-config"
         namespace: "default"
   ```

3. **认证 Secret 格式**

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

## 测试指南

### 单元测试

```bash
# 运行所有测试
make test

# 运行特定包的测试
go test ./internal/controller/...

# 运行测试并查看覆盖率
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### 集成测试

```bash
# 启动测试环境
make test-integration

# 运行端到端测试
make test-e2e
```

### 本地调试

1. **运行控制器**

   ```bash
   # 设置 KUBECONFIG
   export KUBECONFIG=~/.kube/config
   
   # 运行控制器
   go run cmd/main.go
   ```

2. **应用测试资源**

   ```bash
   # 应用 HelmRepository
   kubectl apply -f config/samples/helm-operator_v1alpha1_helmrepository.yaml
   
   # 应用 HelmRelease
   kubectl apply -f config/samples/helm-operator_v1alpha1_helmrelease.yaml
   ```

3. **查看日志**

   ```bash
   # 查看控制器日志
   kubectl logs -f deployment/helm-operator-controller-manager -n helm-operator-system
   ```

## 故障排除

### 常见问题

1. **Chart 引用错误**

   ```txt
   错误: non-absolute URLs should be in form of repo_name/path_to_chart, got: myapp
   解决: 确保使用正确的 chart 引用格式 "repository_name/chart_name"
   ```

2. **Kubernetes 客户端初始化失败**

   ```txt
   错误: kubernetes client not initialized in Helm configuration
   解决: 检查 KUBECONFIG 设置和集群连接
   ```

3. **权限不足**

   ```txt
   错误: forbidden: User cannot create resource
   解决: 检查 RBAC 配置和服务账户权限
   ```

### 调试技巧

1. **启用详细日志**

   ```bash
   # 设置日志级别
   export LOG_LEVEL=debug
   go run cmd/main.go
   ```

2. **查看资源状态**

   ```bash
   # 查看 HelmRelease 状态
   kubectl describe helmrelease myapp-sample
   
   # 查看事件
   kubectl get events --sort-by=.metadata.creationTimestamp
   ```

3. **使用 Helm CLI 验证**

   ```bash
   # 列出发布
   helm list -A
   
   # 查看发布详情
   helm get all <release-name> -n <namespace>
   ```

## 贡献指南

### 代码规范

1. **Go 代码风格**
   - 遵循 `gofmt` 格式化
   - 使用 `golint` 检查代码质量
   - 添加适当的注释和文档

2. **提交信息格式**

   ```txt
   <type>(<scope>): <subject>
   
   <body>
   
   <footer>
   ```

   例如:

   ```txt
   feat(controller): add support for OCI repositories
   
   - Add OCI repository type support
   - Update chart reference handling
   - Add integration tests
   
   Fixes #123
   ```

3. **测试要求**
   - 新功能必须包含单元测试
   - 测试覆盖率不低于 80%
   - 集成测试验证端到端功能

### 发布流程

1. **版本标记**

   ```bash
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin v1.0.0
   ```

2. **构建发布**

   ```bash
   make release VERSION=v1.0.0
   ```

## 参考资源

- [Kubernetes Controller Runtime](https://github.com/kubernetes-sigs/controller-runtime)
- [Helm Go SDK](https://helm.sh/docs/topics/advanced/#go-sdk)
- [Kubebuilder Book](https://book.kubebuilder.io/)
- [Operator Pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)

## 许可证

本项目采用 Apache 2.0 许可证。详见 [LICENSE](LICENSE) 文件。
