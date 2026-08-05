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
# The backup deliberately does not sit beside the file it backs up. public/ is
# the Angular asset root: the build copies everything under it into dist/, and
# the Dockerfile copies dist/ into the image web root, so a sibling backup would
# be served at /VERSION.bak carrying the placeholder. One level up is inside the
# project but outside every asset glob.
backup_file="$(cd "$(dirname "${version_file}")/.." && pwd)/$(basename "${version_file}").bak"

# Callers pass the version through command substitution, and a substitution that
# fails still leaves a well-formed call with an empty argument. Refusing it here
# is what stops a blank /VERSION shipping in an image.
if [ -z "${version//[[:space:]]/}" ]; then
  echo "${0}: refusing to stamp an empty version into ${version_file}" >&2
  exit 1
fi

if [ ! -f "${backup_file}" ]; then
  cp "${version_file}" "${backup_file}"
fi

printf '%s\n' "${version}" > "${version_file}"
echo "Stamped ${version_file} with ${version}"
