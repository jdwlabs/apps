# AGENTS.md

Context for AI agents (OpenAI Codex, GitHub Copilot, and others) working in this repository.

## What This Repo Is

jdwlabs `apps` is an NX monorepo containing the full application stack for the jdwlabs platform:

- **Auth UI** (`apps/angular/authui`) — login, registration, session management
- **Roles UI** (`apps/angular/rolesui`) — role assignment and management
- **Users UI** (`apps/angular/usersui`) — user listing and administration
- **Container** (`apps/angular/container`) — shell app that composes micro-frontends via Module Federation
- **Platform E2E** (`apps/angular/platform-e2e`) — Cypress end-to-end test suite
- **Service Discovery** (`apps/go/servicediscovery`) — Go backend service registry
- **Users/Role service** (`apps/springboot/usersrole`) — Spring Boot/Kotlin user-role assignment API
- **Auth DB** (`apps/database/authdb`) — database migration management

## Key Concepts

- **Module Federation:** frontends are micro-frontends composed at runtime via Webpack Module Federation — each Angular app is independently deployable but the container app assembles them
- **NX affected:** CI only builds/tests code touched by a PR — understand the dependency graph before assuming a change is isolated (`npx nx graph` to visualize)
- **go.work:** a Go workspace at the repo root covers `apps/go/servicediscovery` and `libs/go/shared/util` — always run `go` commands from the repo root

## Navigation

- Angular entry points: `apps/angular/<app>/src/main.ts`
- Go entry point: `apps/go/servicediscovery/main.go`
- Spring Boot entry point: `apps/springboot/usersrole/src/main/kotlin/`
- Shared Angular code: `libs/angular/`
- Shared Go code: `libs/go/`
- NX project graph: `npx nx graph`

## Constraints

- Do not add direct dependencies between Angular apps — use shared libs in `libs/angular/`
- All secrets come from environment variables injected at deploy time — no hardcoded secrets
- Do not push to remote without developer review
