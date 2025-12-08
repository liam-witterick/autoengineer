#!/bin/bash
# Release script for AutoEngineer
# Usage: ./release.sh [major|minor|patch]

set -e

# Default to patch if no argument provided
BUMP_TYPE="${1:-patch}"

# Validate bump type
if [[ ! "$BUMP_TYPE" =~ ^(major|minor|patch)$ ]]; then
    echo "❌ Invalid bump type: $BUMP_TYPE"
    echo "Usage: $0 [major|minor|patch]"
    exit 1
fi

# Get current version from install.sh
CURRENT_VERSION=$(grep -oP 'VERSION="\K[0-9]+\.[0-9]+\.[0-9]+' install.sh)

if [ -z "$CURRENT_VERSION" ]; then
    echo "❌ Could not find current version in install.sh"
    exit 1
fi

echo "📦 Current version: v$CURRENT_VERSION"

# Parse version components
IFS='.' read -r MAJOR MINOR PATCH <<< "$CURRENT_VERSION"

# Bump version based on type
case "$BUMP_TYPE" in
    major)
        MAJOR=$((MAJOR + 1))
        MINOR=0
        PATCH=0
        ;;
    minor)
        MINOR=$((MINOR + 1))
        PATCH=0
        ;;
    patch)
        PATCH=$((PATCH + 1))
        ;;
esac

NEW_VERSION="$MAJOR.$MINOR.$PATCH"

echo "🚀 Bumping to: v$NEW_VERSION"
echo ""

# Update version in install.sh
sed -i "s/VERSION=\"$CURRENT_VERSION\"/VERSION=\"$NEW_VERSION\"/" install.sh
echo "✅ Updated install.sh"

# Update version in go/cmd/autoengineer/main.go
sed -i "s/version = \"$CURRENT_VERSION\"/version = \"$NEW_VERSION\"/" go/cmd/autoengineer/main.go
echo "✅ Updated go/cmd/autoengineer/main.go"

# Update version in Makefile
sed -i "s/VERSION=$CURRENT_VERSION/VERSION=$NEW_VERSION/" Makefile
echo "✅ Updated Makefile"

# Commit the version bump
git add install.sh go/cmd/autoengineer/main.go Makefile
git commit -m "chore: bump version to v$NEW_VERSION"
echo "✅ Committed version bump"

# Create and push tag
git tag -a "v$NEW_VERSION" -m "Release v$NEW_VERSION"
echo "✅ Created tag v$NEW_VERSION"

# Push commit and tag
git push
git push origin "v$NEW_VERSION"
echo "✅ Pushed to remote"

echo ""
echo "🎉 Released v$NEW_VERSION!"
echo "   View release: https://github.com/liam-witterick/autoengineer/releases/tag/v$NEW_VERSION"
