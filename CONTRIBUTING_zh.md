# 贡献指南

首先，感谢您考虑为 Helm Operator 做出贡献！正是因为有像您这样的人，Helm Operator 才成为一个如此出色的工具。

我们欢迎任何形式的贡献，而不仅仅是代码。您可以通过以下方式提供帮助：

* **报告错误**：如果您发现错误，请通过创建 issue 来报告。
* **建议功能**：如果您对新功能有任何想法，请创建一个 issue 来进行讨论。
* **改进文档**：如果您发现文档有任何改进空间，请提交拉取请求。
* **编写代码**：如果您想贡献代码，请阅读以下指南。

## 入门

在开始之前，请确保您已按照 [开发者指南](DEVELOPER_GUIDE_zh.md) 中的说明设置好您的开发环境。

### Fork 和克隆仓库

1. 在 GitHub 上 Fork 本仓库。
2. 在本地克隆您的 Fork：

   ```bash
   git clone https://github.com/您的用户名/helm-operator.git
   cd helm-operator
   ```

3. 添加上游仓库：

    ```bash
    git remote add upstream https://github.com/ketches/helm-operator.git
    ```

### 创建分支

为您的更改创建一个新分支：

```bash
git checkout -b my-feature-branch
```

## 开发流程

### 代码风格

* **Go**：所有 Go 代码必须使用 `gofmt` 进行格式化。我们还使用 `golint` 来检查代码风格问题。
* **YAML**：所有 YAML 文件应使用 2 个空格进行缩进。
* **文档**：所有文档都应使用 Markdown 编写。

### 提交信息格式

我们遵循 [Conventional Commits](https://www.conventionalcommits.org/zh/v1.0.0/) 规范。您的提交信息应按以下格式组织：

```txt
<类型>(<范围>): <主题>

[可选的正文]

[可选的页脚]
```

**示例：**

```txt
feat(controller): add support for OCI repositories

- Add OCI repository type support
- Update chart reference handling
- Add integration tests

Fixes #123
```

* **类型**: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`, `ci`, `build`。
* **范围**: 您正在更改的代码库部分（例如 `controller`, `api`, `helm`）。

### 测试

* 新功能必须包含单元测试。
* 测试覆盖率不应低于 80%。
* 在适当的情况下，添加集成测试以验证端到端功能。

您可以使用以下命令运行测试：

```bash
# 运行单元测试
make test

# 运行集成测试
make test-integration

# 运行端到端测试
make test-e2e
```

### 提交拉取请求

1. 将您的更改推送到您的 Fork：

   ```bash
   git push origin my-feature-branch
   ```

2. 从您的 Fork 创建一个到 `ketches/helm-operator` 仓库 `main` 分支的拉取请求。
3. 在拉取请求的描述中，请说明您的更改并引用任何相关问题。
4. 确保所有状态检查都通过。

## 行为准则

本项目及所有参与者均受我们的 [行为准则](CODE_OF_CONDUCT.md) 约束。通过参与，您应遵守此准则。请举报不可接受的行为。

感谢您的贡献！
