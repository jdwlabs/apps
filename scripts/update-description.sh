#!/usr/bin/env bash

# Script: update-description.sh
# Description: Sync a DockerHub repo's Overview from a source-controlled README so the
#              Hub page never drifts from the repo. Runs as an nx `version` postTarget,
#              so it only fires for affected/released apps — matching the rest of the
#              pipeline. Chosen over peter-evans/dockerhub-description because a script
#              slots into the existing postTarget chain without a separate CI matrix.
# Usage: ./update-description.sh <repo> <readmePath>
#   repo        e.g. jdwlabs/container
#   readmePath  path to that image's README.docker.md
# Auth: DOCKERHUB_USERNAME / DOCKERHUB_PASSWORD (the creds CI already uses to push
#       images). If org 2FA later blocks password login for the API, swap in a scoped
#       PAT via the same env vars — no script change needed.

set -euo pipefail

if [ "${#}" -ne 2 ]; then
  echo "Usage: ${0} <repo> <readmePath>" >&2
  exit 1
fi

repo="${1}"
readme="${2}"

: "${DOCKERHUB_USERNAME:?DOCKERHUB_USERNAME must be set}"
: "${DOCKERHUB_PASSWORD:?DOCKERHUB_PASSWORD must be set}"

# A missing README is a hard error: silently skipping would let the Hub page rot,
# which is the exact drift this script exists to prevent.
if [ ! -f "${readme}" ]; then
  echo "README not found: ${readme}" >&2
  exit 1
fi

# Hub's short description caps at 100 chars. Derive it from the first line that is
# neither blank nor a heading — i.e. the tagline under the title, not the "# name"
# H1 — and strip markdown bold so it reads cleanly. Keeps it in sync with the README.
short="$(grep -m1 -vE '^[[:space:]]*(#|$)' "${readme}" | sed -E 's/\*\*//g; s/^#+ *//' | cut -c1-100)"

token="$(curl -sf -H 'Content-Type: application/json' \
  -d "{\"username\": \"${DOCKERHUB_USERNAME}\", \"password\": \"${DOCKERHUB_PASSWORD}\"}" \
  https://hub.docker.com/v2/users/login/ | jq -r '.token')"

if [ -z "${token}" ] || [ "${token}" = "null" ]; then
  echo "DockerHub login failed for ${repo}" >&2
  exit 1
fi

# jq --rawfile slurps the README into a JSON string, handling quotes/newlines safely
# where naive interpolation would produce invalid JSON.
body="$(jq -n --rawfile full "${readme}" --arg short "${short}" \
  '{full_description: $full, description: $short}')"

resp="$(mktemp)"
trap 'rm -f "${resp}"' EXIT

# Fail loudly on any non-2xx so a broken sync surfaces in CI instead of passing silently.
http_code="$(curl -s -o "${resp}" -w '%{http_code}' -X PATCH \
  -H "Authorization: JWT ${token}" \
  -H 'Content-Type: application/json' \
  -d "${body}" \
  "https://hub.docker.com/v2/repositories/${repo}/")"

if [ "${http_code}" -lt 200 ] || [ "${http_code}" -ge 300 ]; then
  echo "DockerHub description update failed for ${repo} (HTTP ${http_code}):" >&2
  cat "${resp}" >&2
  exit 1
fi

echo "[${repo}]: Overview synced from ${readme}"
