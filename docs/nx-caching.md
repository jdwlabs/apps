# Nx Caching (Self-Hosted — No Nx Cloud)

This repo deliberately runs **zero Nx Cloud**: no `nx-cloud` package, no
`nxCloudAccessToken`, and `neverConnectToCloud: true` in `nx.json` suppresses
every `nx connect` prompt. Caching is self-hosted at these levels:

| Level                           | Mechanism                                      | Status              |
| ------------------------------- | ---------------------------------------------- | ------------------- |
| Shared local cache (worktrees)  | `.nx/cache` at the main worktree root          | Always on           |
| CI cache                        | `actions/cache` on `.nx/cache` (branch-scoped) | Always on in CI     |
| LAN remote cache (MinIO-backed) | Nx OpenAPI cache server                        | Planned — see below |

## Shared local cache (dev machines, worktrees)

Worktrees already share one cache. Nx resolves `cacheDir` through
`getMainWorktreeRoot()`, so a run started from any linked worktree reads and
writes `<main-checkout>/.nx/cache` rather than a worktree-local copy —
verified on Nx 23.1 by running in a fresh worktree with no environment
variables set and watching the main checkout's cache grow.

**Do not set `NX_CACHE_DIRECTORY`.** It overrides the above and, because it
only applies in shells that happen to export it, produces two caches: one at
the custom path and one at the default. Whichever a given shell does not use
looks cold, which reads as "worktrees aren't sharing" when in fact the cache
was simply split. That misdiagnosis is what this section exists to prevent.

Notes:

- Cache entries are keyed by content hash (sources, deps, env inputs), so
  sharing across branches, worktrees, and even repos is collision-safe. A
  stale entry can never be "wrong", only unused.
- `pnpm exec nx reset` clears the cache.
- CI keeps the same default `.nx/cache` path, which the `actions/cache` step
  in `ci.yml` persists across runs — nothing environment-specific to set.
- A committed absolute `cacheDirectory` would not survive CI anyway:
  `path.posix.isAbsolute('F:/Dev/.nx-cache')` is `false`, so a Linux runner
  resolves it _relative to the workspace_ and caches nothing.

## CI cache (GitHub-hosted runners)

CI restores/saves `.nx/cache` via `actions/cache` (see `ci.yml`). GitHub
scopes cache entries per branch, so a PR cannot poison main's cache — this is
the same class of protection the deprecated Nx S3 plugins lacked
(CVE-2025-36852, "CREEP"). This _is_ the self-hosted CI cache; it needs no
credentials and degrades to a plain build on a cache miss.

The homelab MinIO (`http://192.168.1.205:9000`) is a private LAN address.
GitHub-hosted runners cannot reach it, so a MinIO-backed remote cache
currently applies only to machines on the LAN (dev boxes, any future
self-hosted/ARC runners). To extend it to hosted runners later, one of:

- Tailscale on the runner (`tailscale/github-action` + an auth-key secret),
  keeping MinIO off the public internet; or
- a public TLS endpoint in front of the cache server with read-scoped tokens
  for PR builds.

Both are follow-up infrastructure work, not apps-repo changes.

## Remote cache: chosen protocol and why

Options evaluated (Nx 23):

- **Nx Cloud** — excluded by policy (previous token leak; zero dependence).
- **`@nx/s3-cache` / other official self-hosted plugins** — deprecated
  2026-05 due to CVE-2025-36852 (unpatchable-by-design cache poisoning: one
  credential grants read _and_ write to the whole cache), receive no updates,
  and require an activation key issued through nx.app. Excluded on both
  licensing and security grounds.
- **Community task-runner plugins** (`nx-remotecache-minio`,
  `nx-remotecache-custom`) — built on the custom task-runner API that was
  removed in Nx 21. Dead on Nx 23.
- **Nx self-hosted cache server (OpenAPI spec)** — supported since Nx 20.8,
  built into the `nx` package, **no activation key, no Nx Cloud contact**.
  Nx talks plain HTTP (`GET`/`PUT /v1/cache/{hash}`, bearer token) to any
  server implementing the spec. **Chosen.**

Client configuration (per machine/agent, never committed):

```bash
export NX_SELF_HOSTED_REMOTE_CACHE_SERVER=http://<cache-server>:<port>
export NX_SELF_HOSTED_REMOTE_CACHE_ACCESS_TOKEN=<token>
```

If the variables are unset, Nx uses local cache only — remote caching is
strictly opt-in and additive. Degradation is graceful (verified on Nx 23.1
against an unreachable endpoint): the task runs locally, Nx logs a non-fatal
`Failed to send request` for the `/v1/cache/{hash}` call, and exits 0.

## Provisioning the MinIO-backed cache server (follow-up)

MinIO itself does not speak the Nx OpenAPI protocol, so a thin cache server
sits between Nx and the bucket. Plan (lands in the platform/infrastructure
repos, not here):

1. Create a `nx-cache` bucket on the MinIO instance
   (`http://192.168.1.205:9000`) with a dedicated scoped access key —
   do not reuse the Terraform-state credentials.
2. Deploy an open-source server implementing the Nx cache OpenAPI spec with
   S3 storage, pointed at that bucket. Candidates (all MIT/Apache, evaluate
   at deploy time): `nxcite/nx-cache-server` (Rust),
   `IKatsuba/nx-cache-server`. Run it on the cluster or as a TrueNAS app.
3. Issue separate read-only and read-write bearer tokens: read-write for
   trusted branch builds and dev machines, read-only anywhere untrusted
   input runs — this is the CREEP mitigation the deprecated S3 plugins
   could not provide.
4. Point LAN machines at it via the two env vars above.

Secrets stay out of this repo: tokens live in each machine's environment (and
GitHub Actions secrets if hosted runners ever get LAN access).
