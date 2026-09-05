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

| Event                                          | What the relay does                                                                          |
| ---------------------------------------------- | -------------------------------------------------------------------------------------------- |
| `firing`, fingerprint unknown                  | Investigates, upserts a ticket, remembers `fingerprint → issue`                              |
| `firing`, same firing episode, ticket open     | Skips the investigation, comments `Still firing — repeat notification N`                     |
| `firing`, same firing episode, ticket Done     | Forgets the mapping and investigates afresh; the upsert reopens the Done ticket              |
| `firing` while a close is pending              | Cancels the pending close before investigating                                               |
| `resolved`, fingerprint known                  | Comments the resolve time, starts the grace window                                           |
| `resolved`, fingerprint unknown                | No-op (the relay restarted, or never filed a ticket for it)                                  |
| Grace window elapsed, ticket still open        | Comments the resolve time and transitions the ticket to Done, `tickets_auto_closed_total` +1 |
| Grace window elapsed, ticket closed by a human | Drops the pending close, comments nothing                                                    |
| `firing` after the relay closed the ticket     | Investigates and reopens the same ticket rather than filing a duplicate                      |

Two properties are worth stating explicitly, because losing either one is what
made tickets accumulate before:

- **Repeat suppression keys on the fingerprint _and_ the ticket status.** A Done
  ticket never absorbs a firing alert as a repeat, however it was closed — by a
  human, or by the relay itself.
- **A close is always cancellable.** The relay schedules no timers. It records
  a resolve time and re-evaluates "has this stayed resolved long enough" on each
  resolve notification and on a periodic sweep, so a re-fire at any point before
  the window elapses simply erases the resolve time.

### Why the close is deferred at all

An alert that goes quiet has not necessarily been fixed — a failing job that
stops being scheduled, a scrape gap, a flapping probe all resolve. The grace
window is the interval in which a condition that was only sleeping wakes up and
cancels its own close. Six hours covers the observed re-fire intervals of the
alerts that mattered (job-freshness rules re-evaluate hourly) without leaving
fixed conditions sitting in an open ticket for days.

### Restart behaviour

Fingerprint state is in-process only, by choice: the relay is a 32Mi replica
with no datastore. A restart therefore drops every pending close, and pending
closes are the only state whose loss is silent. The failure mode this buys is
the benign one — a ticket stays open until a human closes it, exactly as before
this feature existed — rather than a persisted timer firing against state that
no longer holds. The next resolve notification for a still-tracked alert
re-arms the close.

---

## Configuration

Environment variables; every one below has a working default.

| Variable                        | Default            | Meaning                                                                     |
| ------------------------------- | ------------------ | --------------------------------------------------------------------------- |
| `RESOLVED_CLOSE_GRACE`          | `6h`               | How long an alert must stay resolved before its ticket is closed            |
| `RESOLVED_SWEEP_INTERVAL`       | `10m`              | How often pending closes are re-evaluated, independent of incoming webhooks |
| `INVESTIGATION_TIMEOUT_SECONDS` | `240`              | Per-alert deadline for the whole pipeline                                   |
| `MAX_CONCURRENT`                | `4`                | Investigation workers                                                       |
| `QUEUE_SIZE`                    | `2×MAX_CONCURRENT` | Accepted-but-not-started depth; overflow answers 503                        |

Both duration variables take Go duration syntax (`6h`, `90m`). An unparseable
or non-positive value is ignored with a warning and the default is used, so a
typo cannot silently turn auto-close into never-close.

### Alertmanager receiver requirement

**The relay's webhook receiver must set `send_resolved: true`.** Nothing else
tells the relay a condition cleared: with it off, every ticket the relay opens
stays open until a human closes it, and the auto-close path is dead code. The
`ai-sre` receiver lives in the platform repo at
`tenants/platform/services/kube-prometheus-stack/postInstall/alertmanager-config-externalsecret.yaml`.

---

## Metrics

Prometheus text exposition on `GET /metrics`, unauthenticated.

| Metric                                   | Read it as                                                               |
| ---------------------------------------- | ------------------------------------------------------------------------ |
| `ai_sre_relay_investigations_run_total`  | Alerts investigated                                                      |
| `ai_sre_relay_repeats_skipped_total`     | Repeat notifications absorbed by an open ticket                          |
| `ai_sre_relay_tickets_auto_closed_total` | Tickets the relay transitioned to Done after their alert stayed resolved |
| `ai_sre_relay_repo_rejections_total`     | Remediations dropped at the repository allowlist                         |
| `ai_sre_relay_path_rejections_total`     | Remediations dropped at the path allowlist                               |
| `ai_sre_relay_branches_skipped_total`    | Remediations dropped because the ticket's PR branch existed              |
| `ai_sre_relay_branches_orphaned_total`   | Branches found with no pull request — a previous run failed mid-way      |
| `ai_sre_relay_alerts_rejected_total`     | Refusal responses (503) — retry pressure, not distinct alerts            |

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
5. **Nothing at all for the fingerprint.** The relay restarted after the
   resolve notification arrived, so the pending close was lost. Close the
   ticket by hand; the next resolve for that alert re-arms the mechanism.

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
`go test ./... -race`.
