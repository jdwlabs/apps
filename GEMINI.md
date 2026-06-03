# GEMINI.md

This file provides guidance to Gemini CLI when working in this repository.
For the canonical reference, see [CLAUDE.md](CLAUDE.md) — this file mirrors that content.

## Repository Overview

NX monorepo for jdwlabs application services. Contains Angular micro-frontends, Go backend services, a Spring Boot/Kotlin service, and database migration tooling.

### Structure

- `apps/angular/` — Angular micro-frontend apps (authui, container, rolesui, usersui, platform-e2e)
- `apps/go/` — Go services (servicediscovery)
- `apps/springboot/` — Spring Boot/Kotlin service (usersrole)
- `apps/database/` — Database migration apps (authdb)
- `libs/angular/` — Shared Angular libraries
- `libs/go/` — Shared Go libraries

## Development Commands

```bash
npx nx build <app-name>           # Build one app
npx nx test <app-name>            # Test one app
npx nx lint <app-name>            # Lint one app
npx nx affected -t build,test     # CI-equivalent: affected apps only
go build ./...                    # Go workspace build (from repo root)
go test ./...                     # Go workspace test (from repo root)
```

## Agent Contract

- Use `npx nx` for all Angular build/test/lint — never `ng` directly
- Use `go` commands from repo root for Go services (`go.work` covers `apps/go/` and `libs/go/`)
- Do not modify lockfiles directly
- Do not push to remote — stage and commit only
