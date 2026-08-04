#!/usr/bin/env bash

# Script: build-image.sh
# Description: Build a jdwlabs image with the full org.opencontainers.image.* label
#              set and hand off to `docker buildx`. Centralising the ten OCI labels
#              here means a label change is one edit instead of the same flags
#              duplicated across every app's project.json (where they would drift).
# Usage: ./build-image.sh <image> <version> <dockerfile> <description> [buildx args...]
#   image        image name under jdwlabs/ (e.g. container)
#   version      semver tag; also the image.version label
#   dockerfile   path to the Dockerfile
#   description  one-line image.description label (quote if it has spaces)
#   buildx args  passed straight through to buildx — caller owns --platform and
#                --push/--load (release pushes multi-arch; local uses --load)

set -euo pipefail

if [ "${#}" -lt 4 ]; then
  echo "Usage: ${0} <image> <version> <dockerfile> <description> [buildx args...]" >&2
  exit 1
fi

image="${1}"
version="${2}"
dockerfile="${3}"
description="${4}"
shift 4

# The version arrives through command substitution, and a substitution that fails
# still leaves a well-formed call with an empty argument. Without this the build
# publishes jdwlabs/<image>: with an empty tag rather than stopping.
if [ -z "${version//[[:space:]]/}" ]; then
  echo "${0}: refusing to build ${image} with an empty version" >&2
  exit 1
fi

# revision/created are resolved here rather than passed in so every call site gets
# them for free. A missing git context is non-fatal: an image is still worth building
# without a traceable SHA, so fall back rather than abort.
revision="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"
created="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

# licenses is the SPDX id for PolyForm Noncommercial — the org's actual license.
docker buildx build \
  -f "${dockerfile}" \
  -t "jdwlabs/${image}:latest" \
  -t "jdwlabs/${image}:${version}" \
  --label "org.opencontainers.image.title=${image}" \
  --label "org.opencontainers.image.description=${description}" \
  --label "org.opencontainers.image.vendor=jdwlabs" \
  --label "org.opencontainers.image.licenses=PolyForm-Noncommercial-1.0.0" \
  --label "org.opencontainers.image.url=https://github.com/jdwlabs/apps" \
  --label "org.opencontainers.image.source=https://github.com/jdwlabs/apps" \
  --label "org.opencontainers.image.documentation=https://hub.docker.com/r/jdwlabs/${image}" \
  --label "org.opencontainers.image.version=${version}" \
  --label "org.opencontainers.image.revision=${revision}" \
  --label "org.opencontainers.image.created=${created}" \
  "${@}" \
  .

echo "[${image}]: built ${image}:${version} (revision ${revision})"
