# ADR: live e2e tests gate prd promotion, from outside and inside the cluster

Status: proposed. Adds a harness library and images to this repo, a workflow
and a PostSync Job to deployments, notifications and a GitHub App to platform,
and a required check on prd promotion PRs. Needs maintainer acceptance before
implementation starts.

## Problem

Nothing tests the deployed platform.

- `apps/e2e/platform-e2e` runs Playwright against local `dist/` builds with every
  backend call mocked through `page.route`. Even with `BASE_URL` set,
  `sign-in-as.ts` plants an unsigned `alg:none` JWT and mocks `/api/users/42`.
- The `dispatch-e2e` job in `ci.yml` sends `apps-deployed`, and no workflow
  receives it.
- The deployments `e2e.yml` workflow is `workflow_dispatch`-only on the dormant
  ARC runner, which failed every run it attempted: the runner lost
  communication at about twelve minutes, consistent with Chromium running out
  of memory on 1Gi runners.
- `promote-prd.yml` has a `workflow_run: [E2E]` gate that has never fired.
- ArgoCD runs no PostSync verification; eight charts render `helm test` hooks
  that nothing invokes.
- JVM-to-Go parity is checked by a manual Python script that covers only the
  profile paths and leaves the users it creates behind.

Facts the options below rest on, observed 2026-09-29:

- `*.non.jdwlabs.com` and `*.prd.jdwlabs.com` resolve publicly and route
  router → HAProxy → gateway with no source restriction on 80/443. A
  GitHub-hosted workflow already reaches a public hostname daily.
- The router cannot hairpin, so pods cannot use public hostnames. In-cluster
  traffic has to target `platform-gateway-nginx.nginx-gateway.svc` with the
  public hostname as `Host`.
- `jdwlabs-non` and `jdwlabs-prd` are default-deny, and `jdwlabs-prd` has no
  external egress. prd has roughly 1060m CPU of request headroom.
- ArgoCD v3.5.3: every Application auto-syncs with prune and selfHeal, none
  sets `retry`, and notifications are disabled. A failed hook marks the sync
  failed but leaves the workload deployed, and auto-sync does not retry the
  same revision.
- No GitHub App in the org can write commit statuses or check runs.
- kube-state-metrics already exports `kube_job_status_failed`.

## Options considered

**Revive the ARC runner.** Keeps tests as ordinary workflows with no network
change. Rejected: its only e2e history is a 100% failure rate, it costs idle
controller and listener pods, its `_work` directory has no disk cap, and the
network reachability it was meant to solve no longer needs solving.

**GitHub-hosted runners against public URLs only.** Works today and activates
the existing `workflow_run: [E2E]` gate unchanged. Every run crosses the home
WAN and HAProxy, so a WAN outage turns the gate red, and nothing verifies the
in-cluster path the services actually use to reach each other.

**In-cluster ArgoCD PostSync Job only.** Tests the real in-cluster path right
after every sync. On its own it gates nothing: a failed hook does not block
promotion, and without notifications and a status-writing App nobody sees it.

**Kargo or Testkube.** Kargo does verify-then-promote properly but adds a
controller and UI for a two-stage pipeline. Testkube needs Postgres and object
storage, and its dashboard is now commercial. Both are heavier than the
problem.

**Both vantage points, one harness (chosen).** GitHub-hosted runners carry the
full suite and the existing promotion trigger; an in-cluster PostSync Job runs
the API gate after every sync; the prd promotion PR requires both. It costs the
most up front, and in return it catches breakage on either path and turns the
never-fired gate into a real one.

## Decision

### Project layout

One new Nx project, `apps/e2e/platform-live-e2e` (`type:e2e`, `scope:platform`,
`framework:playwright`): `src/harness/` holds the core described below and
`src/suites/` the Playwright projects that import it. It sits beside
`platform-e2e` rather than inside it because the mocked PR suite and the live
gate differ in trigger, configuration and images.

The core stays in the project until a second consumer needs it — most likely
`platform-e2e` reusing the typed client for type-checked mocks. It then moves
to `tools/testing/e2e-harness` (`type:util`, `scope:shared`,
`framework:playwright`), with the other shared test tooling. It never goes in
`libs/`, which holds product code split by runtime.

### Harness core

- **Environment profiles**: `non-public`, `prd-public`, `non-incluster`,
  `prd-incluster`. In-cluster profiles send to the gateway Service with the
  public hostname as `Host`.
- **Typed client**: `openapi-typescript` types generated from the frozen
  contracts in `usersrole/docs/contracts`, called through `openapi-fetch` with a
  small adapter onto Playwright's `APIRequestContext`, so a contract change is
  a compile error.
- **Auth**: real login producing a real JWT. Live mode never uses the
  `alg:none` helper; mocked mode keeps it.
- **Test users**:
  - Seeded and long-lived: non user, non admin, prd low-privilege synthetic.
    Credentials arrive through GitHub environment secrets or a Secret that ESO
    syncs from Vault `secret/jdwlabs/e2e/<env>`.
  - Ephemeral, non only: `e2e-<runId>-<n>`, created by fixtures and deleted
    in teardown.
  - prd never creates users.
- **Tag guard**: prd profiles run only tests tagged `@prd-safe`. The guard
  runs before any request and exits non-zero otherwise, so the rule lives in
  code rather than in convention.

### Suites

