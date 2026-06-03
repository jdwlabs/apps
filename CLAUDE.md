# CLAUDE.md — jdwlabs/apps Monorepo

This file is read by Claude Code at the start of every session. Keep it current.

## Overview

Nx 22 monorepo with Angular 21 micro-frontend apps, Go services, a Spring Boot service, PostgreSQL migration runners, and shared Angular libraries. All CI runs on self-hosted ARC runners (`ubuntu-jdwlabs`). Package manager is **pnpm**.

## Directory Map

```
apps/
  angular/      # Angular deployable apps (authui, container, rolesui, usersui)
  go/           # Go services (servicediscovery)
  springboot/   # Spring Boot services (usersrole)
  database/     # DB migration runners (authdb — PostgreSQL)
libs/
  angular/      # Shared Angular libs (per-app: authui/container/rolesui/usersui + shared)
  go/           # Go shared packages
tools/
  agents/       # Nx-adjacent Docker dev agent — DO NOT move or restructure
scripts/        # Non-Nx shell scripts and Docker Compose helpers
  docker/       # docker/compose.yaml — local dev stack
docs/
                # architecture, conventions, workflows, onboarding
```

## Key Commands

```bash
# Affected lint + test (use this during development)
pnpm exec nx affected -t lint test

# Format check (must pass in CI)
pnpm exec nx format:check

# Auto-fix formatting
pnpm exec nx format:write

# Run a specific Nx target
pnpm exec nx run <project>:<target>
# e.g. pnpm exec nx run container:build

# Reset Nx project graph (required after editing project.json)
pnpm exec nx reset

# Interactive commit (commitizen)
pnpm run commit

# Start local Docker stack
docker compose -f scripts/docker/compose.yaml up -d
```

## Nx Project Tags

Every `project.json` has three tag families:

- `type:app` | `type:lib` | `type:e2e` | `type:feature` | `type:ui` | `type:data-access` | `type:util`
- `scope:<name>` — app-level scope; names come from `project.json` files. Run `pnpm exec nx show projects` for the current list.
- `framework:angular` | `framework:go` | `framework:springboot` | `framework:database` | `framework:playwright`

Module boundary rules in `eslint.config.ts` enforce:

- Per-app-scope isolation: a `scope:X` lib may only import from `scope:X` or `scope:shared` libs
- Framework isolation: `framework:angular` libs may only import from other `framework:angular` libs
- Type hierarchy: `type:feature` → `type:ui` + `type:util` + `type:data-access` (no circular)

## Angular Architecture

Four Angular apps use **webpack module federation**:

- `container` — shell app, loads remotes at runtime
- `authui`, `rolesui`, `usersui` — remote apps

Each app has libs under `libs/angular/<app-name>/`: feature/, data-access/, and util/ (ui/ in shared).

## Commit Conventions

Format: `<type>(<scope>): <subject>` — scope is optional but encouraged.

Allowed types: `feat` `fix` `chore` `docs` `style` `refactor` `perf` `test` `build` `ci` `revert`

Rules:

- Scope: kebab-case
- Subject: lowercase, no trailing period, max header 100 chars
- Body: blank line before body if present

```
feat(authui): add oauth2 login flow
fix(container): resolve mfe chunk loading error
chore(deps): bump @angular/core to 21.2.10
build(lint): migrate to eslint flat config
```

Use `pnpm run commit` for the interactive Commitizen prompt.

## Coding Conventions

- Angular file naming: kebab-case, suffix-typed (`.component.ts`, `.service.ts`, `.spec.ts`)
- Styles: SCSS only. Themes in `libs/angular/shared/ui/src/lib/styles/themes/`
- No barrel `index.ts` files at app level — use direct path imports
- Go packages: lowercase single-word names, follow standard Go layout

## Versioning / Release

Versioning is managed by `@jscutlery/semver` via the `version` Nx target on each app. It reads conventional commits, bumps semver, tags the repo, and triggers `postTargets` (format, save-version, build-image, update-app). **Do not manually edit version files.**

## CI/CD

GitHub Actions on `ubuntu-jdwlabs` (self-hosted ARC runner):

1. `nx format:check` — formatting gate
2. PR only: `commitlint` — validates all commit messages in the PR
3. `nx affected -t lint test` — affected projects
4. `nx affected -t build` — affected builds
5. **Main push only:** Docker image build+push (DockerHub), semver bump, Helm Chart update, push to `develop`

## Do Not

- `git push --force` to `main`
- `--no-verify` on commits — pre-commit (lint-staged) and commit-msg (commitlint) hooks run for a reason
- Skip `pnpm exec nx reset` after editing `project.json` targets
- Import `scope:authui` libs from `scope:container` code — use `scope:shared` instead
- Add new projects without `type:`, `scope:`, and `framework:` tags in `project.json`
- Commit anything to `docs/superpowers/` — it is gitignored intentionally (local planning only)
- Modify `.github/workflows/ci.yml` `build-image` steps without understanding ARC + Kubernetes BuildKit constraints
- Run `nx affected -t version` locally — semver tagging runs in CI only
