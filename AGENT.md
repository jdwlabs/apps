# AGENT.md — jdwlabs/apps Monorepo

Agent instructions for OpenAI Codex, Gemini, GitHub Copilot, and other AI coding agents.
Claude Code users: see `CLAUDE.md`.

## Safe to Run Autonomously

```bash
pnpm exec nx affected -t lint test        # run affected lint + tests
pnpm exec nx format:check                 # check formatting
pnpm exec nx format:write                 # fix formatting
pnpm exec nx run <project>:<target>       # run any build/lint/test target
pnpm exec nx reset                        # clear Nx cache
pnpm exec nx show projects                # list all projects
docker compose -f scripts/docker/compose.yaml up -d   # start local stack
```

## Requires Confirmation Before Running

- `git push` — always confirm destination branch
- `docker buildx build --push` — pushes to DockerHub
- Any `git reset`, `git rebase`, or `git push --force`
- Changes to `.github/workflows/ci.yml`

## Never Run

- `git push --force origin main`
- `git commit --no-verify`
- `npm install` or `yarn` — this project uses **pnpm** only

## Repository Layout

```
apps/angular/        Angular apps: authui, container, rolesui, usersui
apps/go/             Go services: servicediscovery
apps/springboot/     Spring Boot: usersrole
apps/database/       PostgreSQL: authdb
libs/angular/        Shared Angular libs (per-app + shared)
libs/go/             Go shared packages
tools/agents/        Docker dev agent (do not modify structure)
scripts/             Shell helpers and Docker Compose
```

## Commit Format

`<type>(<scope>): <subject>`

Types: feat fix chore docs style refactor perf test build ci revert
Scope: kebab-case (optional but encouraged)
Subject: lowercase, no trailing period, max 100 chars

## Module Boundary Rules

Angular libs are tag-scoped. A `scope:container` lib may only import from `scope:container` or `scope:shared` libs. Cross-scope imports will fail ESLint lint. See `eslint.config.ts` `depConstraints` for the full ruleset.

## Testing

- Unit tests: Jest (`pnpm exec nx run <project>:test`)
- E2E: Playwright in `apps/angular/platform-e2e` (`pnpm exec nx run platform-e2e:e2e`)
- No mocking of Nx project graph or module boundaries in tests

## Versioning

`nx release` manages versioning via conventional commits (config in `nx.json`). Per-project versions, independent tags `{projectName}-{version}`, conventional-commit driven bumps (feat=minor, fix=patch, breaking=major; chore never bumps). Version manifests: Angular apps `public/VERSION`, backend/database apps `VERSION` at root, shared libs `package.json`. Do not manually edit version files. **Never run `nx release` locally without `--dry-run` — releases run in CI only.** GitHub Releases and per-project CHANGELOG.md are auto-generated.
