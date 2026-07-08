# CLAUDE.md — jdwlabs/apps Monorepo

This file is read by Claude Code at the start of every session. Keep it current.

## Overview

Nx 22 monorepo with Angular 21 micro-frontend apps, Go services, a Spring Boot service, PostgreSQL migration runners, and shared Angular libraries. CI runs on GitHub-hosted runners (`ubuntu-latest`) — the repo is public, so this is free and avoids OOMing the self-hosted cluster. Package manager is **pnpm 10**.

## Worktree Location (Windows — CRITICAL)

This repo lives on **F: drive** (`F:\Dev\projects\personal\jdwlabs\apps`).

**Worktrees MUST be created on the same drive (F:).**

pnpm uses hard links for its content-addressable store. Hard links cannot cross NTFS volume boundaries. If a worktree is on C: and the repo is on F:, `pnpm install` will silently succeed but produce an empty `node_modules` (only `.pnpm` dir) — no binaries, no hoisted packages. Git hooks that call `npx --no-install <tool>` will then fail.

```bash
# CORRECT — same drive as repo
gwta feat/my-feature   # creates F:\Dev\worktrees\apps\feat\my-feature

# or manually:
git worktree add F:/Dev/worktrees/apps/feat/my-feature -b feat/my-feature

# keep worktrees on F: drive:
export WT_BASE=F:/Dev/worktrees   # in ~/.bashrc
```

If a cross-drive worktree left `node_modules` with only `.pnpm` and no `.bin`, replace it with a junction to the main repo's `node_modules`:

```powershell
Remove-Item -Recurse -Force C:\path\to\worktree\node_modules
New-Item -ItemType Junction -Path C:\path\to\worktree\node_modules -Target F:\Dev\projects\personal\jdwlabs\apps\node_modules
```

## Directory Map

Role-based layout (as of JDWLABS-20):

```
apps/
  frontend/     # Angular micro-frontend apps (authui, container, rolesui, usersui)
  backend/      # Go (servicediscovery) + Spring Boot (usersrole) services
  database/     # DB migration runners (authdb — PostgreSQL)
  e2e/          # Playwright E2E tests
libs/
  frontend/     # Shared Angular libs (per-app: authui/container/rolesui/usersui + shared)
  backend/      # Go shared packages
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

Each app has libs under `libs/frontend/<app-name>/`: feature/, data-access/, and util/ (ui/ in shared).

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
- Styles: SCSS only. Themes in `libs/frontend/shared/ui/src/lib/styles/themes/`
- No barrel `index.ts` files at app level — use direct path imports
- Go packages: lowercase single-word names, follow standard Go layout
- Never put a Jira ticket ID (`JDWLABS-*`) or PR/issue number in a code comment — traceability lives in the commit message and PR description, not in source that outlives the ticket

## Versioning / Release

Versioning is managed by `nx release` (config in `nx.json` under `release`). Independent
per-project versions, tags `{projectName}-{version}`, conventional-commit driven bumps
(feat=minor, fix=patch, breaking=major; chore/docs/etc never bump). Version manifests:
Angular apps `public/VERSION`, backend/database apps `VERSION` at project root
(custom actions in `tools/release/`), shared libs `package.json`. Per-project
`CHANGELOG.md` and GitHub Releases are generated by the release job in CI.
**Do not manually edit version files. Never run `nx release` locally without `--dry-run`.**

## CI/CD

GitHub Actions on `ubuntu-latest` (GitHub-hosted):

1. `nx format:check` — formatting gate
2. PR only: `commitlint` — validates all commit messages in the PR
3. `nx affected -t lint test` — affected projects
4. `nx affected -t build` — affected builds
5. **Main push only:** `nx release` job (version, changelog, tag, GitHub Release), then per-project deliver matrix (Docker image → deployments Helm bump → Docker Hub description), then E2E dispatch.

## Do Not

- `git push --force` to `main`
- `--no-verify` on commits — pre-commit (lint-staged) and commit-msg (commitlint) hooks run for a reason
- Skip `pnpm exec nx reset` after editing `project.json` targets
- Import `scope:authui` libs from `scope:container` code — use `scope:shared` instead
- Add new projects without `type:`, `scope:`, and `framework:` tags in `project.json`
- Commit anything to `docs/superpowers/` — it is gitignored intentionally (local planning only)
- Modify `.github/workflows/ci.yml` buildx/`build-image` steps without understanding the docker-container BuildKit + multi-arch qemu setup
- Run `nx release` without `--dry-run` locally — real releases are CI-only
