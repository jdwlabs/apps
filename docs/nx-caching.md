# Nx Caching (Self-Hosted — No Nx Cloud)

This repo deliberately runs **zero Nx Cloud**: no `nx-cloud` package, no
`nxCloudAccessToken`, and `neverConnectToCloud: true` in `nx.json` suppresses
every `nx connect` prompt. Caching is self-hosted at these levels:

| Level                           | Mechanism                                         | Status              |
| ------------------------------- | ------------------------------------------------- | ------------------- |
| Shared local cache (worktrees)  | `.nx/workspace-data` db at the main worktree root | Always on           |
| CI cache                        | `actions/cache` on `.nx/cache` (branch-scoped)    | Always on in CI     |
| LAN remote cache (MinIO-backed) | Nx OpenAPI cache server                           | Planned — see below |

## Shared local cache (dev machines, worktrees)

Worktrees already share one cache. Nx resolves the cache root through
`getMainWorktreeRoot()`, so a run started from any linked worktree reads and
writes the main checkout's `.nx` rather than a worktree-local copy — verified
on Nx 23.1 by running in a fresh worktree with no environment variables set
and watching the main checkout's cache grow.

**Do not set `NX_CACHE_DIRECTORY`.** It overrides the above and, because it
only applies in shells that happen to export it, produces two caches: one at
the custom path and one at the default. Whichever a given shell does not use
looks cold, which reads as "worktrees aren't sharing" when in fact the cache
was simply split. That misdiagnosis is what this section exists to prevent.

### The cache directory is not where the artifacts are

On Nx 23 the cache directory holds `run.json` and `terminalOutputs/`. The task
artifacts live in a SQLite database at `.nx/workspace-data/<uuid>-v3.db`.
Deleting the cache directory therefore does **not** produce a cache miss:
observed on 23.1 by removing the directory entirely and re-running a build,
which reported `Cache: 1/1 hit (100%)` and restored its outputs from the
database while the directory contained no hash entry at all.

Two consequences worth knowing before debugging a cache:

- `pnpm exec nx reset` is the only reliable way to force a cold run. Clearing
  the directory by hand looks like it worked and does not.
- The database is held open by the Nx daemon. On Windows a second checkout's
  daemon keeps a lock, so deleting it fails with `Device or resource busy`
  until that daemon exits (`NX_DAEMON=false` avoids taking the lock).

Notes:

- Cache entries are keyed by content hash, so sharing across branches,
  worktrees, and even repos is collision-safe — but only for the inputs a
  target actually declares. A stale entry _can_ be served as a wrong one when
  a target's `inputs` omit its own sources: the hash then cannot move when the
  code moves. The module-federation build targets hit exactly that, hashing
  two env vars and no fileset at all, so an edited `styles.scss` returned a
  100% hit and re-emitted the previous theme byte-for-byte. When adding
  `inputs` to a `targetDefaults` entry, remember it **replaces** the implicit
  `["default", "^default"]` rather than extending it — always restate the
  filesets alongside whatever env or runtime input prompted the override.
- `pnpm exec nx reset` clears the cache.
- CI keeps the same default `.nx/cache` path, which the `actions/cache` step
  in `ci.yml` persists across runs — nothing environment-specific to set.
  Whether persisting that path is still sufficient on Nx 23 is an open
  question; see the CI section.
- A committed absolute `cacheDirectory` would not survive CI anyway:
  `path.posix.isAbsolute('F:/Dev/.nx-cache')` is `false`, so a Linux runner
  resolves it _relative to the workspace_ and caches nothing.

## CI cache (GitHub-hosted runners)

CI restores/saves `.nx/cache` via `actions/cache` (see `ci.yml`). GitHub
scopes cache entries per branch, so a PR cannot poison main's cache — this is
the same class of protection the deprecated Nx S3 plugins lacked
(CVE-2025-36852, "CREEP"). This _is_ the self-hosted CI cache; it needs no
credentials and degrades to a plain build on a cache miss.

