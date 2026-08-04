#!/usr/bin/env bash

# Script: prepare-version.sh
# Description: Stamp the released version into an Angular app's public/VERSION so
#              the built image serves it at /VERSION. The tracked file holds a
#              development placeholder: versions live in git tags, and a tag is
#              only known once the release has already happened, which is after
#              the last commit anyone could write it into.
# Usage: ./prepare-version.sh <version_file> <version>
#
# Pairs with restore-version.sh, which puts the placeholder back so a build never
# leaves the working tree dirty.

set -euo pipefail

if [ "${#}" -ne 2 ]; then
  echo "Usage: ${0} <version_file> <version>" >&2
  exit 1
fi

version_file="${1}"
version="${2}"
backup_file="${version_file}.bak"

if [ ! -f "${backup_file}" ]; then
  cp "${version_file}" "${backup_file}"
fi

printf '%s\n' "${version}" > "${version_file}"
echo "Stamped ${version_file} with ${version}"
