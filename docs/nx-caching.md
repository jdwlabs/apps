# Nx Caching (Self-Hosted — No Nx Cloud)

This repo deliberately runs **zero Nx Cloud**: no `nx-cloud` package, no
`nxCloudAccessToken`, and `neverConnectToCloud: true` in `nx.json` suppresses
every `nx connect` prompt. Caching is self-hosted at these levels:

| Level                           | Mechanism                                         | Status                    |
| ------------------------------- | ------------------------------------------------- | ------------------------- |
| Shared local cache (worktrees)  | `.nx/workspace-data` db at the main worktree root | Always on                 |
| CI cache                        | none — every task runs cold                       | Removed, see below        |
| LAN remote cache (MinIO-backed) | Nx OpenAPI cache server                           | **Not built** — see below |

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
- The same trap arrives from **plugin-inferred** targets, where no entry in this
  repo declares the bad `inputs` at all. `@nx-go/nx-go` applies its flat
  `GO_PROJECT_INPUTS` list — project-scoped globs with no `^` entry — to every
  target it infers, so dependency sources never reach the hash. Auditing
  `nx.json` alone will not show it; the offending config exists only in the
  resolved graph. `workspace-checks:test` is the standing guard for this and
  reads the graph rather than any file, so it catches both sources.
- A cacheable target that declares **no `outputs`** is a green with nothing
  behind it: the hit restores no artifact, so the cache is replaying only the
  exit code. Prefer `cache: false` for such targets, and for any target that
  writes tracked files back into the tree.
- `pnpm exec nx reset` clears the cache.
- CI does not cache Nx at all — see the CI section. The cache described here
  is local-only.
- A committed absolute `cacheDirectory` would not survive CI anyway:
  `path.posix.isAbsolute('F:/Dev/.nx-cache')` is `false`, so a Linux runner
  resolves it _relative to the workspace_ and caches nothing.

## CI cache (GitHub-hosted runners)

**There is no Nx cache in CI.** No workflow references `.nx/cache`,
`.nx/workspace-data` or `NX_CACHE_DIRECTORY`; the only `cache:` keys in
`ci.yml` are `setup-node`'s pnpm store. Every Nx task on every run executes
cold, and a cache hit is impossible by construction rather than by miss.