**Open question: whether that path still restores anything.** The cached path
is `.nx/cache`, but the section above establishes that on Nx 23 the artifacts
are in `.nx/workspace-data/<uuid>-v3.db`, which the step does not cache. If
that holds on a Linux runner too, CI is persisting terminal outputs and run
metadata while re-running every task — which would match the 0% cache-hit
rates seen on local cold runs. This has **not** been confirmed against a real
CI run; confirm before changing the path, because adding `.nx/workspace-data`
naively would also cache the project graph and file map, which are not safe to
carry across differing checkouts.

The homelab MinIO (`https://192.168.1.205:9000`) is a private LAN address.
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
  licensing and security grounds. Confirmed on the registry: `@nx/s3-cache`
  and `@nx/powerpack-s3-cache` are both `5.0.7`, both carry a `deprecated`
  notice, and both are published under a **`Commercial`** license. There is
  no `@nx/s3` package at all — a plain-`s3` name is worth double-checking
  before anyone plans work around it.
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

Both names are present in the Nx 23.1 native binary, alongside the legacy
alias `NX_REMOTE_CACHE_URL`. The wire contract, captured against a stub server
implementing the spec:

```
GET /v1/cache/<hash>   Authorization: Bearer <token>   -> 404 on miss
PUT /v1/cache/<hash>   Authorization: Bearer <token>   -> 202, body = artifact
```

A miss is a `404` followed by a `PUT` of a few kilobytes once the task
finishes; nothing else is exchanged.

**The client's TLS does not go through Node.** `getHttpCache()` returns a
native `HttpRemoteCache` and the request runs in Rust (reqwest), which Nx's
own source notes "honors `NODE_TLS_REJECT_UNAUTHORIZED` but bypasses Node's
TLS stack". `NODE_EXTRA_CA_CERTS` will therefore **not** make the client trust
an internal CA. Plan for either a publicly-trusted certificate on the cache
server or a CA that the OS trust store already carries — turning verification
off wholesale is not an acceptable substitute on a read-write cache.

If the variables are unset, Nx uses local cache only — remote caching is
strictly opt-in and additive. Degradation is graceful (verified on Nx 23.1
against an unreachable endpoint): the task runs locally, Nx logs a non-fatal
`Failed to send request` for the `/v1/cache/{hash}` call, and exits 0.

## Provisioning the MinIO-backed cache server (follow-up)

MinIO itself does not speak the Nx OpenAPI protocol, so a thin cache server
sits between Nx and the bucket. Plan (lands in the platform/infrastructure
repos, not here):

1. Create a `nx-cache` bucket on the MinIO instance with a dedicated scoped
   access key — do not reuse the Terraform-state credentials. MinIO serves
   TLS on `https://192.168.1.205:9000` since the state-backend cutover; the
   plaintext port still answers, so pick the endpoint deliberately rather
   than inheriting whichever one an older note mentions.
2. Deploy an open-source server implementing the Nx cache OpenAPI spec with
   S3 storage, pointed at that bucket. `nxcite/nx-cache-server` is the only
   viable candidate: Rust, Apache-2.0, actively pushed, and it takes
   `S3_ENDPOINT_URL` so it speaks to MinIO directly. `IKatsuba/nx-cache-server`
   publishes **no license**, which makes it un-reusable regardless of merit —
   it should not be re-proposed. Note that neither ships a container image or
   a Dockerfile; releases are bare binaries, so running it on the cluster
   means building an image first. A TrueNAS app or a systemd unit on the NAS
   avoids that.
3. Accept that a single bearer token is what the server offers.
   `nxcite/nx-cache-server` takes one `SERVICE_ACCESS_TOKEN`, so the
   read-only/read-write split this plan previously assumed is not available,
   and the CREEP exposure that disqualified the deprecated S3 plugins is
   **not** mitigated by switching to this server. What the switch buys is a
   maintained, Apache-2.0, no-activation-key implementation. Keep the token
   to trusted LAN actors, and treat "untrusted input writes to this cache" as
   the risk that is still open.
4. Point LAN machines at it via the two env vars above.

Secrets stay out of this repo: tokens live in each machine's environment (and
GitHub Actions secrets if hosted runners ever get LAN access).
