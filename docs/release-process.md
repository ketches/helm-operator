# Release Process

This document describes the process for releasing a new version of the helm-operator.

## Version Management

The project follows [Semantic Versioning](https://semver.org/):

- **MAJOR** version when you make incompatible API changes
- **MINOR** version when you add functionality in a backwards compatible manner
- **PATCH** version when you make backwards compatible bug fixes

## Pre-Release Checklist

Before creating a new release, ensure:

1. [ ] All tests pass
2. [ ] Documentation is up to date
3. [ ] CHANGELOG.md is updated (if exists)
4. [ ] All desired features/fixes are merged
5. [ ] Version numbers are consistent across all files

## Release Steps

### 1. Update Version Information

Use the provided script to update version information across the project:

```bash
# Make the script executable (first time only)
chmod +x scripts/update-version.sh

# Update to new version
./scripts/update-version.sh 0.3.0
```

This script will update:

- `charts/helm-operator/Chart.yaml` (version and appVersion)
- `charts/helm-operator/values.yaml` (image tag if present)
- `README.md` (version references)

### 2. Review Changes

```bash
git diff
```

Verify that all version references have been updated correctly.

### 3. Commit Version Changes

```bash
git add .
git commit -m "chore: bump version to 0.3.0"
```

### 4. Create and Push Git Tag

```bash
# Create annotated tag with release notes
git tag -a v0.3.0 -m "Release v0.3.0

Features:
- List new features here

Bug Fixes:
- List bug fixes here

Breaking Changes:
- List any breaking changes here"

# Push the tag
git push origin v0.3.0
```

### 5. Build and Push Docker Image

```bash
# Build the Docker image
docker build -t ketches/helm-operator:0.3.0 .
docker build -t ketches/helm-operator:latest .

# Push to registry
docker push ketches/helm-operator:0.3.0
docker push ketches/helm-operator:latest
```

### 6. Package and Publish Helm Chart

```bash
# Package the Helm chart
helm package charts/helm-operator

# If you have a chart repository, push to it
# helm repo index . --url https://your-chart-repo.com
# Upload helm-operator-0.3.0.tgz to your chart repository
```

### 7. Create GitHub Release

1. Go to the [Releases page](https://github.com/ketches/helm-operator/releases)
2. Click "Create a new release"
3. Select the tag you just created (v0.3.0)
4. Fill in the release title and description
5. Attach any relevant files (e.g., packaged Helm chart)
6. Publish the release

## Automated Release (Optional)

You can create a GitHub Action to automate parts of this process. Here's an example workflow:

```yaml
# .github/workflows/release.yaml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    
    - name: Set up Docker Buildx
      uses: docker/setup-buildx-action@v3
    
    - name: Login to Docker Hub
      uses: docker/login-action@v3
      with:
        username: ${{ secrets.DOCKER_USERNAME }}
        password: ${{ secrets.DOCKER_PASSWORD }}
    
    - name: Extract version
      id: version
      run: echo "VERSION=${GITHUB_REF#refs/tags/v}" >> $GITHUB_OUTPUT
    
    - name: Build and push Docker image
      uses: docker/build-push-action@v5
      with:
        context: .
        push: true
        tags: |
          ketches/helm-operator:${{ steps.version.outputs.VERSION }}
          ketches/helm-operator:latest
    
    - name: Package Helm chart
      run: |
        helm package charts/helm-operator
    
    - name: Create GitHub Release
      uses: softprops/action-gh-release@v1
      with:
        files: helm-operator-*.tgz
        generate_release_notes: true
```

## Version File Locations

The following files contain version information that should be updated:

1. **charts/helm-operator/Chart.yaml**
   - `version`: Chart version
   - `appVersion`: Application version

2. **charts/helm-operator/values.yaml** (if applicable)
   - `image.tag`: Docker image tag

3. **README.md** (if applicable)
   - Installation examples
   - Docker image references

4. **Dockerfile** (if applicable)
   - LABEL version

## Rollback Process

If you need to rollback a release:

1. **Revert the Git tag:**

   ```bash
   git tag -d v0.3.0
   git push origin :refs/tags/v0.3.0
   ```

2. **Remove Docker images** (if possible)

3. **Create a new patch release** with fixes

## Best Practices

1. **Always test before releasing** - Use a staging environment
2. **Keep release notes detailed** - Help users understand changes
3. **Follow semantic versioning** - Don't break compatibility unexpectedly
4. **Coordinate with team** - Ensure everyone is aware of the release
5. **Monitor after release** - Watch for issues in production

## Troubleshooting

### Common Issues

1. **Version mismatch between files**
   - Use the update script to ensure consistency
   - Double-check all files manually

2. **Docker build fails**
   - Ensure all dependencies are available
   - Check Dockerfile syntax

3. **Helm chart validation fails**
   - Run `helm lint charts/helm-operator`
   - Validate with `helm template charts/helm-operator`

4. **Git tag already exists**
   - Delete the existing tag: `git tag -d v0.3.0`
   - Push deletion: `git push origin :refs/tags/v0.3.0`
   - Create new tag with correct information
