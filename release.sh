#!/bin/bash
# Release script for AutoEngineer
# Usage: ./release.sh [major|minor|patch]

set -e

# Default to patch if no argument provided
BUMP_TYPE="${1:-patch}"

# Determine base branch (falls back to master)
BASE_BRANCH=$(git symbolic-ref --quiet --short refs/remotes/origin/HEAD 2>/dev/null | sed 's@^origin/@@')
BASE_BRANCH=${BASE_BRANCH:-master}

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
echo "📌 Base branch: $BASE_BRANCH"
echo ""

# Prepare release branch
git fetch origin "$BASE_BRANCH"
git checkout "$BASE_BRANCH"
git pull --ff-only origin "$BASE_BRANCH"

RELEASE_BRANCH="release/v$NEW_VERSION"
git checkout -B "$RELEASE_BRANCH" "origin/$BASE_BRANCH"
echo "✅ Created release branch: $RELEASE_BRANCH"

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
git push origin "v$NEW_VERSION"
echo "✅ Pushed tag to remote"

# Push release branch
git push -u origin "$RELEASE_BRANCH"
echo "✅ Pushed branch to remote"

# Open a pull request if gh is available; otherwise print the URL
ORIGIN_URL=$(git config --get remote.origin.url)
REPO_SLUG=$(echo "$ORIGIN_URL" | sed -E 's#(git@github.com:|https?://github.com/)([^/]+/[^/.]+)(\\.git)?#\\2#')
PR_TITLE="Release v$NEW_VERSION"
PR_URL="https://github.com/$REPO_SLUG/compare/$BASE_BRANCH...$RELEASE_BRANCH?expand=1"

if command -v gh >/dev/null 2>&1; then
    if gh auth status >/dev/null 2>&1; then
        gh pr create --title "$PR_TITLE" --body "Automated release for v$NEW_VERSION" --base "$BASE_BRANCH" --head "$RELEASE_BRANCH" || {
            echo "⚠️  gh pr create failed; open manually: $PR_URL"
        }
    else
        echo "⚠️  gh installed but not authenticated; open PR manually: $PR_URL"
    fi
else
    echo "ℹ️  Open PR: $PR_URL"
fi

echo ""
echo "🎉 Released v$NEW_VERSION!"
echo "   Branch: $RELEASE_BRANCH"
echo "   PR: $PR_URL"
