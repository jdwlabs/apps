# ai-sre-relay

Receives Alertmanager webhooks, drives a Holmes investigation, and fans the
result out to Discord, Jira and (behind an allowlist) a remediation pull
request. Design decisions that needed argument live in `docs/adr/` at the repo
root; this file is the operator-facing description of what the relay does with
a ticket over the life of an alert.

---

## Ticket lifecycle

The relay owns one Jira ticket per alert fingerprint and now owns its close as
well as its open. States below are per fingerprint, tracked in process memory.

| Event                                          | What the relay does                                                                            |
| ---------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| `firing`, fingerprint unknown                  | Investigates, upserts a ticket, remembers `fingerprint → issue`                                |
| `firing`, same firing episode, ticket open     | Skips the investigation, comments `Still firing — repeat notification N`                       |
| `firing`, same firing episode, ticket Done     | Forgets the mapping and investigates afresh; the upsert reopens the Done ticket                |
| `firing` while a close is pending              | Cancels the pending close before investigating                                                 |
| `resolved`, fingerprint known                  | Comments the resolve time, posts a Discord notice, starts the grace window                     |
| `resolved`, fingerprint untracked              | Recovers the ticket by its `amfp-<fingerprint>` label and adopts it, then as above             |
| `resolved`, no open ticket for the fingerprint | No-op — a resolve never opens a ticket                                                         |
| Grace window elapsed, ticket still open        | Transitions the ticket to Done, then comments the resolve time; `tickets_auto_closed_total` +1 |
| Transition landed, closing comment did not     | Counted as closed and logged at Error; never retried, since the ticket is already Done         |
| Grace window elapsed, ticket closed by a human | Drops the pending close, comments nothing                                                      |
| `firing` after the relay closed the ticket     | Investigates and reopens the same ticket rather than filing a duplicate                        |
| `firing` that lost the race with a close       | The close is undone: the ticket is reopened with the reason on it                              |

Two properties are worth stating explicitly, because losing either one is what
made tickets accumulate before:

- **Repeat suppression keys on the fingerprint _and_ the ticket status.** A Done
  ticket never absorbs a firing alert as a repeat, however it was closed — by a
  human, or by the relay itself.
- **A close is always cancellable.** The relay schedules no timers. It records
  a resolve time and re-evaluates "has this stayed resolved long enough" on each
  resolve notification and on a periodic sweep, so a re-fire at any point before
  the window elapses simply erases the resolve time. The resolve time is
  re-read immediately before the close goes out, and a re-fire that still beats
  it — landing while the transition itself is in flight — is repaired by
  reopening the ticket rather than left as a Done ticket for a firing alert.

### Why the close is deferred at all

An alert that goes quiet has not necessarily been fixed — a failing job that
stops being scheduled, a scrape gap, a flapping probe all resolve. The grace
window is the interval in which a condition that was only sleeping wakes up and
cancels its own close. Six hours covers the observed re-fire intervals of the
alerts that mattered (job-freshness rules re-evaluate hourly) without leaving
fixed conditions sitting in an open ticket for days.

### Restart behaviour

Fingerprint state is in-process only, by choice: the relay is a 32Mi replica
with no datastore. The ticket outlives the process, though, and both directions
recover from it — a firing alert finds its ticket through the `amfp-` label
search in the upsert, and a resolve for a fingerprint this process never saw
does the same lookup before giving up. Neither direction is stranded by a
restart.

What a restart does lose is a close already _pending_: the resolve time that
started the grace window lives only in memory, and Alertmanager sends a
resolved notification once rather than repeating it. If the relay restarts
between the resolve and the end of the grace window, that ticket waits for a
human. This is the benign failure — the pre-existing behaviour, a ticket that
stays open — rather than a persisted timer firing against state that no longer
holds, which is why the state was left in memory.

One rough edge sits here: a resolve's Jira work runs on a 60s budget of its
own, detached from the per-alert deadline, while shutdown drains for 25s. A
rolling update landing in that gap can cut a close between its transition and
its comment — a Done ticket with no explanation on it. The next sweep will not
retry it, because the ticket is already Done.

---

## Configuration

Environment variables; every one below has a working default.

| Variable                        | Default            | Meaning                                                                     |
| ------------------------------- | ------------------ | --------------------------------------------------------------------------- |
| `RESOLVED_CLOSE_GRACE`          | `6h`               | How long an alert must stay resolved before its ticket is closed            |
| `RESOLVED_SWEEP_INTERVAL`       | `10m`              | How often pending closes are re-evaluated, independent of incoming webhooks |
| `JIRA_DONE_STATUS`              | `Done`             | Status name an automatic close aims for before falling back to the category |
| `INVESTIGATION_TIMEOUT_SECONDS` | `240`              | Per-alert deadline for the whole pipeline                                   |
| `MAX_CONCURRENT`                | `4`                | Investigation workers                                                       |
| `QUEUE_SIZE`                    | `2×MAX_CONCURRENT` | Accepted-but-not-started depth; overflow answers 503                        |

