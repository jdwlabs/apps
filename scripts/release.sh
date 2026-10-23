#!/bin/bash

# Script: release.sh
# Description: Convenience wrapper around Nx Release. Versioning, changelogs,
#              tagging and the release commit are all driven by the "release"
#              configuration in nx.json.
# Usage: ./release.sh [nx release options...]
#   e.g. ./release.sh --dry-run
#        ./release.sh --skip-publish --yes

set -euo pipefail

pnpm exec nx release "$@"