That is deliberate. The step was removed in `316f61b47` ("drop the unreadable
Nx cache") once it was established that the artifacts live in
`.nx/workspace-data/<uuid>-v3.db` on Nx 23 while the step persisted only
`.nx/cache` — so it was writing over a gigabyte per run of terminal output and
run metadata that Nx never read back, and evicting the pnpm, Trivy and gradle
caches that do work.

Restoring an Nx cache here is therefore a design question, not a path fix.
Persisting `.nx/workspace-data` naively would also carry the project graph and
file map, which are not safe across differing checkouts. Two properties have to
hold first: every verification target must be `cache: false`, since a cache
artifact replays a stored exit code rather than re-deriving a verdict; and
every cacheable target must declare its own filesets, or the hash cannot move
when the code moves. Until both hold, a CI cache converts gates into replays.

Note what the absent cache has been masking: the inputs defects described above
never reached production precisely because CI's hit rate is zero. Fixing the
cache without fixing the inputs first would remove that accidental protection.

The homelab MinIO (`https://192.168.1.205:9000`) is a private LAN address.
GitHub-hosted runners cannot reach it, so a MinIO-backed remote cache
currently applies only to machines on the LAN (dev boxes, any future
self-hosted/ARC runners). Reachability could be bought with Tailscale on the
runner (`tailscale/github-action` + an auth-key secret) or a public TLS
endpoint in front of the cache server — but neither would produce a single
cache hit, for the reason below.

### A Windows checkout and a Linux runner cannot share a task hash

**Decision: no remote cache for GitHub-hosted runners. This is not a
reachability problem and buying reachability does not fix it.**

`.gitattributes` pins `eol=lf` for `js/ts/html/css/json/md/sh/yml/yaml` only.
Everything else falls through to `* text=auto`, which with `core.autocrlf=true`
checks out **CRLF on Windows** and LF on a Linux runner. That is 210 of 678
tracked files — every `.scss` (49), every `.go` (30), every `.java` (83), plus
the Dockerfiles and `VERSION` manifests:

```
$ git ls-files --eol | awk '{print $2}' | sort | uniq -c
    458 w/lf
    210 w/crlf
```

Nx hashes file contents as they sit on disk and performs **no EOL
normalization**. Its native hasher, on the same logical file:

```
$ node -e "const n=require('nx/dist/src/native');
           console.log(n.hashFile('lf.scss'), n.hashFile('crlf.scss'))"
15293204973246296553   102058601192357194
```

Every project in the workspace contains at least one CRLF file in its own
root, and `libs/frontend/shared` contributes 22 of them to every frontend
through `^production`. So no cacheable target anywhere in this repo can
compute the same hash on a dev box and on `ubuntu-latest` — the hit rate
across that boundary is **0% by construction**, not by tuning.

Two consequences worth keeping straight:

- **Dev↔CI reuse is dead** until the checkout is normalized. Pinning the
  remaining text types to `eol=lf` and renormalizing would lift the blocker,
  but it rewrites ~210 files across every project and has to be verified
  against the Gradle and Go builds (`gradlew` wants LF, `*.bat` wants CRLF) —
  its own change, not a line edit here.
- **LAN dev↔dev reuse is unaffected**, since those machines are all Windows
  and agree on CRLF. That is the only sharing a MinIO-backed cache could
  actually serve today, and it is also the tier that already works locally.

This also bounds the prize. Recent `ci.yml` runs sit at a median of ~3m18s,
of which the Nx-cacheable portion is a few minutes; `apps` is a public repo,
so Actions minutes are free. A permanent read-write network service is not a
proportionate trade for that, which is the other half of why this stays
unbuilt.

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
strictly opt-in and additive.

**Degradation is graceful only for connection-refused.** Verified on Nx 23.1
against an unreachable endpoint: the task runs locally, Nx logs a non-fatal
`Failed to send request` for the `/v1/cache/{hash}` call, and exits 0. A
server that _answers_ with a **5xx** does not behave that way — Nx exits `1`
with every task reported successful. That distinction decides the deployment
topology: a cache behind a gateway that returns 502 when its backend has no
ready endpoints turns every pod rollout, node drain or upgrade into red builds
for every developer and every CI job, whereas a cache that is simply down is
harmless. "Unreachable endpoint still builds" is therefore the wrong
acceptance test — it passes in the scenario that was never the risk.

## Provisioning the MinIO-backed cache server (not started, gated)

MinIO itself does not speak the Nx OpenAPI protocol, so a thin cache server
would sit between Nx and the bucket. **Nothing below is built**, and two gates
have to clear before any of it should be.

### Gate 1 — every gate target must stop being cacheable

A cache artifact stores `(hash, terminalOutput, outputs, code)`. The **exit
code travels inside the artifact** and is replayed on a hit with convincing
green output, so a target satisfiable from a shared cache is a replay and not
a gate. Under a local-only cache a bad entry costs one machine and dies at the
next `nx reset`; under a shared one, whoever can write a hash decides whether
`lint` and `test` passed for everyone.

`nx.json` today still marks the verification targets cacheable:

```
@nx/eslint:lint              cache: true
@analogjs/vitest-angular:test cache: true
@nx/vitest:test              cache: true
@nx-go/nx-go:lint            cache: true
@nx-go/nx-go:test            cache: true
```

That is correct for a local cache and unsafe for a shared one. Flipping them
to `cache: false` costs real local iteration speed, so it should happen as
part of introducing a shared cache — not speculatively before one exists.

### Gate 2 — the build→image path must not be cache-fed

`build-image` is `cache: false`, but it `dependsOn: ["build", "^build"]` and
`build` is cacheable. `apps` is a **public** repo, so task hashes are
computable by anyone. A remote **hit** on `build` therefore means the shipped
image is packaged from cache contents that were never compiled — and the
delivery path runs on to Docker Hub and ArgoCD apps with prune + selfHeal,
with no commit, PR or diff anywhere in it. Any shared cache needs `deliver` to
run `--skip-nx-cache`, and needs that property tested rather than assumed.

If both gates clear, the provisioning sketch is:

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
