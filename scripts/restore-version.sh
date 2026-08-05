#!/usr/bin/env bash

# Script: restore-version.sh
# Description: Put the development placeholder back into an Angular app's
#              public/VERSION after prepare-version.sh stamped a release version
#              into it for an image build.
# Usage: ./restore-version.sh <version_file>

set -euo pipefail

if [ "${#}" -ne 1 ]; then
  echo "Usage: ${0} <version_file>" >&2
  exit 1
fi

version_file="${1}"
# Must mirror prepare-version.sh, which parks the backup one level above the
# asset root so it never ships inside the image.
backup_file="$(cd "$(dirname "${version_file}")/.." && pwd)/$(basename "${version_file}").bak"

if [ -f "${backup_file}" ]; then
  mv "${backup_file}" "${version_file}"
  echo "Restored ${version_file}"
else
  echo "No backup found for ${version_file}" >&2
  exit 1
fi
