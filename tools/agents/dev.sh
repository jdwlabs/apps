#!/bin/bash

set -euo pipefail

# Get the directory where this script is located
SCRIPT_DIR="$(dirname "$(realpath "$0")")"

IMAGE_NAME="jdwillmsen/jdw-agents"
CONTAINER_NAME="jdw-agents-dev"
VERSION_FILE="$SCRIPT_DIR/VERSION"

VERSION=$(cat "$VERSION_FILE" 2>/dev/null || echo "dev")
BUILD_DATE="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
GIT_COMMIT="$(git rev-parse --short HEAD)"

case "${1:-}" in
  build)
    echo "🔨 Building local image with buildx: $IMAGE_NAME:$VERSION"
    docker buildx build "$SCRIPT_DIR" \
      --platform linux/amd64 \
      --label org.opencontainers.image.source="https://github.com/jdwillmsen/jdw" \
      --label org.opencontainers.image.version="$VERSION" \
      --label org.opencontainers.image.revision="$GIT_COMMIT" \
      --label org.opencontainers.image.created="$BUILD_DATE" \
      --label org.opencontainers.image.description="A custom Docker image based on ghcr.io/actions/actions-runner with additional preinstalled tools for build and deployment tasks." \
      -t "$IMAGE_NAME:$VERSION" \
      --load  # Use --load to load the image locally
    ;;

  run)
    echo "🚀 Running container: $CONTAINER_NAME"
    docker run -it --rm \
      --name "$CONTAINER_NAME" \
      "$IMAGE_NAME:$VERSION" \
      bash
    ;;

  shell)
    echo "🔧 Shell into container: $CONTAINER_NAME"
    docker exec -it "$CONTAINER_NAME" bash
    ;;

  rebuild)
    "$0" build
    "$0" run
    ;;

  clean)
    echo "🧹 Cleaning up container and image"
    docker rm -f "$CONTAINER_NAME" 2>/dev/null || true
    docker rmi "$IMAGE_NAME:$VERSION" 2>/dev/null || true
    ;;

  help|"")
    echo "🛠️  Usage:"
    echo "  $(basename "$0") build     - Build image locally"
    echo "  $(basename "$0") run       - Run container interactively"
    echo "  $(basename "$0") rebuild   - Rebuild and run fresh"
    echo "  $(basename "$0") shell     - Exec into running container"
    echo "  $(basename "$0") clean     - Remove image and container"
    echo "  $(basename "$0") help      - Show this help"
    ;;

  *)
    echo "❌ Unknown command: $1"
    exit 1
    ;;
esac
