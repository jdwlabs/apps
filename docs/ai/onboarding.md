# New Developer / AI Session Onboarding

## Prerequisites

- Node.js 22+
- pnpm (`npm install -g pnpm`)
- Go 1.22+
- Docker Desktop (for local stack and image builds)
- JDK 21 (for Spring Boot)

## First-Time Setup

```bash
# Clone the repo
git clone <repo-url>
cd apps

# Install all dependencies
pnpm install

# Verify Nx is working
pnpm exec nx show projects

# Start local Docker stack (PostgreSQL, etc.)
docker compose -f scripts/docker/compose.yaml up -d

# Run lint + tests on everything (may take a few minutes)
pnpm exec nx run-many -t lint test
```

## For AI Agents Starting a New Session

1. **Read `CLAUDE.md`** (or `AGENT.md`) — covers all commands, conventions, and do-not-dos
2. **Read `docs/ai/architecture.md`** — understand the service map and project graph
3. **Read `docs/ai/conventions.md`** — coding standards before writing any code
4. **Check branch status:** `git status && git log --oneline -5`
5. **Run affected targets** before starting work: `pnpm exec nx affected -t lint test`

## Key Files to Know

| File                          | Purpose                                                    |
| ----------------------------- | ---------------------------------------------------------- |
| `nx.json`                     | Nx workspace config (plugins, target defaults, generators) |
| `.eslintrc.json`              | Root ESLint config including module boundary rules         |
| `.commitlintrc.json`          | Conventional commit rules                                  |
| `package.json`                | Deps, scripts, commitizen + lint-staged config             |
| `.husky/commit-msg`           | Runs commitlint on every commit                            |
| `.husky/pre-commit`           | Runs lint-staged on staged files                           |
| `.github/workflows/ci.yml`    | CI pipeline definition                                     |
| `scripts/docker/compose.yaml` | Local development stack                                    |

## Project Name Conventions

Nx project names follow the directory path pattern:

- Apps: short name (e.g., `container`, `authui`)
- Libs: hyphenated path (e.g., `angular-container-feature-core`, `angular-shared-ui`)

Run `pnpm exec nx show projects` to see all project names.
