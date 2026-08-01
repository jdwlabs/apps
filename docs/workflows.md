# Common Development Workflows

## Adding a New Angular Library

```bash
# Generate the library
pnpm exec nx g @nx/angular:library --name=my-lib --directory=libs/frontend/myapp/my-lib --tags="type:util,scope:myapp,framework:angular"

# Run lint + test to verify it's set up correctly
pnpm exec nx run <lib-project-name>:lint
pnpm exec nx run <lib-project-name>:test
```

## Adding a New Angular App

```bash
# Generate the app
pnpm exec nx g @nx/angular:application --name=myapp --directory=apps/frontend/myapp/myapp
```

Then manually:

1. Add tags to `project.json`: `["type:app", "scope:myapp", "framework:angular"]`
2. Add `scope:myapp` isolation rule to `eslint.config.ts` `depConstraints`
3. Add remote entry to `container` `webpack.config.ts`

## Running the Local Development Stack

```bash
# Start databases and services
docker compose -f scripts/docker/compose.yaml up -d

# Start all Angular apps (discover project names first)
pnpm exec nx show projects --json | grep angular
pnpm exec nx run-many --target=serve --projects=<comma-separated-app-names> --parallel=4
```

## Running Affected Lint + Tests

```bash
# Against main branch (default)
pnpm exec nx affected -t lint test

# Against a specific base SHA
pnpm exec nx affected -t lint test --base=<sha>

# All projects (slow — prefer affected)
pnpm exec nx run-many -t lint test
```

## Committing Changes

```bash
# Interactive commit (recommended)
pnpm run commit

# Or manually (must match conventional commit format)
git commit -m "feat(container): add dark mode toggle"
```

## Releasing a New Version

Versioning runs in CI on push to `main`. Do not run `nx release` locally without `--dry-run`.
After merging, CI will:

1. Run `nx release` job: bump versions, generate per-project CHANGELOGs, create tags, publish GitHub Releases
2. Run per-project deliver matrix: Docker image build/push, update Helm chart appVersion, update Docker Hub description
3. Dispatch E2E tests

### Why the release App bypasses the `Baseline` ruleset

The release GitHub App holds `bypass_mode: always` on `Baseline`, the ruleset
protecting `main`. That is an exception to "main is a merge target only", and it
is load-bearing rather than an oversight.

`nx release` runs with `git.push: true`. It rewrites each project's version
manifest, writes the changelogs, commits `chore(release): publish [skip ci]`, and
pushes that commit **and** its tags onto `main` directly. `Baseline` requires a
pull request with one approving review, linear history, and the `main` status
check, so without the bypass the release cannot land.

The narrower `bypass_mode: pull_request` does not help here. It permits an actor
to bypass rules _on a pull request_ while still blocking a direct push — and a
direct push is exactly what this job does. Narrowing the mode without first
changing the release flow stops publishing outright.

Moving the release onto a pull request is not a configuration change either,
because the tree is the source of truth for versions. Eight projects resolve
their version by reading a manifest out of the working tree at delivery time:

```
"$(tr -d '[:space:]' < apps/backend/servicediscovery/VERSION)"
```

That expression backs both `build-image` and `update-app`. Releasing therefore
_means_ writing to the tree, and writing to the tree means writing to `main`.
Removing the bypass requires first re-sourcing versions from the git tag for
every one of those projects, plus the custom version actions in `tools/release/`
— a change to the pipeline that publishes production images, not a flag flip.

What bounds the exception in the meantime:

- The job runs only on a push to `main`, so it cannot be triggered from a branch
- It is guarded against its own commits (`github.triggering_actor` check), which
  is what stops an endless re-version loop
- `concurrency: release-main` serializes racing merges, so it never pushes onto a
  moved tip
- The commit it writes is generated, single-purpose, and recorded in the
  changelogs and GitHub Releases it creates alongside

Revisit if the version source ever moves off the tree, or if the App starts
pushing anything other than the generated release commit. Until then the honest
description is a bounded, recorded exception — not a closed hole.

## Resetting After a Failed Nx Operation

```bash
pnpm exec nx reset          # clears .nx/cache and workspace-data
pnpm install                # re-install if needed
pnpm exec nx show projects  # verify project graph is healthy
```