`JIRA_DONE_STATUS` exists because "Won't Do" and "Duplicate" usually sit in
the Done category too, and closing an incident as a duplicate says something
the relay does not mean. The named status wins; the category is the fallback
for workflows that call it something else.

Both duration variables take Go duration syntax (`6h`, `90m`). An unparseable
or non-positive value is ignored with a warning and the default is used, so a
typo cannot silently turn auto-close into never-close.

### Alertmanager receiver requirement

**The relay's webhook receiver must set `send_resolved: true`.** Nothing else
tells the relay a condition cleared: with it off, every ticket the relay opens
stays open until a human closes it, and the auto-close path is dead code. The
`ai-sre` receiver lives in the platform repo at
`tenants/platform/services/kube-prometheus-stack/postInstall/alertmanager-config-externalsecret.yaml`.
Until that flag is on, everything below this line is inert.

---

## Metrics

Prometheus text exposition on `GET /metrics`, unauthenticated.

| Metric                                      | Read it as                                                                      |
| ------------------------------------------- | ------------------------------------------------------------------------------- |
| `ai_sre_relay_investigations_run_total`     | Alerts investigated                                                             |
| `ai_sre_relay_repeats_skipped_total`        | Repeat notifications absorbed by an open ticket                                 |
| `ai_sre_relay_tickets_auto_closed_total`    | Tickets the relay transitioned to Done after their alert stayed resolved        |
| `ai_sre_relay_repo_rejections_total`        | Remediations dropped at the repository allowlist                                |
| `ai_sre_relay_path_rejections_total`        | Remediations dropped at the path allowlist                                      |
| `ai_sre_relay_branches_skipped_total`       | Remediations dropped because the ticket's PR branch existed                     |
| `ai_sre_relay_branches_orphaned_total`      | Branches found with no pull request — a previous run failed mid-way             |
| `ai_sre_relay_ticket_reopens_total`         | Closes that raced a re-fire and were undone, by `result` — `failed` is terminal |
| `ai_sre_relay_fingerprints_untracked_total` | Alerts past the tracking ceiling; each loses repeat suppression and auto-close  |
| `ai_sre_relay_alerts_rejected_total`        | Refusal responses (503) — retry pressure, not distinct alerts                   |

---

## Runbook: a ticket is not closing

Work down this list; each step distinguishes two states that look identical
from the Jira board alone.

1. **`ai_sre_relay_tickets_auto_closed_total` is flat while investigations keep
   landing.** The relay is not being told anything resolved. Check
   `send_resolved` on the `ai-sre` receiver first — it is the only dependency
   outside this service.
2. **Logs show `alert resolved; ticket close pending` but never
   `ticket closed`.** The grace window has not elapsed (the line carries
   `resolved_at` and `grace`), or the sweep is failing. A failing sweep logs
   at Error — `pending close failed` or `ticket state unreadable` — naming
   the issue and the Jira error, and retries on the next pass.
3. **Logs show `alert re-fired inside the close grace period`.** Working as
   intended: the condition came back. The ticket is the right place for it.
4. **Logs show `pending close dropped: ticket is already closed`.** A human got
   there first; the relay left the ticket alone.
5. **Nothing at all for the fingerprint.** The relay restarted between the
   resolve notification and the end of the grace window, and Alertmanager does
   not repeat a resolve. Close the ticket by hand. A resolve arriving _after_
   a restart is fine — the relay looks the ticket up by its `amfp-` label and
   adopts it, logging `resolve: adopted an existing ticket`.

The relay's Jira workflow transitions are resolved by target **status
category**, never by transition id — both the close and the reopen path pick
the first transition whose target sits on the wanted side of Done. A workflow
with no reachable Done transition surfaces as a `no Done transition available`
error and the close is retried on the next sweep.

---

## Development

```bash
npx nx run ai-sre-relay:build     # compile
npx nx run ai-sre-relay:test      # unit tests
npx nx run ai-sre-relay:lint      # vet
```

Or directly, from `apps/backend/ai-sre-relay`: `go build ./...`,
`go test ./... -race`. The Nx `test` target sets `race: true`, so the detector
runs in CI as well as locally — the sweep runs on its own goroutine and shares
the tracking map with the dispatcher workers.
