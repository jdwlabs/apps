# AGENTS.md

Canonical context for AI agents (Claude Code, OpenAI Codex, Gemini CLI, GitHub Copilot, and others) working in this repository. `CLAUDE.md`, `GEMINI.md`, and `AGENT.md` are thin pointers to this file — make edits here.

## What This Repo Is

jdwlabs `apps` is an Nx 22 monorepo containing the full application stack for the jdwlabs platform. Angular 21 micro-frontends, Go services, a Spring Boot/Kotlin service, PostgreSQL migration runners, and shared libraries. Package manager is **pnpm 10**. CI runs on GitHub-hosted runners (`ubuntu-latest`).

- **Auth UI** (`apps/frontend/authui`) — login, registration, session management
- **Roles UI** (`apps/frontend/rolesui`) — role assignment and management
- **Users UI** (`apps/frontend/usersui`) — user listing and administration
- **Container** (`apps/frontend/container`) — shell app that composes micro-frontends via Module Federation
- **Platform E2E** (`apps/e2e/platform-e2e`) — Playwright end-to-end test suite
- **Service Discovery** (`apps/backend/servicediscovery`) — Go backend service registry
- **AI-SRE Relay** (`apps/backend/ai-sre-relay`) — Go alert-relay service for the AI-SRE stack
- **Users/Role service** (`apps/backend/usersrole`) — Spring Boot/Kotlin user-role assignment API
- **Auth DB** (`apps/database/authdb`) — database migration management

## Directory Map

```
apps/
  frontend/     # Angular micro-frontend apps (authui, container, rolesui, usersui)
  backend/      # Go (servicediscovery, ai-sre-relay) + Spring Boot (usersrole) services
  database/     # DB migration runners (authdb — PostgreSQL)
  e2e/          # Playwright E2E tests (platform-e2e)
libs/
  frontend/     # Shared Angular libs (per-app: authui/container/rolesui/usersui + shared)
  backend/      # Go shared packages
tools/
  agents/       # Nx-adjacent Docker dev agent — DO NOT move or restructure
  release/      # Custom nx release version actions
  testing/      # Shared test tooling
scripts/        # Non-Nx shell scripts and Docker Compose helpers
  docker/       # docker/compose.yaml — local dev stack
docs/           # architecture, conventions, workflows, onboarding
```

## Key Concepts

- **Module Federation:** frontends are micro-frontends composed at runtime via Webpack Module Federation — each Angular app is independently deployable but the `container` app assembles them
- **NX affected:** CI only builds/tests code touched by a PR — understand the dependency graph before assuming a change is isolated (`npx nx graph` to visualize)
- **go.work:** a Go workspace at the repo root covers `apps/backend/servicediscovery`, `apps/backend/ai-sre-relay`, and `libs/backend/shared/util` — always run `go` commands from the repo root

## Worktree Location (Windows — CRITICAL)

This repo lives on **F: drive** (`F:\Dev\projects\personal\jdwlabs\apps`).
**Worktrees MUST be created on the same drive (F:).**

pnpm uses hard links for its content-addressable store. Hard links cannot cross NTFS volume boundaries. If a worktree is on C: and the repo is on F:, `pnpm install` will silently succeed but produce an empty `node_modules` (only `.pnpm` dir) — no binaries, no hoisted packages. Git hooks that call `npx --no-install <tool>` will then fail.

```bash
# CORRECT — same drive as repo
git worktree add F:/Dev/worktrees/apps/feat/my-feature -b feat/my-feature
```

Do **not** set `NX_CACHE_DIRECTORY`. Nx resolves its cache against the main
worktree root, so every worktree already shares one cache at
`apps/.nx/cache` — see [docs/nx-caching.md](docs/nx-caching.md). Exporting the
variable in some shells and not others splits the cache in two and makes
whichever half you did not use look cold. Never accept an `nx connect` / Nx
Cloud prompt; this repo is self-hosted-cache only.

If a cross-drive worktree left `node_modules` with only `.pnpm` and no `.bin`, replace it with a junction to the main repo's `node_modules`:

```powershell
Remove-Item -Recurse -Force C:\path\to\worktree\node_modules
New-Item -ItemType Junction -Path C:\path\to\worktree\node_modules -Target F:\Dev\projects\personal\jdwlabs\apps\node_modules
```

## Key Commands

