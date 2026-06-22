#!/bin/bash

# Script: update-app.sh
# Description: Bumps a Helm chart's appVersion in the deployments repo by opening
#              a pull request (never a direct push to the default branch).
# Usage: update-app.sh <chart_file_path> <version_number> <project_name>
#
# Auth (one of):
#   GH_APP_ID + GH_APP_PRIVATE_KEY  GitHub App credentials. A fresh installation
#                      token scoped to the target repo (contents:write +
#                      pull_requests:write) is minted at push time — preferred in
#                      CI, never expires mid-run.
#   DEPLOYMENTS_TOKEN  A pre-minted token with write + PR access (escape hatch for
#                      local/manual runs). Takes precedence if set.
# Optional env:
#   DEPLOYMENTS_REPO     Target repo slug.  Default: jdwlabs/deployments
#   DEPLOYMENTS_HOST     Git host.          Default: github.com
#   RELEASE_AUTO_MERGE   When "true", enable auto-merge on the opened PR so it
#                        merges once branch protection is satisfied. Default off:
#                        the PR waits for the required review. Flip this on only
#                        after the deployments rulesets/repo settings actually
#                        permit an unattended merge (auto-merge enabled, required
#                        approvals reconciled) — otherwise it is a no-op warning.

set -euo pipefail

deployments_repo="${DEPLOYMENTS_REPO:-jdwlabs/deployments}"
deployments_host="${DEPLOYMENTS_HOST:-github.com}"
api_url="${GITHUB_API_URL:-https://api.${deployments_host}}"
git_repository="https://${deployments_host}/${deployments_repo}.git"

base64url() { openssl base64 -A | tr '+/' '-_' | tr -d '='; }

# Mint a short-lived installation token scoped to ${deployments_repo} from App
# credentials. Done at push time so a long build (e.g. multi-arch build-image
# running before this postTarget) can't expire a token minted at job start.
# Scopes contents:write (push the branch) + pull_requests:write (open the PR).
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

if [ -n "${DEPLOYMENTS_TOKEN:-}" ]; then
  deployments_token="${DEPLOYMENTS_TOKEN}"
elif [ -n "${GH_APP_ID:-}" ] && [ -n "${GH_APP_PRIVATE_KEY:-}" ]; then
  deployments_token=$(mint_installation_token "${GH_APP_ID}" "${GH_APP_PRIVATE_KEY}" "${deployments_repo}")
else
  echo "Error: set GH_APP_ID + GH_APP_PRIVATE_KEY (preferred) or DEPLOYMENTS_TOKEN." >&2
  exit 1
fi

# Authenticate via an http extraheader instead of embedding the token in the
# remote URL. This keeps the token out of .git/config and out of git's stderr
# (which echoes the remote URL on failure). Mirrors actions/checkout.
git_auth_header="AUTHORIZATION: basic $(printf '%s' "x-access-token:${deployments_token}" | base64 | tr -d '\n')"

temp_dir=$(mktemp -d)

# Always remove the clone, even on an unexpected exit, so failed runs leave no temp dirs behind.
trap 'rm -rf "${temp_dir}"' EXIT

clone_repository() {
  # A failed clone leaves temp_dir empty; without this guard the later git checks
  # run in a non-repo and report the misleading "Not inside a Git repository".
  # Surface the real cause instead — usually the token lacking access.
  if ! git -c http.extraheader="${git_auth_header}" clone "${git_repository}" "${temp_dir}"; then
    echo "Error: Failed to clone ${deployments_repo}. Ensure the token has write access." >&2
    exit 1
  fi
  cd "${temp_dir}" || exit 1
}

clean_repository() {
  rm -rf "${temp_dir}"
}

# Function to check if the repository is a Git repository
check_git_repository() {
  if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    echo "Error: Not inside a Git repository."
    clean_repository
    exit 1
  fi
}

# Function to check if a file exists and is tracked by Git
check_file() {
  local file_path=${1}
  if [ ! -f "${file_path}" ]; then
    echo "Error: File ${file_path} does not exist."
    clean_repository
    exit 1
  fi

  if ! git ls-files --error-unmatch "${file_path}" >/dev/null 2>&1; then
    echo "Error: File ${file_path} is not tracked by Git."
    clean_repository
    exit 1
  fi
}

