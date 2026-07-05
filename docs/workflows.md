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

## Resetting After a Failed Nx Operation

```bash
pnpm exec nx reset          # clears .nx/cache and workspace-data
pnpm install                # re-install if needed
pnpm exec nx show projects  # verify project graph is healthy
```
