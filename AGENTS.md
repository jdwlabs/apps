# AGENTS.md

Canonical instructions for AI agents in this repo. `CLAUDE.md`, `GEMINI.md` and
`AGENT.md` only import or point here — edit this file.

## Repo facts

- Nx monorepo: Angular micro-frontends (Module Federation; `container` is the
  shell, `authui`/`rolesui`/`usersui` are remotes), Go services, one Spring Boot
  service (`usersrole`), PostgreSQL migration runners. `pnpm exec nx show projects`
  lists everything.
- pnpm only (`pnpm-lock.yaml`); never `npm install` or `yarn`.
- `usersrole` is being replaced by `identity-service` (`/auth`, `/api/users`,
  `/api/roles`) and `profile-service` (`/api/profiles`). Its frozen API contracts
  stay in `apps/backend/usersrole/docs/contracts` — change them only deliberately.
- `tools/agents/` is the Docker dev-agent image; do not move or restructure it.
- CI runs `nx affected`, so a change is only as isolated as the project graph says
  (`pnpm exec nx graph`).

## Commands

```bash
pnpm exec nx affected -t lint test        # during development
pnpm exec nx format:check                 # CI gate; fix with format:write
pnpm exec nx run <project>:<target>
pnpm exec nx reset                        # required after editing project.json targets
pnpm run commit                           # Commitizen prompt
docker compose -f scripts/docker/compose.yaml up -d   # local stack
```

Go: run from the repo root, and name module roots. The root is a `go.work`
workspace, not a module, so `./...` fails there (`directory prefix . does not
contain modules listed in go.work`). `go.work` is also where CI reads the Go
version from.

```bash
modules=$(sed -n '/^use (/,/^)/p' go.work | grep -oE '\./\S+' | sed 's|$|/...|')
go build $modules
go test $modules
```

## Nx cache

Do not set `NX_CACHE_DIRECTORY` — every worktree already shares the main
worktree's `.nx/cache`, and setting it in some shells splits the cache.
Never accept an `nx connect` / Nx Cloud prompt; the cache is self-hosted only.
Details: [docs/nx-caching.md](docs/nx-caching.md).

## Projects and boundaries

Every `project.json` needs a `type:`, `scope:` and `framework:` tag. Lint
(`eslint.config.ts`) enforces the layering, per-app scope isolation and
no cross-framework imports; share code between Angular apps through
`scope:shared` libs, never app-to-app. Rules and the grandfathered exceptions:
[docs/conventions.md](docs/conventions.md#module-boundaries).

## Commits

Conventional Commits, enforced by commitlint (`.commitlintrc.json`) in the
`commit-msg` hook and in CI. Do not bypass hooks with `--no-verify`; the
pre-commit hook runs lint-staged. Footers and trailers: [CONTRIBUTING.md](CONTRIBUTING.md).

## Releases

The git tag (`{projectName}-{version}`) is the only record of a version: no
manifest carries one, `nx release` never commits, and GitHub Releases are the
changelog. A new project needs an initial tag. Releases run only in CI on push
to `main` — never run `nx release` locally without `--dry-run`, because tags
are irreversible. Details: [docs/workflows.md](docs/workflows.md#releasing-a-new-version).

## CI and rulesets

- Required checks and review rules live in [`.github/rulesets/`](.github/rulesets/)
  and are applied by hand with `apply.sh` after merge. Read its header before
  renaming, merging or removing a CI job — the wrong order leaves PRs
  permanently unmergeable.
- Ask before changing `.github/workflows/ci.yml`; its `build-image` steps rely on
  a docker-container BuildKit builder with QEMU multi-arch.
- Ask before `docker buildx build --push` (publishes to Docker Hub).
- Secrets come from the environment at deploy time; never hardcode one.

## Working with other agents

Several agents push to this repo concurrently. Re-fetch `origin/main` right
before a rebase or push instead of trusting the worktree's view of it.

Ticket evidence older than about a week is a hypothesis: re-check its premises
against live state, state the scope you searched before claiming something is
absent, and record a disproved premise on the ticket.

## Tool output traps

RTK-filtered output (`rtk go build`, `rtk gh pr view`) can report success or a
stale state — use `rtk proxy <cmd>` before acting on a result. This and the
Windows-checkout traps (pnpm across drives, curl exit codes, CRLF) are in
[docs/agent-tooling-traps.md](docs/agent-tooling-traps.md).
