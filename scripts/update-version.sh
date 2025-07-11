#!/bin/bash

# Script to update version information across the project
# Usage: ./scripts/update-version.sh <new-version>
# Example: ./scripts/update-version.sh 1.0.0

set -e

if [ $# -eq 0 ]; then
    echo "Usage: $0 <new-version>"
    echo "Example: $0 1.0.0"
    exit 1
fi

NEW_VERSION=$1

# Validate version format (basic semver check)
if ! [[ $NEW_VERSION =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.-]+)?$ ]]; then
    echo "Error: Version must follow semantic versioning format (e.g., 1.2.3 or 1.2.3-alpha.1)"
    exit 1
fi

echo "Updating version to $NEW_VERSION..."

# Update Chart.yaml
echo "Updating charts/helm-operator/Chart.yaml..."
sed -i.bak "s/^version: .*/version: $NEW_VERSION/" charts/helm-operator/Chart.yaml
sed -i.bak "s/^appVersion: .*/appVersion: \"$NEW_VERSION\"/" charts/helm-operator/Chart.yaml

# Clean up backup files
rm -f charts/helm-operator/Chart.yaml.bak

# Update any version references in documentation
if [ -f "README.md" ]; then
    echo "Updating version references in README.md..."
    # Update image tags in README if they exist
    sed -i.bak "s/helm-operator:[0-9]\+\.[0-9]\+\.[0-9]\+/helm-operator:$NEW_VERSION/g" README.md
    rm -f README.md.bak
fi

# Update values.yaml if it contains version references
if [ -f "charts/helm-operator/values.yaml" ]; then
    echo "Checking charts/helm-operator/values.yaml for version references..."
    if grep -q "tag:" charts/helm-operator/values.yaml; then
        sed -i.bak "s/tag: .*/tag: \"$NEW_VERSION\"/" charts/helm-operator/values.yaml
        rm -f charts/helm-operator/values.yaml.bak
        echo "Updated image tag in values.yaml"
    fi
fi

echo "Version update completed!"
echo ""
echo "Files updated:"
echo "- charts/helm-operator/Chart.yaml"
echo "- charts/helm-operator/values.yaml (if applicable)"
echo "- README.md (if applicable)"
echo ""
echo "Next steps:"
echo "1. Review the changes: git diff"
echo "2. Commit the changes: git add . && git commit -m 'chore: bump version to $NEW_VERSION'"
echo "3. Create and push tag: git tag -a v$NEW_VERSION -m 'Release v$NEW_VERSION' && git push origin v$NEW_VERSION"
echo "4. Build and push Docker image with new tag"