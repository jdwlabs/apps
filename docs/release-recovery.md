# Release Job Partial-Failure Recovery

The CI `release` job (`.github/workflows/ci.yml`) runs `nx release --skip-publish`
against the live `main` tip, then the `deliver` matrix builds/publishes each
released project. Because this spans a git push, tag push, GitHub Release
creation, and a Docker Hub matrix, a mid-run failure can leave state partially
applied. This runbook covers recovery for each failure point.

## (a) Commit pushed, tags missing

**Symptom:** `Run Nx Release` step fails after `nx release` pushed the version
bump commit to `main` but before (or during) tag creation/push.

**Recovery:** Re-run the `release` job. `nx release` computes versions from
conventional commits since the last matching tag; since the version-bump
commit is already on `main` and unversioned (no tag points at it yet), the
re-run recomputes the _same_ version for each affected project and pushes the
missing tags. No manual tag creation should be necessary — only reach for a
manual `git tag` + push if the re-run computes a different version than
expected (e.g. an intervening commit landed on `main` first).

## (b) Tags pushed, GitHub Release missing

**Symptom:** Tags exist on the released commit, but no GitHub Release was
created (e.g. the job died between tag push and Release API call).

**Recovery:** Re-running the `release` job no-ops — `nx release` sees the tag
already points at the target commit and won't re-version or re-push. Recover
manually per project:

```bash
gh release create <project>-<version> --title "<project> <version>" --generate-notes
```

## (c) Deliver failure

**Symptom:** `release` job succeeded (commit + tags + GitHub Release all
landed) but one or more `deliver` matrix jobs failed (Docker build, Helm chart
update, or Docker Hub description push).

**Recovery:** Idempotent — re-run only the failed matrix job(s) from the
Actions UI ("Re-run failed jobs"). `deliver` reads `needs.release.outputs.sha`
and `matrix` from the already-completed `release` job's outputs, so it
re-runs against the same released commit and project set without re-triggering
`nx release`. Verified safe to re-run: `build-image`, `update-app`, and
`update-description` targets are all safe to repeat against the same version.