| Suite      | Kind                                           | Tagging                     | Runs                         |
| ---------- | ---------------------------------------------- | --------------------------- | ---------------------------- |
| `api-gate` | API only, no browser, ~2 min                   | `@gate`, subset `@prd-safe` | GitHub-hosted and in-cluster |
| `ui-live`  | Browser, real backend                          | —                           | GitHub-hosted only           |
| `parity`   | Same operation against JVM and Go, diffed      | —                           | On demand, non               |
| `load`     | Paced loop of `api-gate` operations, hard caps | —                           | On demand                    |

Tests tagged `@quarantine` run in the full suite and are excluded from the
gate. The existing mocked suite stays as the PR-time check.

### Images

Pinned to the exact `@playwright/test` version and dual-published to ghcr.io
and docker.io:

- `platform-e2e-api`: slim Node, no browsers, for in-cluster runs.
- `platform-e2e-ui`: `mcr.microsoft.com/playwright:<version>-noble`, for
  GitHub-hosted runs.

### Flow

1. CI releases; deliver bumps non values; `dispatch-e2e` sends
   `apps-deployed`.
2. ArgoCD syncs non. The PostSync Job waits for the new ReplicaSet to be
   ready, then runs `api-gate` in-cluster.
3. ArgoCD notifications post commit status `e2e/in-cluster` on the deployments
   revision through a dedicated GitHub App holding only `statuses:write` and
   `checks:write`.
4. The GitHub-hosted `E2E` workflow, triggered by `apps-deployed`, polls the
   services' version endpoints until non serves the expected tag (15-minute
   timeout), then runs `api-gate` and `ui-live` over public URLs. Without the
   wait it races ArgoCD and tests the previous pods.
5. `E2E` success fires `promote-prd.yml`, which opens the digest-pinned prd
   PR. The deployments ruleset requires both `E2E` and `e2e/in-cluster`.
6. A human approves and merges. ArgoCD syncs prd, the PostSync Job runs the
   `@prd-safe` subset, and failure sets a red status and alerts.
7. Rollback stays a values revert PR, as in the cutover runbook.

### Failure handling

- Playwright retries each gate test once. ArgoCD sync `retry` (limit 2, with
  backoff) covers auto-sync's refusal to retry a failed revision.
- The PostSync Job sets `activeDeadlineSeconds: 600`, `backoffLimit: 0`, and
  `hook-delete-policy: BeforeHookCreation`, so a failed Job stays inspectable.
- A NetworkPolicy on the Job allows egress only to the gateway Service and DNS.
- `E2EGateFailed` fires from `kube_job_status_failed{job_name=~"e2e-gate-.*"}`:
  warning in non, critical in prd, routed to Discord and the ai-sre relay.
- Evidence: junit and json reporters, traces on failure. GitHub runs keep them
  as artifacts; in-cluster runs log to Loki under the `jdwlabs` tenant.

### Testing the harness itself

- vitest unit tests for the tag guard, the fetch adapter, profile resolution
  and the version wait.
- CI regenerates contract types and fails on drift.
- Every phase that adds a gate proves it can fail: a deliberately failing
  canary must block promotion, then is reverted. A gate never seen red is not
  trusted.
- In-cluster changes are verified live — Synced/Healthy, Job logs in Loki —
  never inferred from a green PR.

### Phases

0. Human-only prerequisites: seeded accounts created and stored in Vault from
   a human terminal; the status App created and its key stored in Vault;
   `non` and `prd` GitHub environments holding the account secrets.
1. Core and `api-gate`: the `platform-live-e2e` project, typed client, auth, fixtures, tag guard, all
   33 operations smoke-covered, `platform-e2e-api` image.
2. GitHub-hosted `E2E` workflow: dispatch receiver, version wait, `api-gate`
   against public non; activates the existing promotion trigger; stale ARC
   references removed.
3. `ui-live`: real-backend browser flows and the `platform-e2e-ui` image.
4. In-cluster gate: shared PostSync Job template, readiness wait,
   NetworkPolicy, ESO credentials, ArgoCD retry, non then prd, alert rule.
5. Reporting and required checks: notifications enabled, commit status through
   the new App, ruleset requiring both checks, canary proving it.
6. Parity, load and cleanup: `parity` across all 33 operations, `load` mode,
   the Python parity script retired once the usersrole cutover completes, and
   a sweeper CronJob in non removing `e2e-*` and leftover `parity-*` users
   older than 24 hours.

Phase 2 is the first to deliver a working gate. Phases 3, 4 and 6 can proceed
in parallel once phase 1 lands, subject to their prerequisites.

## Consequences

- Promotion to prd stops being a manual judgement and becomes two required
  checks; a broken WAN or a broken in-cluster path both block it.
- Three credentials become long-lived operational state (seeded accounts and
  the status App key), all in Vault, created outside agent sessions.
- ArgoCD gains notifications and sync retry across the Applications that adopt
  the Job, which changes how a failed sync presents.
- prd carries one synthetic account whose traffic analytics and alerts must
  exclude.
- The mocked suite remains, so a PR can still pass while the deployed system
  fails; the live gate is what closes that gap, not a replacement for it.

Open questions, resolved in the phase named:

- Each service must expose its build tag for the version wait; any that does
  not gains one in phase 1.
- How prd excludes the synthetic account (naming convention or a role flag) is
  decided in phase 0.
