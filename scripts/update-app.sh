#!/bin/bash

# Script: update-app.sh
# Description: Opens an auto-merging pull request that bumps a Helm chart's appVersion
#              in the deployments repo.
# Usage: update-app.sh <chart_file_path> <version_number> <project_name>
#
# Auth (one of):
#   GH_APP_ID + GH_APP_PRIVATE_KEY  GitHub App credentials. A fresh installation
#                      token scoped to the target repo (contents:write,
#                      pull_requests:write) is minted at push time — preferred in
#                      CI, never expires mid-run.
#   DEPLOYMENTS_TOKEN  A pre-minted token with write access (escape hatch for
#                      local/manual runs). Takes precedence if set.
# Optional env:
#   DEPLOYMENTS_REPO   Target repo slug.  Default: jdwlabs/deployments
#   DEPLOYMENTS_HOST   Git host.          Default: github.com
#
# Requires gh and jq. Every mutation goes through the API rather than a clone:
# the commit is then minted server-side under the App's own identity, and the
# branch/PR path leaves an audit trail that a direct push does not.

set -euo pipefail

deployments_repo="${DEPLOYMENTS_REPO:-jdwlabs/deployments}"
deployments_host="${DEPLOYMENTS_HOST:-github.com}"
api_url="${GITHUB_API_URL:-https://api.${deployments_host}}"

base64url() { openssl base64 -A | tr '+/' '-_' | tr -d '='; }

# Mint a short-lived installation token scoped to ${deployments_repo} from App
# credentials. Done at push time so a long build (e.g. multi-arch build-image
# running before this postTarget) can't expire a token minted at job start.
mint_installation_token() {
  local app_id="${1}" private_key="${2}" repo="${3}"
  local now iat exp header payload jwt inst_id repo_name token

  now=$(date +%s)
  iat=$((now - 60))   # backdate to tolerate clock skew
  exp=$((now + 540))  # 9 min; max allowed by GitHub is 10
  header='{"alg":"RS256","typ":"JWT"}'
  payload="{\"iat\":${iat},\"exp\":${exp},\"iss\":\"${app_id}\"}"
  jwt="$(printf '%s' "${header}" | base64url).$(printf '%s' "${payload}" | base64url)"
  jwt="${jwt}.$(printf '%s' "${jwt}" | openssl dgst -sha256 -sign <(printf '%s' "${private_key}") -binary | base64url)"

  inst_id=$(curl -sf -H "Authorization: Bearer ${jwt}" -H "Accept: application/vnd.github+json" \
    "${api_url}/repos/${repo}/installation" | grep -o '"id"[[:space:]]*:[[:space:]]*[0-9]*' | head -1 | grep -o '[0-9]*$')
  [ -n "${inst_id}" ] || { echo "Error: no App installation found for ${repo} (is the App installed?)." >&2; return 1; }

  repo_name="${repo#*/}"
  token=$(curl -sf -X POST -H "Authorization: Bearer ${jwt}" -H "Accept: application/vnd.github+json" \
    "${api_url}/app/installations/${inst_id}/access_tokens" \
    -d "{\"repositories\":[\"${repo_name}\"],\"permissions\":{\"contents\":\"write\",\"pull_requests\":\"write\"}}" \
    | grep -o '"token"[[:space:]]*:[[:space:]]*"[^"]*"' | head -1 | sed 's/.*"\([^"]*\)"$/\1/')
  [ -n "${token}" ] || { echo "Error: failed to mint installation token for ${repo}." >&2; return 1; }
  printf '%s' "${token}"
}

require_tools() {
  local tool
  for tool in gh jq; do
    command -v "${tool}" >/dev/null 2>&1 || {
      echo "Error: ${tool} is required but was not found on PATH." >&2
      exit 1
    }
  done
}

# Point the branch at the current main tip, creating it if absent. Rebuilding
# rather than appending keeps a rerun idempotent and the diff a single commit.
reset_branch_to_main() {
  local branch="${1}" main_sha="${2}"

  if gh api "repos/${deployments_repo}/git/ref/heads/${branch}" >/dev/null 2>&1; then
    gh api -X PATCH "repos/${deployments_repo}/git/refs/heads/${branch}" \
      -f sha="${main_sha}" -F force=true >/dev/null
  else
    gh api -X POST "repos/${deployments_repo}/git/refs" \
      -f ref="refs/heads/${branch}" -f sha="${main_sha}" >/dev/null
  fi
}

