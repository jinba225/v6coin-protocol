#!/bin/bash
set -e

VERSION=$1

if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 v0.1.0"
    exit 1
fi

echo "Tagging release $VERSION..."

# Tag the commit
git tag -a "$VERSION" -m "Release $VERSION"

# Push tag to remote
git push origin "$VERSION"

echo "✓ Release $VERSION tagged and pushed"
echo "GitHub Actions will build and publish the release"
