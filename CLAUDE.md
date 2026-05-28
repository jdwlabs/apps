# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working in this repository.

## Repository Overview

NX monorepo for jdwlabs application services. Contains Angular micro-frontends, Go backend services, and database migration tooling.

### Structure

- `apps/frontend/` — Angular micro-frontend apps (authui, rolesui, usersui)
- `apps/backend/` — Go backend services (servicediscovery, usersrole)
- `apps/database/` — Database migration apps (authdb)
- `apps/e2e/` — Cypress end-to-end tests
- `libs/frontend/` — Shared Angular libraries
- `libs/backend/` — Shared Go libraries

### Tech Stack

- **Frontend:** Angular, Module Federation, Jest, Cypress
- **Backend:** Go (workspace at repo root via go.work)
- **Monorepo tooling:** NX, pnpm

## Development Commands

### Build

```bash
npx nx build <app-name>           # Build a single app
npx nx run-many -t build          # Build all apps
```

### Test

```bash
npx nx test <app-name>            # Unit tests for one app
npx nx run-many -t test           # All unit tests
npx nx e2e <app-name>-e2e         # Cypress E2E tests
```

### Lint

```bash
npx nx lint <app-name>            # Lint one app
npx nx run-many -t lint           # Lint all
```

### Affected (used in CI to scope work to changed code)

```bash
npx nx affected -t build          # Build only what changed vs main
npx nx affected -t test           # Test only what changed
npx nx affected -t lint           # Lint only what changed
```

### Go (backend services)

```bash
go build ./...                    # Run from repo root (go.work)
go test ./...                     # Run all Go tests
```

## Common Tasks

### Add a new Angular app

```bash
npx nx g @nx/angular:application <name> --directory=apps/frontend/<name>
```

### Add a new shared Angular library

```bash
npx nx g @nx/angular:library <name> --directory=libs/frontend/<name>
```

### Add a dependency

```bash
pnpm add <package> --filter <project-name>
pnpm install --frozen-lockfile    # Restore lockfile-exact install
```

## AI Agent Contract

- Use `npx nx` for all build/test/lint operations — never invoke `ng` directly
- For Go: use `go build ./...` and `go test ./...` from the repo root (go.work handles workspace)
- Do not modify `pnpm-lock.yaml` directly — run `pnpm install` to update
- Do not run `git push` — leave that to the developer
- CI runs `nx affected` — changes that don't appear in the affected graph will not be tested in CI

## References

- [NX Documentation](https://nx.dev)
- [Conventional Commits](CONTRIBUTING.md)
