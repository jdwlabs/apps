#!/bin/bash

set -euo pipefail

# === CONFIG ===
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"  # Get the script's directory
DOCKERHUB_IMAGE="jdwillmsen/jdw-agents"
GHCR_IMAGE="ghcr.io/jdwillmsen/jdw-agents"
VERSION_FILE="$SCRIPT_DIR/VERSION"
README_FILE="$SCRIPT_DIR/README.md"

bump_version() {
  local current="$1"
  local part="$2"
  IFS='.' read -r major minor patch <<< "$current"

  case "$part" in
    major)
      echo "$((major + 1)).0.0"
      ;;
    minor)
      echo "$major.$((minor + 1)).0"
      ;;
    patch)
      echo "$major.$minor.$((patch + 1))"
      ;;
    *)
      echo "Invalid bump type: $part" >&2
      exit 1
      ;;
  esac
}

# === DETERMINE VERSION ===
if [[ -f "$VERSION_FILE" ]]; then
  CURRENT_VERSION=$(cat "$VERSION_FILE")
else
  CURRENT_VERSION="0.0.0"
fi

if [[ $# -ge 1 ]]; then
  if [[ "$1" =~ ^(major|minor|patch)$ ]]; then
    VERSION=$(bump_version "$CURRENT_VERSION" "$1")
    echo "$VERSION" > "$VERSION_FILE"
  else
    VERSION="$1"
    echo "$VERSION" > "$VERSION_FILE"
  fi
else
  VERSION="$CURRENT_VERSION"
fi

# === UPDATE README.md VERSION BADGE ===
sed -i.bak -E "s|version-[0-9]+\.[0-9]+\.[0-9]+-blue|version-${VERSION}-blue|g" "$README_FILE" && rm "$README_FILE.bak"

# === BUILD METADATA ===
BUILD_DATE="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
GIT_COMMIT="$(git rev-parse --short HEAD)"

echo "🚀 Building version: $VERSION"
echo "📦 Tags:"
echo "  - $DOCKERHUB_IMAGE:$VERSION"
echo "  - $GHCR_IMAGE:$VERSION"

# === BUILD AND PUSH USING BUILDX ===
docker buildx build . \
  -f "$SCRIPT_DIR/Dockerfile" \
  --platform linux/amd64 \
  --label org.opencontainers.image.source="https://github.com/jdwillmsen/jdw" \
  --label org.opencontainers.image.version="$VERSION" \
  --label org.opencontainers.image.revision="$GIT_COMMIT" \
  --label org.opencontainers.image.created="$BUILD_DATE" \
  --label org.opencontainers.image.description="A custom Docker image based on ghcr.io/actions/actions-runner with additional preinstalled tools for build and deployment tasks." \
  -t "$DOCKERHUB_IMAGE:$VERSION" \
  -t "$DOCKERHUB_IMAGE:latest" \
  -t "$GHCR_IMAGE:$VERSION" \
  -t "$GHCR_IMAGE:latest" \
  --push

# === AUTOMATIC COMMIT AND PUSH ===
git add "$VERSION_FILE" "$README_FILE"
git commit -m "chore(tools): update agents to version ${VERSION}"
git push origin

echo "✅ Published $VERSION successfully!"
