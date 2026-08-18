# ADR: the alert relay sheds load with backpressure, not a deeper queue

Status: proposed. Changes the relay's webhook contract (a full queue now answers
503 instead of 202) and re-derives the queue default; needs maintainer
acceptance before the image is rebuilt and rolled out.

Numbering and structure follow the convention established by `0001` in this
repo: `# ADR:` title, a `Status:` line, then Problem / Options considered /
Decision / Consequences. Numbering is per-repo.

## Problem

During a cascading cluster-networking incident, `ai-sre-relay` logged 15 dropped
investigations inside a six-second window — 14 distinct `ArgoCDAppNotSynced`
fingerprints and one `KubePodCrashLooping`. The alert burst that the system
exists to handle is also the burst that exceeded it.

Three things were wrong, and the queue's capacity was the least of them.

**The drop was acknowledged as a success.** `enqueue` returned `false` on a full
queue, and the HTTP handler discarded that result:

```go
for _, a := range payload.Alerts {
    if a.Status == "firing" {
        e.enqueue(a)
    }
}
w.WriteHeader(http.StatusAccepted)
```

The drop path logged `"investigation queue full; dropping alert (alertmanager
will retry)"`. That parenthetical was wrong. Alertmanager's `Retrier.Check`
short-circuits on any 2xx — `// 2xx responses are considered to be always
successful.` — and returns "do not retry" without reading the body. A 202 is an
irreversible acknowledgement: the notification log is written, the sender's only
copy is retired, and there is no redelivery path. The relay was not deferring
those 15 alerts; it was destroying them and reporting success.

**The drop was invisible.** Both existing counters are incremented inside
`Pipeline.Handle`, which a refused alert never reaches. An overflow moved no
metric, so no alerting rule could see it. The only trace was an ERROR line, in a
service that had previously gone two weeks without anyone reading its output.

**The queue was already far too deep to be useful.** This is the finding that
reframes the ticket. Drain rate is set by worker count over investigation
latency, not by capacity. Production runs `MAX_CONCURRENT=4` (default) with
`INVESTIGATION_TIMEOUT_SECONDS=900`, and the manifest records 433–600s observed
latency on the model in use. Time to drain a full queue:

| depth | L=433s | L=500s | L=600s | L=900s (cap) |
| ----: | -----: | -----: | -----: | -----------: |
|     4 |     7m |     8m |    10m |          15m |
|     8 |    14m |    17m |    20m |          30m |
|    16 |    29m |    33m |    40m |          60m |
|    32 |    58m |    67m |    80m |         120m |
|    64 |   115m |   133m |   160m |         240m |

At the shipped default of 64, the tail of a full queue is investigated 2–4 hours
after it fired. Alertmanager's `repeat_interval` is 4h. The bottom of that queue
is therefore reached at roughly the moment Alertmanager re-notifies everything
anyway — those slots hold work that is simultaneously stale and redundant. The
64th slot was never coverage; it was a promise the relay could not keep.

## Options considered

**Raise `QUEUE_SIZE`.** The obvious reading of the incident, and wrong. Capacity
does not change throughput, so every added slot converts a refusal the sender
would have retried into an investigation delivered later. A larger queue buys a
quieter log and worse outcomes, and an in-process buffer is unbounded-in-badness
by construction: whatever the number, a storm one alert larger overruns it, and
a pod restart erases whatever it held.

**Drop the oldest instead of the newest.** Strictly worse here. Queued alerts
were already answered 202, so the sender has discarded them; evicting one
destroys the only remaining copy. The newest alert is the only one still held
somewhere else, which makes it the only safe thing to refuse.

**Block the handler until a slot frees.** Applies backpressure, but through the
wrong mechanism — it converts queue pressure into held-open connections and
sender-side timeouts, and stalls the whole batch behind the slowest slot. It
also risks tripping the relay's own liveness probe under sustained load.

