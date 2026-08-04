# Release Job Partial-Failure Recovery

The CI `release` job (`.github/workflows/ci.yml`) runs `nx release --skip-publish`
against the live `main` tip, then the `deliver` matrix builds/publishes each
released project. Because this spans a tag push, GitHub Release creation, and a
Docker Hub matrix, a mid-run failure can leave state partially applied. This
runbook covers recovery for each failure point.

A release writes nothing to `main` — the tag is the only record it creates — so
`main` is never left in a half-released state and there is no version-bump
commit to reconcile.

## (a) Tags missing

**Symptom:** `Run Nx Release` step fails before (or during) tag creation/push.

**Recovery:** Re-run the `release` job. `nx release` computes versions from
conventional commits since the last matching tag, and nothing about the failed
attempt persists, so the re-run recomputes the _same_ version for each affected
project and pushes the missing tags — provided `main` has not moved. If an
intervening commit landed first, the re-run tags that newer tip instead; reach
for a manual `git tag` + push against the intended commit only in that case.

**Partial tag pushes are not possible.** The push is `--atomic`, so either
every tag in the release lands or none does.

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

**Symptom:** `release` job succeeded (tags + GitHub Release both landed) but
one or more `deliver` matrix jobs failed (Docker build, Helm chart update, or
Docker Hub description push).

**Recovery:** Idempotent — re-run only the failed matrix job(s) from the
Actions UI ("Re-run failed jobs"). `deliver` reads `needs.release.outputs.sha`
and `matrix` from the already-completed `release` job's outputs, so it
re-runs against the same released commit and project set without re-triggering
`nx release`. Verified safe to re-run: `build-image`, `update-app`, and
`update-description` targets are all safe to repeat against the same version.
