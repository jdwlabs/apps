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

## What CI Runs on a Pull Request

`.github/workflows/ci.yml` opens with four jobs that all start at once:

| job          | what it does                                                               | required check |
| ------------ | -------------------------------------------------------------------------- | -------------- |
| `main`       | format, commitlint, affected lint/test, affected build, release dry-run    | yes            |
| `e2e-shard`  | N runners, each `nx affected -t e2e --shard=<i>/<N>`, one blob report each | no             |
| `e2e`        | reads the matrix outcome and nothing else — the suite's one verdict        | yes            |
| `e2e-report` | merges the shard blobs into one `platform-e2e-report` artifact             | no             |

The shards are deliberately **not** required checks: a required context is
matched by name, so requiring `e2e-shard (1)`…`(N)` would pin branch protection
to the matrix size and strand a context that never reports again the moment N
changes. `e2e` is required instead, and it runs under `!cancelled()` so it always
reports — a required check that never reports blocks the branch forever.

`e2e-report` never decides anything. It exists to explain a result, so a flake in
the artifact download costs a reviewer nothing.

### Why a CI change re-runs everything

`nx.json` lists `.github/workflows/ci.yml` under `namedInputs.sharedGlobals`, so
editing this workflow marks every project affected. Without it `nx affected`
resolves a CI-only diff to zero projects, and the pipeline a change is trying to
alter never actually runs against it — every job reports green having done
nothing. The cost is a full lint/test/build on workflow edits, which is the point.

### Changing the required checks

Branch protection is managed as code in `.github/rulesets/`. Adding or renaming a
required check means editing `baseline.json` in the same PR as the workflow
change, then a repo admin running `.github/rulesets/apply.sh` to push it live —
nothing applies it automatically, and the script has no delete path, so it only
ever adds or updates.

## Releasing a New Version

Versioning runs in CI on push to `main`. Do not run `nx release` locally without `--dry-run`.
After merging, CI will:

1. Run `nx release` job: work out each project's next version, create tags, publish GitHub Releases
2. Run per-project deliver matrix: Docker image build/push, update Helm chart appVersion, update Docker Hub description
3. Dispatch E2E tests

### The git tag is the only record of a version

Nothing in the working tree carries a released version. `nx release` runs with
`git.commit: false`, so a release creates tags and pushes those tags — it never
writes a commit, and therefore never writes to `main`. Every project resolves
its current version from the tag matching `releaseTag.pattern`
(`{projectName}-{version}`), and every delivery target resolves the version it
is shipping the same way:

```
"$(bash scripts/resolve-version.sh servicediscovery)"
```

That expression backs both `build-image` and `update-app`. In CI the deliver
matrix passes `RELEASE_VERSION` instead, because that job checks out a single
commit with no tags fetched; `resolve-version.sh` prefers it and falls back to
the tags for a local or manual run. It aborts rather than guessing — an image
or a chart bump carrying an invented version is worse than a build that stops.

Consequences worth knowing before changing any of this:

- **A new project needs an initial tag.** There is no manifest to fall back to,
  so `nx release` has nothing to compute a first version from. Create
  `<project>-0.0.0` on a relevant commit, or run the first release with
  `--first-release`.
- **Release notes live in GitHub Releases only.** In-tree `CHANGELOG.md` files
  would need a commit on `main` to stay current, so they are not generated
  (`changelog.projectChangelogs.file: false`).
- **Angular apps still serve `/VERSION`.** The tracked `public/VERSION` holds
  `0.0.0-dev`; `prepare-version.sh` stamps the resolved version into it before
  the image build and `restore-version.sh` puts the placeholder back. A local
  build or dev server therefore reports `0.0.0-dev`, which is true.
- **`usersrole` takes its version from Gradle.** `build.gradle.kts` reads
  `-PreleaseVersion`, then `RELEASE_VERSION`, then falls back to `0.0.0-dev`.
- **The two shared libs keep a version in `package.json`.** Nx still writes it
  during a release and the write is discarded with everything else, so the
  committed number freezes. Both are `private: true` and never published; their
  tag is the real version.

### Why the release App holds no `Baseline` bypass

It does not need one. Tag creation is governed by the `Release Tag Protection`
ruleset over `refs/tags/**`, which restricts only deletion and non-fast-forward
— the App has never held a bypass there and has been pushing release tags all
along. `Baseline` covers `refs/heads/main`, and a release sends no update to
that ref, so its rules are never evaluated.

`OrganizationAdmin` keeps `bypass_mode: always` on `Baseline`. That is the
deliberate break-glass path and a separate risk, not this one.

Revisit if a release ever starts writing to the tree again: the moment it does,
it needs somewhere to commit that write, and `main` is the only branch it is on.

## Resetting After a Failed Nx Operation

```bash
pnpm exec nx reset          # clears .nx/cache and workspace-data
pnpm install                # re-install if needed
pnpm exec nx show projects  # verify project graph is healthy
```