```bash
pnpm exec nx affected -t lint test        # affected lint + tests (use during development)
pnpm exec nx format:check                 # check formatting (must pass in CI)
pnpm exec nx format:write                 # fix formatting
pnpm exec nx run <project>:<target>       # run any build/lint/test target
pnpm exec nx reset                        # clear Nx cache (required after editing project.json)
pnpm exec nx show projects                # list all projects
pnpm run commit                           # interactive commit (commitizen)
docker compose -f scripts/docker/compose.yaml up -d   # start local stack
go build ./... && go test ./...           # Go workspace build/test (from repo root)
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

Use `pnpm run commit` for the interactive Commitizen prompt.

## Coding Conventions

- Angular file naming: kebab-case, suffix-typed (`.component.ts`, `.service.ts`, `.spec.ts`)
- Styles: SCSS only. Themes in `libs/frontend/shared/ui/src/lib/styles/themes/`
- No barrel `index.ts` files at app level — use direct path imports
- Go packages: lowercase single-word names, follow standard Go layout
- Never put a Jira ticket ID (`JDWLABS-*`) or PR/issue number in a code comment — traceability lives in the commit message and PR description, not in source that outlives the ticket

## Tooling Traps

RTK's filtered output is **not** the tool's output — it summarises, truncates, and
prints its own status lines. Every `rtk` row below is that one root cause. Run
anything you intend to act on through `rtk proxy <cmd>` and read the raw result.

| Symptom                                                                     | Cause                                                                                                                                                                                                                                                                     | Fix                                                                                                                                                                                                                       |
| --------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `rtk go build -o <path>` reports `Go build: Success` with no binary written | RTK's success line doesn't reflect the actual Go toolchain result — it suppresses the compiler's **output** (the exit code is still propagated, so check it); inside a git worktree the underlying failure is VCS stamping, `error obtaining VCS status: exit status 128` | `rtk proxy go build ...` for the real output; add `-buildvcs=false` when building from a worktree                                                                                                                         |
| `gh pr edit` fails on every PR in this org                                  | `gh` resolves the org through a GraphQL **query** that requires the `read:org` scope, and the active `GITHUB_TOKEN` (`ghp_...`) lacks it — it fails before any mutation is attempted (`the 'login' field requires ... ['read:org']`)                                      | `unset GITHUB_TOKEN` so `gh` falls back to the keyring `gho_` OAuth token, which already carries `read:org`. Fallback if that token is unavailable: `gh api -X PATCH repos/<owner>/<repo>/pulls/<n> --input payload.json` |
| `gh run watch <n>` errors or watches nothing                                | It takes the run's **databaseId**, not the run number shown in the UI or in a `gh run list` number column                                                                                                                                                                 | Resolve it first — `gh run list --json databaseId,number,headBranch` — and pass the `databaseId`                                                                                                                          |
| `curl --cacert <ca>.pem https://host` returns HTTP 000 on Windows           | Windows curl uses the **Schannel** TLS backend, which silently ignores `--cacert`. HTTP 000 is a generic connection failure and is **not** evidence that the CA is wrong or untrusted                                                                                     | Test the CA with `openssl s_client -connect <host>:<port> -CAfile <ca>.pem`, which actually honours the bundle                                                                                                            |

## Verify Before You Start

Ticket evidence more than ~a week old (or gathered in a different investigation) is a hypothesis, not ground truth. Before acting on it:

- Re-confirm the ticket's premises against live state — don't build on a stale finding
- State the scope you searched before claiming something is absent ("checked all N projects", "every affected target") — one sample is not the whole set
- A disproved premise is a valuable result: record it on the ticket, don't quietly work around it

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
5. **Main push only:** the `release` job resolves every project's delivery targets from `nx graph` _first_, then runs `nx release` (version, changelog, tag, GitHub Release) and joins the pushed tags against that map. Downstream: per-project deliver matrix (Docker image → deployments Helm bump → Docker Hub description), then E2E dispatch.

Target resolution deliberately precedes `nx release` because tagging is irreversible and graph resolution is not. Once tags exist, the job must fail rather than produce an empty matrix — a skipped `deliver` reads as benign on the run summary while leaving versions tagged with no image behind them. `deliver` and `dispatch-e2e` gate on the `released` output, so "nothing to release" and "detection broke" are distinguishable.

## Autonomy Boundaries

Safe to run autonomously: any `pnpm exec nx ...` build/lint/test/format target, `docker compose -f scripts/docker/compose.yaml up -d`.

Require confirmation first:

- `git push` — always confirm destination branch
- `docker buildx build --push` — pushes to DockerHub
- Any `git reset`, `git rebase`, or `git push --force`
- Changes to `.github/workflows/ci.yml`

## Do Not

- `git push --force` to `main`
- `--no-verify` on commits — pre-commit (lint-staged) and commit-msg (commitlint) hooks run for a reason
- `npm install` or `yarn` — this project uses **pnpm** only
- Skip `pnpm exec nx reset` after editing `project.json` targets
- Add direct dependencies between Angular apps — use shared libs in `libs/frontend/` (e.g. never import `scope:authui` libs from `scope:container` code; use `scope:shared`)
- Add new projects without `type:`, `scope:`, and `framework:` tags in `project.json`
- Commit anything to `docs/superpowers/` — it is gitignored intentionally (local planning only)
- Modify `.github/workflows/ci.yml` buildx/`build-image` steps without understanding the docker-container BuildKit + multi-arch qemu setup
- Run `nx release` without `--dry-run` locally — real releases are CI-only
- Hardcode secrets — all secrets come from environment variables injected at deploy time