**Persist the queue.** Solves durability, but buys a datastore, a schema and a
failure mode for a problem the sender already solves. Alertmanager is a durable,
retrying, bounded sender that is already deployed.

**Refuse with a 5xx and let the sender retry.** Chosen.

## Decision

**The relay refuses work it cannot start promptly, and says so in the status
code.** When any firing alert in a batch cannot be queued, the webhook answers
`503 Service Unavailable` rather than `202 Accepted`. The remaining alerts in
the batch are still offered, so capacity that frees mid-batch is used.

The status code must be 5xx specifically. Alertmanager's webhook receiver
constructs a bare `&notify.Retrier{}` with no `RetryCodes`, and retries only
`statusCode/100 == 5`. The intuitive "too many requests" reply is a trap: 429 is
retryable for the Opsgenie, Jira, PagerDuty and incident.io receivers, but not
for webhook, where it is a permanent failure that discards the alert.

Verified retry behaviour, read from the receiver's source rather than inferred
from the log message: retries begin immediately and back off exponentially from
500ms by 1.5x to a 60s ceiling with ±50% jitter, bounded by a context deadline
of `max(group_interval, 10s)` — about 5 minutes and roughly 15 attempts at the
configured `group_interval: 5m`. If that budget is exhausted the notification
log is never updated, so a group that has not previously notified successfully
is retried at every subsequent `group_interval`, indefinitely. A group carrying
new firing alerts likewise re-sends at `group_interval`. Only an unchanged
repeat of an already-delivered group falls back to `repeat_interval`, and that
is precisely the case the relay's own dedup would skip anyway. New alert content
therefore has a durable recovery path; the previous 202 had none.

**Overflow is counted.** `ai_sre_relay_alerts_rejected_total` increments on
every refusal, and the log line drops its false promise and carries the queue
depth. A refusal is now the one thing a storm makes visible rather than the one
thing it hides.

**The queue default is re-derived as `2 * MAX_CONCURRENT`, not raised.** Its
only legitimate job is keeping workers fed across the gap between HTTP accept
and pickup; one round in flight and one round waiting does that. At the default
of 4 workers this is 8, down from 64. The point is not the digit — it is that
depth is now bounded by what the pool can start promptly, and everything beyond
it is held by Alertmanager, which is durable and retries, rather than by a
32Mi process that is neither.

Sizing beyond that derivation is deliberately not attempted. The relay exposes
no investigation-latency metric, so any specific number would be a guess
dressed as a decision; the 433–600s figure above comes from a manifest comment,
not from instrumentation. Backpressure is what makes it safe not to know.

## Consequences

- The webhook's contract changes: 503 is now an expected, healthy response
  meaning "busy, re-send", not an error. Alertmanager handles this natively.
  The drift-scan CronJob's `curl -sf --retry 3 --retry-delay 5` already treats
  5xx as retriable and fails the Job loudly if it persists — previously it was
  told 202 and reported success while its escalation was discarded.
- Refusals will be visible and, during a storm, frequent — the shallower queue
  makes them more frequent than before. That is the intended trade: a counted
  refusal that the sender retries in minutes is strictly better than a silent
  acknowledgement, and better than an investigation delivered hours late.
- `ai_sre_relay_alerts_rejected_total` is unalerted on arrival. A rule on it
  belongs with the relay's other alerting rules, which live in the platform
  repo and ship independently; a sustained non-zero rate is the signal that
  worker concurrency, not queue depth, needs raising.
- The queue-depth reduction only takes effect once the image is rebuilt and its
  digest bumped. `QUEUE_SIZE` is unset in the manifest, so the code default
  governs and no manifest env var is needed; the deployment carries a pinned
  digest, so nothing changes until that is rolled forward.
- Throughput is untouched, and remains the real ceiling. If storm coverage is
  still inadequate once refusals are visible, the lever is `MAX_CONCURRENT`
  (bounded by the 32Mi limit) or investigation latency — not capacity.