update_file() {
  local file_path="${1}" new_version="${2}" project_name="${3}"
  local main_sha branch payload current file_sha before after changed removed open_prs

  main_sha=$(gh api "repos/${deployments_repo}/git/ref/heads/main" --jq .object.sha)
  branch="chore/${project_name}-appversion-${new_version}"

  payload=$(gh api "repos/${deployments_repo}/contents/${file_path}?ref=${main_sha}")
  # jq builds that open stdout in text mode terminate lines with CRLF; the stray
  # CR corrupts the blob sha and makes base64 reject the content outright.
  file_sha=$(printf '%s' "${payload}" | jq -r .sha | tr -d '\r')
  before=$(mktemp); after=$(mktemp)
  trap 'rm -f "${before}" "${after}"' RETURN
  printf '%s' "${payload}" | jq -r .content | tr -d '\r' | base64 -d > "${before}"

  # Fail loudly if the line is absent: otherwise sed is a no-op and the
  # misleading failure surfaces later as an empty commit.
  if ! grep -q '^appVersion:' "${before}"; then
    echo "Error: no 'appVersion:' line found in ${file_path}." >&2
    exit 1
  fi

  current=$(sed -n 's/^appVersion:[[:space:]]*"\{0,1\}\([^"]*\)"\{0,1\}[[:space:]]*$/\1/p' "${before}" | head -1)
  if [ "${current}" = "${new_version}" ]; then
    echo "[${project_name}]: ${file_path} already pins ${new_version}; nothing to do."
    return 0
  fi

  sed "s/^appVersion: .*/appVersion: \"${new_version}\"/" "${before}" > "${after}"

  # Guard the blast radius of the sed: anything other than one appVersion line
  # means the chart's shape changed and this script should not be writing to it.
  removed=$(diff "${before}" "${after}" | grep '^<' || true)
  changed=$(printf '%s\n' "${removed}" | grep -c '^<' || true)
  if [ "${changed}" -ne 1 ] || ! printf '%s\n' "${removed}" | grep -q '^< appVersion:'; then
    echo "Error: expected exactly one changed line (appVersion) in ${file_path}, got ${changed}." >&2
    exit 1
  fi

  reset_branch_to_main "${branch}" "${main_sha}"

  # Commit through the contents API: the commit is created server-side, signed
  # by GitHub, and authored by the bot identity behind the token.
  gh api -X PUT "repos/${deployments_repo}/contents/${file_path}" \
    -f message="chore(${project_name}): update app version to version ${new_version}" \
    -f branch="${branch}" \
    -f sha="${file_sha}" \
    -f content="$(base64 -w0 "${after}")" >/dev/null

  open_prs=$(gh pr list -R "${deployments_repo}" --head "${branch}" --state open --json number --jq length)
  if [ "${open_prs}" -gt 0 ]; then
    echo "[${project_name}]: pull request already open for ${branch}; branch updated in place."
    return 0
  fi

  gh pr create -R "${deployments_repo}" --base main --head "${branch}" \
    --title "chore(${project_name}): update app version to version ${new_version}" \
    --body "Automated appVersion bump opened by the apps release pipeline.

| | |
|---|---|
| Chart | \`${file_path}\` |
| appVersion | \`${current}\` -> \`${new_version}\` |

This pull request changes **only** the \`appVersion\` line in \`${file_path}\`.
Merging it causes ArgoCD to sync the non environment to the new version."

  gh pr merge -R "${deployments_repo}" "${branch}" --auto --rebase --delete-branch

  echo "[${project_name}]: appVersion in ${file_path} set to ${new_version} via pull request on ${branch}."
}

require_tools

if [ "${#}" -lt 3 ]; then
  echo "Usage: ${0} <file_path> <version_number> <project_name>"
  exit 1
fi

if [ -n "${DEPLOYMENTS_TOKEN:-}" ]; then
  deployments_token="${DEPLOYMENTS_TOKEN}"
elif [ -n "${GH_APP_ID:-}" ] && [ -n "${GH_APP_PRIVATE_KEY:-}" ]; then
  deployments_token=$(mint_installation_token "${GH_APP_ID}" "${GH_APP_PRIVATE_KEY}" "${deployments_repo}")
else
  echo "Error: set GH_APP_ID + GH_APP_PRIVATE_KEY (preferred) or DEPLOYMENTS_TOKEN." >&2
  exit 1
fi

export GH_TOKEN="${deployments_token}"
export GH_HOST="${deployments_host}"

update_file "${1}" "${2}" "${3}"
