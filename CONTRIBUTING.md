# Contributing to Helm Operator

First off, thank you for considering contributing to Helm Operator! It's people like you that make Helm Operator such a great tool.

We welcome any type of contribution, not just code. You can help with:

* **Reporting a bug**: If you find a bug, please report it by creating an issue.
* **Suggesting a feature**: If you have an idea for a new feature, please create an issue to discuss it.
* **Improving documentation**: If you see any room for improvement in the documentation, please submit a pull request.
* **Writing code**: If you want to contribute with code, please read the following guidelines.

## Getting Started

Before you start, please make sure you have set up your development environment as described in the [Developer Guide](DEVELOPER_GUIDE.md).

### Fork and Clone the Repository

1. Fork the repository on GitHub.
2. Clone your fork locally:

   ```bash
   git clone https://github.com/YOUR_USERNAME/helm-operator.git
   cd helm-operator
   ```

3. Add the upstream repository:

    ```bash
    git remote add upstream https://github.com/ketches/helm-operator.git
    ```

### Create a Branch

Create a new branch for your changes:

```bash
git checkout -b my-feature-branch
```

## Development Process

### Code Style

* **Go**: All Go code must be formatted with `gofmt`. We also use `golint` to check for style issues.
* **YAML**: All YAML files should be indented with 2 spaces.
* **Documentation**: All documentation should be written in Markdown.

### Commit Message Format

We follow the [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) specification. Your commit messages should be structured as follows:

```txt
<type>(<scope>): <subject>

[optional body]

[optional footer]
```

**Example:**

```txt
feat(controller): add support for OCI repositories

- Add OCI repository type support
- Update chart reference handling
- Add integration tests

Fixes #123
```

* **Types**: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`, `ci`, `build`.
* **Scope**: The part of the codebase you are changing (e.g., `controller`, `api`, `helm`).

### Testing

* New features must include unit tests.
* Test coverage should not be below 80%.
* Add integration tests to verify end-to-end functionality where appropriate.

You can run the tests with the following commands:

```bash
# Run unit tests
make test

# Run integration tests
make test-integration

# Run end-to-end tests
make test-e2e
```

### Submitting a Pull Request

1. Push your changes to your fork:

   ```bash
   git push origin my-feature-branch
   ```

2. Create a pull request from your fork to the `main` branch of the `ketches/helm-operator` repository.
3. In the pull request description, please describe your changes and reference any related issues.
4. Make sure all status checks are passing.

## Code of Conduct

This project and everyone participating in it is governed by our [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code. Please report unacceptable behavior.

Thank you for your contribution!
