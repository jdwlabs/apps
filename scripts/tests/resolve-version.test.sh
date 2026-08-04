#!/bin/bash
# Offline tests for resolve-version.sh. Each case runs against a throwaway git
# repo so the assertions are about real tag resolution rather than a stub.
set -uo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
target="${script_dir}/../resolve-version.sh"
failures=0

setup() {
  repo=$(mktemp -d)
  git -C "${repo}" init -q
  git -C "${repo}" config user.email test@example.com
  git -C "${repo}" config user.name test
  git -C "${repo}" config commit.gpgsign false
  git -C "${repo}" config tag.gpgsign false
  git -C "${repo}" commit -q --allow-empty -m "init"
  unset RELEASE_VERSION || true
}

run() {
  (cd "${repo}" && bash "${target}" "$@" 2>/dev/null)
}

check() {
  local label="$1" expected="$2" actual="$3"
  if [ "${expected}" = "${actual}" ]; then
    echo "PASS ${label}"
  else
    echo "FAIL ${label}: expected '${expected}', got '${actual}'"
    failures=$((failures + 1))
  fi
}

echo "== resolves the tag pointing at HEAD =="
setup
git -C "${repo}" tag -a demo-1.2.3 -m demo-1.2.3
check "reads the tag" "1.2.3" "$(run demo)"

echo
echo "== prefers RELEASE_VERSION over any tag =="
setup
git -C "${repo}" tag -a demo-1.2.3 -m demo-1.2.3
check "reads the env" "9.9.9" "$(RELEASE_VERSION=9.9.9 run demo)"

echo
echo "== falls back to the newest reachable tag when HEAD is ahead =="
setup
git -C "${repo}" tag -a demo-1.2.3 -m demo-1.2.3
git -C "${repo}" tag -a demo-1.10.0 -m demo-1.10.0
git -C "${repo}" commit -q --allow-empty -m "later work"
# Sorted as semver, not as text: 1.10.0 must beat 1.2.3.
check "reads the newest tag" "1.10.0" "$(run demo)"

echo
echo "== ignores tags belonging to another project =="
setup
git -C "${repo}" tag -a other-4.5.6 -m other-4.5.6
run demo >/dev/null 2>&1
check "exits non-zero" "1" "$?"

echo
echo "== refuses to guess when nothing resolves =="
setup
run demo >/dev/null 2>&1
check "exits non-zero" "1" "$?"

echo
if [ "${failures}" -eq 0 ]; then
  echo "all tests passed"
else
  echo "${failures} test(s) failed"
fi
[ "${failures}" -eq 0 ]