# Function to check if there are uncommitted changes in the repository
check_uncommitted_changes() {
  if ! git diff-index --quiet HEAD --; then
    echo "Error: There are uncommitted changes in the repository. Please commit or stash them first."
    clean_repository
    exit 1
  fi
}

# Open a pull request for the pushed branch. The default branch is protected
# (PR required + signed commits), so the bump lands via PR review/merge, never a
# direct push. The squash/merge commit is created by GitHub and is therefore
# verified, satisfying the required-signatures rule.
open_pull_request() {
  local branch=${1} base=${2} title=${3} body=${4}
  local payload resp pr_url pr_node

  payload=$(printf '{"title":"%s","head":"%s","base":"%s","body":"%s"}' \
    "${title}" "${branch}" "${base}" "${body}")

  if ! resp=$(curl -sf -X POST \
      -H "Authorization: token ${deployments_token}" \
      -H "Accept: application/vnd.github+json" \
      "${api_url}/repos/${deployments_repo}/pulls" -d "${payload}"); then
    echo "Error: Failed to open pull request on ${deployments_repo} (token needs pull_requests:write; branch may already have an open PR)." >&2
    clean_repository
    exit 1
  fi

  pr_url=$(printf '%s' "${resp}" | grep -o '"html_url"[[:space:]]*:[[:space:]]*"[^"]*"' | head -1 | sed 's/.*"\([^"]*\)"$/\1/')
  pr_node=$(printf '%s' "${resp}" | grep -o '"node_id"[[:space:]]*:[[:space:]]*"[^"]*"' | head -1 | sed 's/.*"\([^"]*\)"$/\1/')

  echo "Opened pull request: ${pr_url:-"(URL unavailable)"}"

  if [ "${RELEASE_AUTO_MERGE:-false}" = "true" ] && [ -n "${pr_node}" ]; then
    enable_auto_merge "${pr_node}" || \
      echo "Warning: could not enable auto-merge; PR will wait for a manual/required merge." >&2
  fi
}

# Best-effort: enable auto-merge via GraphQL. Requires the repo to allow
# auto-merge and the token to have pull_requests:write. A failure here is not
# fatal — the PR is open and can be merged once protection is satisfied.
enable_auto_merge() {
  local pr_node=${1} query
  query=$(printf '{"query":"mutation { enablePullRequestAutoMerge(input: {pullRequestId: \\"%s\\", mergeMethod: SQUASH}) { clientMutationId } }"}' "${pr_node}")
  curl -sf -X POST \
    -H "Authorization: token ${deployments_token}" \
    -H "Accept: application/vnd.github+json" \
    "${api_url}/graphql" -d "${query}" >/dev/null
}

# Function to update a file in a Git repository and open a PR for the change
update_file() {
  local file_path=${1}
  local new_version=${2}
  local project_name=${3}
  local base_branch branch title body

  clone_repository
  check_git_repository
  check_file "${file_path}"
  check_uncommitted_changes

  base_branch=$(git symbolic-ref --short HEAD)
  branch="chore/bump-${project_name}-appversion-${new_version}"

  git checkout HEAD "${file_path}"

  # Fail loudly if the line is absent: otherwise sed is a no-op, nothing is
  # staged, and the misleading failure surfaces later at the empty commit.
  if ! grep -q '^appVersion:' "${file_path}"; then
    echo "Error: no 'appVersion:' line found in ${file_path}." >&2
    clean_repository
    exit 1
  fi

  git switch -c "${branch}"

  # Replace the app version line in the file
  sed -i "s/^appVersion: .*/appVersion: \"${new_version}\"/" "${file_path}"

  title="chore(${project_name}): update appVersion to ${new_version}"
  body="Automated appVersion bump for ${project_name} to ${new_version}. Updates ${file_path}."

  git add "${file_path}"
  git commit -m "${title}"

  if ! git -c http.extraheader="${git_auth_header}" push -u origin "${branch}"; then
    echo "Error: Failed to push branch ${branch} to ${deployments_repo}." >&2
    clean_repository
    exit 1
  fi

  open_pull_request "${branch}" "${base_branch}" "${title}" "${body}"

  echo "[${project_name}]: appVersion bump to ${new_version} proposed via PR against ${base_branch}."
  clean_repository
}

# Check if correct number of arguments are provided
if [ "${#}" -lt 3 ]; then
  echo "Usage: ${0} <file_path> <version_number> <project_name>"
  clean_repository
  exit 1
fi

update_file "${1}" "${2}" "${3}"
