# Release Example

This document shows how to use the new release management tools to create a new version.

## Quick Release (Automated)

For a complete automated release:

```bash
# This will update versions, run tests, commit, tag, and package
make release-complete VERSION=1.0.0
```

## Step-by-Step Release

### 1. Prepare the Release

```bash
# Update versions and run all checks
make release-prepare VERSION=1.0.0
```

This will:

- Update `charts/helm-operator/Chart.yaml`
- Update `charts/helm-operator/values.yaml` (if applicable)
- Update version references in `README.md`
- Run `make manifests` to update CRDs
- Run `make test` to ensure everything works
- Run `make lint` to check code quality

### 2. Review Changes

```bash
git diff
```

### 3. Commit and Tag

```bash
# Commit the version changes
git add .
git commit -m "chore: bump version to 1.0.0"

# Create and push the tag
make release-tag VERSION=1.0.0 MESSAGE="Release v1.0.0

Features:
- New feature 1
- New feature 2

Bug Fixes:
- Fixed issue 1
- Fixed issue 2"
```

### 4. Build and Push Docker Image

```bash
# Build the Docker image
make docker-build IMG=ketches/helm-operator:1.0.0

# Push to registry
make docker-push IMG=ketches/helm-operator:1.0.0
```

### 5. Package Helm Chart

```bash
# Package the Helm chart
make helm-package
```

This creates `helm-operator-1.0.0.tgz` in the current directory.

## Manual Version Update Only

If you just want to update version numbers without running tests:

```bash
# Update version numbers only
make update-version VERSION=1.0.0
```

Or use the script directly:

```bash
./scripts/update-version.sh 1.0.0
```

## Files Updated by Version Scripts

The version update process modifies these files:

1. **charts/helm-operator/Chart.yaml**

   ```yaml
   version: 1.0.0
   appVersion: "1.0.0"
   ```

2. **charts/helm-operator/values.yaml** (if it contains image tags)

   ```yaml
   image:
     tag: "1.0.0"
   ```

3. **README.md** (version references in examples)

   ```markdown
   docker pull ketches/helm-operator:1.0.0
   ```

## Verification

After release, verify everything is correct:

```bash
# Check the tag was created
git tag -l | grep v1.0.0

# Check Chart.yaml
grep -E "^(version|appVersion):" charts/helm-operator/Chart.yaml

# Check if Docker image exists
docker pull ketches/helm-operator:1.0.0

# Verify Helm chart
helm lint charts/helm-operator
```

## Rollback

If you need to rollback:

```bash
# Delete the tag locally and remotely
git tag -d v1.0.0
git push origin :refs/tags/v1.0.0

# Revert version changes
git revert HEAD
```

## Best Practices

1. **Always test before releasing**
2. **Use semantic versioning**
3. **Write meaningful release notes**
4. **Coordinate with the team**
5. **Monitor after release**
