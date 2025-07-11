# Changelog Generation Examples

This document shows how to use the new changelog generation tools.

## Available Commands

### 1. Generate Changelog (Detailed)

```bash
# Generate detailed changelog from latest tag to HEAD
make generate-changelog

# Generate changelog between specific tags
make generate-changelog FROM=v0.1.0 TO=v0.2.0

# Generate changelog from previous tag to specific tag
make generate-changelog TO=v0.2.0
```

### 2. Generate Release Notes (GitHub Format)

```bash
# Generate GitHub-style release notes
make generate-release-notes

# Generate release notes between specific tags
make generate-release-notes FROM=v0.1.0 TO=v0.2.0

# Generate release notes and save to file
make release-notes-file VERSION=0.3.0
```

## Example Output

### Detailed Changelog Format

```markdown
# Changelog

## [v0.2.0] - 2025-01-11

### 💥 Breaking Changes

- ♻️ **Refactor**: rename RawValues to OriginalValues and implement proper chart values extraction ([db67880](../../commit/db67880))

### ✨ Features

- ✨ **Feature**: add ConfigMap creation for chart values during HelmRepository sync ([31e3174](../../commit/31e3174))
- ✨ **Feature**: use dynamic namespace configuration for Helm operations ([db600eb](../../commit/db600eb))

### 🐛 Bug Fixes

- 🐛 **Fix**: resolve finalizer removal conflict in HelmRepository deletion ([db67880](../../commit/db67880))
- 🐛 **Fix**: namespace deployment issues in HelmRelease ([db600eb](../../commit/db600eb))

### 🔧 Other Changes

- 🔧 **Chore**: bump version to 0.2.0 ([abc1234](../../commit/abc1234))
- 📚 **Documentation**: add comprehensive release process documentation ([def5678](../../commit/def5678))

### 👥 Contributors

- John Doe
- Jane Smith

**Full Changelog**: https://github.com/ketches/helm-operator/compare/v0.1.0...v0.2.0
```

### GitHub Release Notes Format

```markdown
## What's Changed

### ✨ New Features

- add ConfigMap creation for chart values during HelmRepository sync ([31e3174](../../commit/31e3174))
- use dynamic namespace configuration for Helm operations ([db600eb](../../commit/db600eb))

### 🐛 Bug Fixes

- resolve finalizer removal conflict in HelmRepository deletion ([db67880](../../commit/db67880))
- fix namespace deployment issues in HelmRelease ([db600eb](../../commit/db600eb))

### 🔧 Other Changes

- rename RawValues to OriginalValues and implement proper chart values extraction ([31e3174](../../commit/31e3174))
- bump version to 0.2.0 ([abc1234](../../commit/abc1234))

### 👥 New Contributors

- @johndoe made their first contribution
- @janesmith made their first contribution

**Full Changelog**: https://github.com/ketches/helm-operator/compare/v0.1.0...v0.2.0
```

## Integration with Release Process

### Automated Release with Changelog

```bash
# Complete release with automatic changelog generation
make release-complete VERSION=0.3.0
```

This will:
1. Update version numbers
2. Run tests and checks
3. Commit changes
4. Generate release notes file (`release-notes-v0.3.0.md`)
5. Create and push git tag
6. Package Helm chart

### Manual Release with Custom Changelog

```bash
# 1. Prepare release
make release-prepare VERSION=0.3.0

# 2. Generate and review release notes
make release-notes-file VERSION=0.3.0
cat release-notes-v0.3.0.md

# 3. Edit release notes if needed
vim release-notes-v0.3.0.md

# 4. Commit and tag
git add .
git commit -m "chore: bump version to 0.3.0"
make release-tag VERSION=0.3.0 MESSAGE="$(cat release-notes-v0.3.0.md)"

# 5. Package chart
make helm-package
```

## Conventional Commit Support

The changelog generator recognizes conventional commit formats:

- `feat:` or `feature:` → ✨ New Features
- `fix:` → 🐛 Bug Fixes
- `docs:` → 📚 Documentation
- `style:` → 💄 Style
- `refactor:` → ♻️ Refactor
- `perf:` → ⚡ Performance
- `test:` → 🧪 Test
- `build:` → 🔨 Build
- `ci:` → 👷 CI
- `chore:` → 🔧 Chore

### Breaking Changes Detection

The tool automatically detects breaking changes:
- Commits with `BREAKING CHANGE:` in the message
- Commits with `!` after the type (e.g., `feat!:`)

## Tips for Better Changelogs

### 1. Use Conventional Commits

```bash
# Good commit messages
git commit -m "feat: add chart values ConfigMap creation"
git commit -m "fix: resolve namespace deployment issue"
git commit -m "docs: update release process documentation"

# Breaking change
git commit -m "feat!: rename RawValues to OriginalValues"
```

### 2. Write Descriptive Commit Messages

```bash
# Instead of:
git commit -m "fix bug"

# Use:
git commit -m "fix: resolve finalizer removal conflict in HelmRepository deletion"
```

### 3. Group Related Changes

```bash
# Make logical commits that group related changes
git commit -m "feat: add ConfigMap creation for chart values

- Create ConfigMaps during HelmRepository sync
- Add OwnerReference for automatic cleanup
- Include comprehensive labels and annotations"
```

## Customization

### Modify Changelog Format

Edit `scripts/generate-changelog.sh` to customize:
- Commit categorization
- Output format
- Emoji usage
- Link formats

### Modify Release Notes Format

Edit `scripts/generate-release-notes.sh` to customize:
- GitHub-specific formatting
- Contributor detection
- Section organization

## Troubleshooting

### No Tags Found

```bash
# If you get "No tags found", create your first tag
git tag -a v0.1.0 -m "Initial release"
git push origin v0.1.0
```

### Empty Changelog

```bash
# If changelog is empty, check if commits exist between tags
git log v0.1.0..v0.2.0 --oneline
```

### Wrong Tag Range

```bash
# List all tags to verify correct range
git tag -l --sort=-version:refname
```