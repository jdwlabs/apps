#!/usr/bin/env bash

# Script: resolve-version.sh
# Description: Print the released semver for a project. The git tag is the single
#              source of truth for a version — no manifest in the working tree
#              records it, so nothing has to be committed for a release to happen.
# Usage: ./resolve-version.sh <project>
#   project   Nx project name, matching the `{projectName}-{version}` tag pattern
#
# Resolution order:
#   1. $RELEASE_VERSION — the version the release job read off the pushed tag and
#      passed down through its deliver matrix. Preferred in CI because the deliver
#      job checks out a single commit without tags.
#   2. A tag pointing at HEAD. Exact by construction: the release tags the commit
#      it is delivering.
#   3. The newest reachable matching tag. Covers a local build from a working
#      tree that is ahead of the last release.
#
# Exits non-zero rather than emitting a placeholder: an image or chart bump
# carrying a guessed version is worse than a build that stops.

set -euo pipefail

if [ "${#}" -ne 1 ]; then
  echo "Usage: ${0} <project>" >&2
  exit 1
fi

project="${1}"
semver='[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?'

if [ -n "${RELEASE_VERSION:-}" ]; then
  printf '%s\n' "${RELEASE_VERSION}"
  exit 0
fi

extract() {
  grep -E "^${project}-${semver}$" | sed -E "s/^${project}-//" | head -1
}

version="$(git tag --points-at HEAD 2>/dev/null | extract || true)"

if [ -z "${version}" ]; then
  version="$(git tag --list "${project}-*" --sort=-v:refname 2>/dev/null | extract || true)"
fi

if [ -z "${version}" ]; then
  echo "${0}: no release tag matching '${project}-<semver>' and RELEASE_VERSION is unset" >&2
  exit 1
fi

printf '%s\n' "${version}"
