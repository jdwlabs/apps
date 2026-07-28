#!/bin/bash
# Offline tests for update-app.sh. No network, no token, no deployments clone.
set -uo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
target="${script_dir}/../update-app.sh"
failures=0

setup() {
  FAKE_GH_DIR=$(mktemp -d)
  export FAKE_GH_DIR
  printf '1.0.0' > "${FAKE_GH_DIR}/appversion"
  : > "${FAKE_GH_DIR}/calls.log"
  stub_dir=$(mktemp -d)
  cp "${script_dir}/fake-gh" "${stub_dir}/gh"
  chmod +x "${stub_dir}/gh"
  export PATH="${stub_dir}:${PATH}"
  export DEPLOYMENTS_TOKEN="fake-token"
  unset FAKE_GH_FAIL || true
  rm -f "${FAKE_GH_DIR}/pr-open" "${FAKE_GH_DIR}/branch-exists"
}

check() {
  local label="$1" expected="$2"
  if grep -qF -- "${expected}" "${FAKE_GH_DIR}/calls.log"; then
    echo "PASS ${label}"
  else
    echo "FAIL ${label}: no call matching '${expected}'"
    echo "  calls were:"; sed 's/^/    /' "${FAKE_GH_DIR}/calls.log"
    failures=$((failures + 1))
  fi
}

check_status() {
  local label="$1" expected="$2" actual="$3"
  if [ "${expected}" = "${actual}" ]; then
    echo "PASS ${label}"
  else
    echo "FAIL ${label}: expected exit ${expected}, got ${actual}"
    failures=$((failures + 1))
  fi
}

echo "== opens a pull request for a new version =="
setup
bash "${target}" charts/demo/Chart.yaml 1.1.0 demo >/dev/null 2>&1
check "reads the main tip"      "git/ref/heads/main"
check "creates the branch"      "refs/heads/chore/demo-appversion-1.1.0"
check "commits via contents API" "contents/charts/demo/Chart.yaml"
check "opens the pull request"  "pr create"
check "enables auto-merge"      "--rebase"

echo
echo "== is a no-op when the chart already pins the version =="
setup
printf '1.1.0' > "${FAKE_GH_DIR}/appversion"
bash "${target}" charts/demo/Chart.yaml 1.1.0 demo >/dev/null 2>&1
check_status "exits clean" 0 $?
if grep -qF -- "pr create" "${FAKE_GH_DIR}/calls.log"; then
  echo "FAIL no-op: opened a pull request anyway"; failures=$((failures + 1))
else
  echo "PASS no-op: opened no pull request"
fi

echo
echo "== fails loudly when the branch cannot be created =="
setup
FAKE_GH_FAIL="git/refs" bash "${target}" charts/demo/Chart.yaml 1.1.0 demo >/dev/null 2>&1
check_status "propagates the failure" 1 $?

echo
if [ "${failures}" -eq 0 ]; then
  echo "all tests passed"
else
  echo "${failures} test(s) failed"
fi
[ "${failures}" -eq 0 ]
