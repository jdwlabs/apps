# Common Development Workflows

## Adding a New Angular Library

```bash
# Generate the library
pnpm exec nx g @nx/angular:library --name=my-lib --directory=libs/angular/myapp/my-lib --tags="type:util,scope:myapp,framework:angular"

# Run lint + test to verify it's set up correctly
pnpm exec nx run <lib-project-name>:lint
pnpm exec nx run <lib-project-name>:test
```

## Adding a New Angular App

```bash
# Generate the app
pnpm exec nx g @nx/angular:application --name=myapp --directory=apps/angular/myapp/myapp
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

Releases are driven by [Nx Release](https://nx.dev/recipes/nx-release/get-started-with-nx-release),
configured under the `release` key in `nx.json`:

- **Independent versioning** — each of the 7 releasable apps (`authui`, `container`,
  `rolesui`, `usersui`, `authdb`, `servicediscovery`, `usersrole`) versions separately
- **Tag format** — `{projectName}-{version}` (e.g. `usersui-1.3.5`), resolved from
  existing git tags (`currentVersionResolver: git-tag`)
- **Conventional commits** — the bump type (patch/minor/major) is derived from commit
  messages since the last tag
- **Per-project changelogs** — `CHANGELOG.md` in each project root
- **Version files** — `tools/release/version-actions.ts` extends the default bump to
  also write Angular `public/VERSION` files and the Spring Boot `build.gradle.kts`
  project version; non-JS apps carry a `private` version-tracking `package.json`

Releases run in CI on push to `main` (`bash scripts/release.sh --skip-publish --yes`).
Do not run `nx release` against the real repo locally — preview with a dry run instead:

```bash
# Preview version bumps, changelogs, and the release commit — writes nothing
bash scripts/release.sh --dry-run

# Preview versioning only
pnpm exec nx release version --dry-run

# Release a subset of projects (CI normally handles this)
bash scripts/release.sh --projects=usersui --dry-run
```

After Nx Release commits, tags, and pushes, CI then for each newly tagged project:

1. Builds and pushes the Docker image (`<project>:build-image`)
2. Updates the Helm chart `appVersion` in the `deployments` repo (`<project>:update-app`)

## Resetting After a Failed Nx Operation

```bash
pnpm exec nx reset          # clears .nx/cache and workspace-data
pnpm install                # re-install if needed
pnpm exec nx show projects  # verify project graph is healthy
```
