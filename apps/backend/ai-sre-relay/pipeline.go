package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// maxTrackedFirings bounds the repeat-tracking map. Beyond it, new
// fingerprints are simply not tracked and get a full investigation — the
// memory ceiling matters more than suppressing every possible repeat on a
// 32Mi replica.
const maxTrackedFirings = 1024

type holmesInvestigator interface {
	Investigate(ctx context.Context, a Alert) (Analysis, error)
}
type patcher interface {
	Generate(ctx context.Context, an Analysis) (*Patch, error)
}
type jiraUpserter interface {
	Upsert(ctx context.Context, a Alert, an Analysis) (IssueKey, error)
	// IsOpen reports whether an issue is still actionable, so a repeat that a
	// human has already closed out is investigated afresh rather than skipped.
	IsOpen(ctx context.Context, key IssueKey) (bool, error)
	// NoteRefire records a repeat that was not investigated. A skipped repeat
	// must stay visible; silent suppression is indistinguishable from a
	// pipeline that has stopped working.
	NoteRefire(ctx context.Context, key IssueKey, count int) error
	// NoteResolved records that Alertmanager reported the alert resolved, so
	// the pending close is on the ticket before it happens.
	NoteResolved(ctx context.Context, key IssueKey, resolvedAt time.Time) error
	// Close transitions the issue to Done, naming the resolve time. A ticket
	// that reached Done but could not be commented on reports
	// ErrClosedWithoutNote, which the caller must not retry.
	Close(ctx context.Context, key IssueKey, resolvedAt time.Time) error
	// Reopen undoes a close that raced a re-fire.
	Reopen(ctx context.Context, key IssueKey) error
	// FindOpenByFingerprint recovers the open ticket already filed for this
	// alert. The relay's own fingerprint tracking does not survive a restart;
	// the ticket does.
	FindOpenByFingerprint(ctx context.Context, a Alert) (IssueKey, error)
}
type prOpener interface {
	OpenPR(ctx context.Context, p Patch, issue IssueKey) (PRLink, error)
}
type discordNotifier interface {
	Notify(ctx context.Context, a Alert, an Analysis, issue IssueKey, pr *PRLink, patch *Patch) error
}

// firing is the last completed investigation for one fingerprint. startsAt
// distinguishes a repeat notification of the same firing episode from the same
// alert firing again after it resolved: Alertmanager fingerprints hash the
// labelset, so the fingerprint alone cannot tell those apart.
type firing struct {
	startsAt string
	issue    IssueKey
	repeats  int
	// resolvedAt is zero while the alert is firing. Once set it is both the
	// pending-close clock and the time named on the ticket; a re-fire clears
	// it again.
	resolvedAt time.Time
}

// resolveHandlingTimeout bounds the Jira and Discord work one resolve
// notification does, on a context detached from the per-alert deadline.
const resolveHandlingTimeout = 60 * time.Second

// defaultCloseGrace is how long an alert must stay resolved before the relay
// closes its own ticket. Long enough that a condition which merely went quiet
// — a flapping job, a scrape gap — re-fires and cancels the close first.
const defaultCloseGrace = 6 * time.Hour

type Pipeline struct {
	holmes  holmesInvestigator
	patch   patcher
	jira    jiraUpserter
	github  prOpener
	discord discordNotifier
	log     *slog.Logger

	closeGrace time.Duration
	now        func() time.Time

	counters counters

	mu     sync.Mutex
	active map[string]firing

	// sweepMu keeps two sweeps — the ticker's and the one a resolve
	// notification triggers — from racing onto the same ticket and closing it
	// twice.
	sweepMu sync.Mutex
}

type pipelineOption func(*Pipeline)

func withCloseGrace(d time.Duration) pipelineOption {
	return func(p *Pipeline) { p.closeGrace = d }
}

func withClock(now func() time.Time) pipelineOption {
	return func(p *Pipeline) { p.now = now }
}

func NewPipeline(h holmesInvestigator, pg patcher, j jiraUpserter, gh prOpener, d discordNotifier, log *slog.Logger, opts ...pipelineOption) *Pipeline {
	p := &Pipeline{
		holmes: h, patch: pg, jira: j, github: gh, discord: d, log: log,
		closeGrace: defaultCloseGrace,
		now:        time.Now,
		active:     map[string]firing{},
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Counters exposes the pipeline's own operational signal for the metrics
// endpoint.
func (p *Pipeline) Counters() *counters { return &p.counters }

// repeatOf returns the open issue recorded for this exact firing episode, if
// this process has already investigated it.
func (p *Pipeline) repeatOf(a Alert) (firing, bool) {
	if a.Fingerprint == "" {
		return firing{}, false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	f, ok := p.active[a.Fingerprint]
	if !ok || f.startsAt != a.StartsAt || f.issue == "" {
		return firing{}, false
	}
	return f, true
}

// countRepeat increments and returns the repeat count for a skipped refire.
// It reports false when the fingerprint is no longer tracked — a sweep can
// forget one between the repeat check and here — because writing the entry
// back would insert one with no issue key, unusable and holding a slot a real
// alert then cannot have.
func (p *Pipeline) countRepeat(a Alert) (int, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	f, ok := p.active[a.Fingerprint]
	if !ok {
		return 0, false
	}
	f.repeats++
	p.active[a.Fingerprint] = f
	return f.repeats, true
}

// remember records the ticket for a finished investigation. It merges into any
// existing entry rather than replacing it: a resolve can land while the
// investigation is still running — the dispatcher tracks firing and resolved
// as separate in-flight keys precisely so it can — and overwriting would drop
// the pending close it started, silently, after the ticket had already been
// commented and announced as resolving.
func (p *Pipeline) remember(a Alert, issue IssueKey) {
	if a.Fingerprint == "" || issue == "" {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	f, known := p.active[a.Fingerprint]
	if !known {
		if len(p.active) >= maxTrackedFirings {
			p.counters.fingerprintsUntracked.Add(1)
			p.log.Warn("fingerprint tracking is full; this alert gets no repeat suppression and no auto-close",
				"fingerprint", a.Fingerprint, "alert", a.Name(), "issue", issue, "tracked", len(p.active))
			return
		}
		p.active[a.Fingerprint] = firing{startsAt: a.StartsAt, issue: issue}
		return
	}
	next := firing{startsAt: a.StartsAt, issue: issue, resolvedAt: f.resolvedAt}
	// Repeat numbering belongs to one firing episode; a new episode starts
	// again at one.
	if f.startsAt == a.StartsAt {
		next.repeats = f.repeats
	}
	p.active[a.Fingerprint] = next
}

func (p *Pipeline) forget(a Alert) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.active, a.Fingerprint)
}

// resolveTime is when the condition cleared: the alert's own end time when it
// carries a usable one, so a delayed or redelivered notification does not
// restart the grace window.
func (p *Pipeline) resolveTime(a Alert) time.Time {
	now := p.now()
	ends, err := time.Parse(time.RFC3339, a.EndsAt)
	if err != nil || ends.After(now) {
		return now
	}
	return ends
}

// markResolved starts the pending close for a fingerprint the relay already
// tracks. The bool reports whether this notification is the one that started
// it, so repeated resolve notifications do not re-comment or push the close
// further out.
func (p *Pipeline) markResolved(a Alert) (IssueKey, time.Time, bool) {
	if a.Fingerprint == "" {
		return "", time.Time{}, false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	f, ok := p.active[a.Fingerprint]
	if !ok || f.issue == "" {
		return "", time.Time{}, false
	}
	return p.startCloseLocked(a, f)
}

// adoptResolved starts the pending close for a ticket recovered from Jira
// rather than from this process's own memory. It reports an empty key when the
// tracking table is full, which is the one case where the close cannot be
// scheduled at all.
func (p *Pipeline) adoptResolved(a Alert, issue IssueKey) (IssueKey, time.Time, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	f, known := p.active[a.Fingerprint]
	if !known {
		if len(p.active) >= maxTrackedFirings {
			return "", time.Time{}, false
		}
		f = firing{startsAt: a.StartsAt, issue: issue}
	}
	return p.startCloseLocked(a, f)
}

// startCloseLocked stamps the resolve time on a tracked firing. Callers hold
// p.mu.
func (p *Pipeline) startCloseLocked(a Alert, f firing) (IssueKey, time.Time, bool) {
	if !f.resolvedAt.IsZero() {
		return f.issue, f.resolvedAt, false
	}
	f.resolvedAt = p.resolveTime(a)
	p.active[a.Fingerprint] = f
	return f.issue, f.resolvedAt, true
}

// cancelPendingClose withdraws a scheduled close because the alert is firing
// again. It reports whether there was one to withdraw.
func (p *Pipeline) cancelPendingClose(a Alert) (IssueKey, bool) {
	if a.Fingerprint == "" {
		return "", false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	f, ok := p.active[a.Fingerprint]
	if !ok || f.resolvedAt.IsZero() {
		return "", false
	}
	f.resolvedAt = time.Time{}
	p.active[a.Fingerprint] = f
	return f.issue, true
}

// pendingClose is one ticket whose grace window has elapsed.
type pendingClose struct {
	fingerprint string
	issue       IssueKey
	resolvedAt  time.Time
}

// dueCloses snapshots the tickets whose grace window has elapsed. The Jira
// calls that follow run outside the lock, so a webhook arriving mid-sweep is
// never blocked behind them.
func (p *Pipeline) dueCloses() []pendingClose {
	cutoff := p.now().Add(-p.closeGrace)
	p.mu.Lock()
	defer p.mu.Unlock()
	var due []pendingClose
	for fp, f := range p.active {
		if f.issue == "" || f.resolvedAt.IsZero() || f.resolvedAt.After(cutoff) {
			continue
		}
		due = append(due, pendingClose{fingerprint: fp, issue: f.issue, resolvedAt: f.resolvedAt})
	}
	return due
}

// stillPending reports whether the pending close a sweep snapshotted is still
// the current one. Read immediately before the close goes out, it shrinks the
// window in which a re-fire cannot stop it to the duration of the Jira call
// itself.
func (p *Pipeline) stillPending(fingerprint string, resolvedAt time.Time) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	f, ok := p.active[fingerprint]
	return ok && f.resolvedAt.Equal(resolvedAt)
}

// clearPendingClose drops the tracked firing after its close landed. It
// reports false when the entry no longer carries that resolve time — a re-fire
// beat the close — in which case the ticket is Done while its alert is firing
// and has to be put back. An entry that is simply gone reports true: another
// sweep already finished the job.
func (p *Pipeline) clearPendingClose(fingerprint string, resolvedAt time.Time) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	f, ok := p.active[fingerprint]
	if !ok {
		return true
	}
	if !f.resolvedAt.Equal(resolvedAt) {
		return false
	}
	delete(p.active, fingerprint)
	return true
}

// dueClose returns one fingerprint's elapsed pending close, if it has one.
func (p *Pipeline) dueClose(fingerprint string) (pendingClose, bool) {
	cutoff := p.now().Add(-p.closeGrace)
	p.mu.Lock()
	defer p.mu.Unlock()
	f, ok := p.active[fingerprint]
	if !ok || f.issue == "" || f.resolvedAt.IsZero() || f.resolvedAt.After(cutoff) {
		return pendingClose{}, false
	}
	return pendingClose{fingerprint: fingerprint, issue: f.issue, resolvedAt: f.resolvedAt}, true
}

// SweepResolved closes every ticket whose alert has stayed resolved for the
// grace period. It is driven by a periodic sweep rather than by per-ticket
// timers, so a restart drops pending closes instead of leaving a timer to fire
// against stale state.
func (p *Pipeline) SweepResolved(ctx context.Context) {
	p.sweepMu.Lock()
	defer p.sweepMu.Unlock()
	for _, due := range p.dueCloses() {
		// Each ticket costs at least one Jira round-trip; without this a
		// shutdown or an expired deadline still walks the whole table.
		if err := ctx.Err(); err != nil {
			p.log.Info("sweep stopped before finishing", "err", err)
			return
		}
		p.closeIfStillDue(ctx, due)
	}
}

// sweepOne evaluates a single fingerprint's pending close, for the resolve
// notification that may have just made it due. It gives up rather than waiting
// on a sweep already in progress: this runs on a dispatcher worker, of which
// there are four, and blocking them behind a slow Jira is what turns a Jira
// outage into refused alerts. Anything skipped here is picked up by the ticker.
func (p *Pipeline) sweepOne(ctx context.Context, fingerprint string) {
	if !p.sweepMu.TryLock() {
		return
	}
	defer p.sweepMu.Unlock()
	if due, ok := p.dueClose(fingerprint); ok {
		p.closeIfStillDue(ctx, due)
	}
}

// closeIfStillDue commits one pending close. Callers hold sweepMu.
func (p *Pipeline) closeIfStillDue(ctx context.Context, due pendingClose) {
	log := p.log.With("fingerprint", due.fingerprint, "issue", due.issue, "resolved_at", due.resolvedAt)
	open, err := p.jira.IsOpen(ctx, due.issue)
	switch {
	case err != nil:
		// Leave the pending close in place: the next sweep retries it.
		log.Error("pending close: ticket state unreadable", "err", err)
		return
	case !open:
		log.Info("pending close dropped: ticket is already closed")
		p.clearPendingClose(due.fingerprint, due.resolvedAt)
		return
	}
	// Re-read after the status round-trip: a re-fire during it has cancelled
	// this close, and the snapshot is the only thing still claiming the alert
	// is resolved.
	if !p.stillPending(due.fingerprint, due.resolvedAt) {
		log.Info("pending close abandoned: the alert re-fired while it was being closed")
		return
	}

	cerr := p.jira.Close(ctx, due.issue, due.resolvedAt)
	switch {
	case cerr == nil:
	case errors.Is(cerr, ErrClosedWithoutNote):
		// The transition landed and only the explanation did not. Retrying
		// would re-transition a Done ticket, so this counts as closed — but
		// the ticket now says nothing about why, which needs a human.
		log.Error("ticket closed but its closing comment did not post", "err", cerr)
	default:
		log.Error("pending close failed; will retry", "err", cerr)
		return
	}
	if !p.clearPendingClose(due.fingerprint, due.resolvedAt) {
		// The alert re-fired while the close was in flight. The ticket is Done
		// for a firing alert; nothing else will notice, because the re-fire's
		// own path already ran. This is not an auto-close — the ticket ends up
		// open again — so it is counted as a reopen instead.
		log.Info("close raced a re-fire; reopening the ticket")
		if rerr := p.jira.Reopen(ctx, due.issue); rerr != nil {
			p.counters.ticketReopensFailed.Add(1)
			log.Error("reopening a ticket closed against a firing alert failed", "err", rerr)
			return
		}
		p.counters.ticketReopens.Add(1)
		return
	}
	p.counters.ticketsAutoClosed.Add(1)
	log.Info("alert stayed resolved for the grace period; ticket closed", "grace", p.closeGrace.String())
}

// handleResolved records the resolve on the ticket and schedules its close.
//
// The per-alert context is often already spent by the time an alert reaches
// the pipeline, and a dropped resolve is invisible: the ticket simply never
// closes. So the Jira work here runs on a detached, bounded context, the same
// way the Holmes failure notice does.
func (p *Pipeline) handleResolved(ctx context.Context, a Alert) error {
	log := p.log.With("fingerprint", a.Fingerprint, "alert", a.Name())
	rctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), resolveHandlingTimeout)
	defer cancel()

	issue, at, first := p.markResolved(a)
	if issue == "" {
		// Nothing tracked here: either this process never filed the ticket, or
		// it restarted since. The ticket outlives the process, so ask Jira.
		found, err := p.jira.FindOpenByFingerprint(rctx, a)
		switch {
		case err != nil:
			log.Error("resolve: fingerprint lookup failed; ticket left open", "err", err)
			return nil
		case found == "":
			log.Info("resolve notification for an alert with no open ticket; nothing to close")
			return nil
		}
		if issue, at, first = p.adoptResolved(a, found); issue == "" {
			p.counters.fingerprintsUntracked.Add(1)
			log.Warn("resolve: fingerprint tracking is full; ticket left open", "issue", found)
			return nil
		}
		log.Info("resolve: adopted an existing ticket for an untracked fingerprint", "issue", issue)
	}

	if first {
		if err := p.jira.NoteResolved(rctx, issue, at); err != nil {
			log.Error("resolve note failed", "issue", issue, "err", err)
		}
		// The investigation was announced in Discord; its resolution is the
		// other half of that story and nothing else tells it.
		if derr := p.discord.Notify(rctx, a, Analysis{RootCause: resolveNotice(issue, at, p.closeGrace)}, issue, nil, nil); derr != nil {
			log.Error("discord resolve notice failed", "issue", issue, "err", derr)
		}
		log.Info("alert resolved; ticket close pending",
			"issue", issue, "resolved_at", at, "grace", p.closeGrace.String())
	}
	p.sweepOne(rctx, a.Fingerprint)
	return nil
}

func resolveNotice(issue IssueKey, at time.Time, grace time.Duration) string {
	return fmt.Sprintf("✅ Resolved at %s. %s closes automatically after %s unless the alert fires again.",
		at.UTC().Format(time.RFC3339), issue, grace.String())
}

// Handle runs one alert through the pipeline. Each output is independent: a
// failure is logged with the fingerprint and never suppresses later outputs.
func (p *Pipeline) Handle(ctx context.Context, a Alert) error {
	if a.Status == statusResolved {
		return p.handleResolved(ctx, a)
	}
	log := p.log.With("fingerprint", a.Fingerprint, "alert", a.Name())

	// A condition that fires again inside the grace window was never fixed.
	// The withdrawal happens before the investigation, not after it: an
	// investigation runs for minutes and a sweep in the middle of one would
	// otherwise close the ticket it is about to write to.
	if issue, cancelled := p.cancelPendingClose(a); cancelled {
		log.Info("alert re-fired inside the close grace period; pending close cancelled", "issue", issue)
	}

	// Alertmanager re-notifies a still-firing alert every repeat_interval.
	// Re-running the investigation produces the same analysis at full LLM
	// cost, so a repeat of a firing episode this process already investigated
	// is recorded on the existing ticket instead.
	if f, ok := p.repeatOf(a); ok {
		open, err := p.jira.IsOpen(ctx, f.issue)
		switch {
		case err != nil:
			// Unknown ticket state is not a reason to skip work.
			log.Error("refire ticket check failed; investigating", "issue", f.issue, "err", err)
		case open:
			n, tracked := p.countRepeat(a)
			if !tracked {
				// A sweep forgot this fingerprint between the check above and
				// here, so nothing is left to hang the skip on: treat it as
				// new work rather than record a repeat of nothing.
				log.Info("refire lost its tracked ticket mid-check; investigating", "issue", f.issue)
				break
			}
			if nerr := p.jira.NoteRefire(ctx, f.issue, n); nerr != nil {
				log.Error("refire note failed", "issue", f.issue, "err", nerr)
			}
			p.counters.repeatsSkipped.Add(1)
			log.Info("alert still firing with an open ticket; skipping investigation",
				"issue", f.issue, "repeat", n)
			return nil
		default:
			// Closed since we filed it: a refire is new work again.
			p.forget(a)
		}
	}
	p.counters.investigationsRun.Add(1)

	an, err := p.holmes.Investigate(ctx, a)
	if err != nil {
		// Holmes is terminal: nothing to fan out. Still tell humans so they
		// are not left blind. The per-alert context is usually already dead
		// here (deadline expired mid-investigation), so the notice gets a
		// short-lived context of its own.
		log.Error("holmes investigation failed", "err", err)
		nctx, ncancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer ncancel()
		if derr := p.discord.Notify(nctx, a, Analysis{RootCause: "⚠️ investigation failed: " + err.Error()}, "", nil, nil); derr != nil {
			log.Error("discord failure notice failed", "err", derr)
		}
		return err
	}

	// Structured patch is best-effort; nil is the common case.
	var patch *Patch
	if pt, perr := p.patch.Generate(ctx, an); perr != nil {
		log.Error("patch generation failed", "err", perr)
	} else {
		patch = pt
	}

	var issue IssueKey
	if k, jerr := p.jira.Upsert(ctx, a, an); jerr != nil {
		log.Error("jira upsert failed", "err", jerr)
	} else {
		issue = k
		p.remember(a, issue)
	}

	var prLink *PRLink
	if patch != nil {
		switch link, gerr := p.github.OpenPR(ctx, *patch, issue); {
		case gerr == nil:
			prLink = &link
		case errors.Is(gerr, ErrRepoNotAllowed):
			// Not a transport hiccup: a complete remediation was produced and
			// then thrown away because it addressed the wrong repository. It
			// gets its own message and the proposed repo as a field so a run of
			// them is greppable and countable, rather than reading as one more
			// flaky GitHub call.
			p.counters.reposRejected.Add(1)
			log.Error("remediation discarded: proposed repository is not allowlisted",
				"proposed_repo", patch.Repo, "file_path", patch.FilePath, "issue", issue, "err", gerr)
		case errors.Is(gerr, ErrPathNotAllowed), errors.Is(gerr, ErrPathDenied):
			// Same shape as the repo refusal: a complete patch aimed at a
			// file the GitOps controller does not read, does not exist, or is
			// a known exception to the allowlist (a dormant release that an
			// allow glob matches anyway). Counted together — both mean the
			// same thing to an operator, a proposal that could never take
			// effect — separately from repo rejections and flaky GitHub
			// calls, so a model that keeps inventing layouts shows up as a
			// trend.
			p.counters.pathsRejected.Add(1)
			log.Error("remediation discarded: proposed file is not a watched, existing manifest",
				"proposed_repo", patch.Repo, "file_path", patch.FilePath, "issue", issue, "err", gerr)
		case errors.Is(gerr, ErrBranchExists):
			// A PR for this ticket already carries an earlier proposal; this
			// one is dropped rather than pushed onto it. Not an error state:
			// the ticket is already in review.
			p.counters.branchesSkipped.Add(1)
			log.Info("remediation skipped: a PR branch for this ticket already exists",
				"proposed_repo", patch.Repo, "file_path", patch.FilePath, "issue", issue)
		case errors.Is(gerr, ErrBranchOrphaned):
			// The branch exists but no PR was ever opened from it: an earlier
			// run's commit, if the PUT got that far, is sitting on a branch
			// nobody has been asked to review. This is not the quiet "already
			// in review" skip above — it needs a human to go look, so it gets
			// its own counter rather than branchesSkipped, which is expected
			// to tick on ordinary refires and would hide this.
			p.counters.branchesOrphaned.Add(1)
			log.Error("remediation branch exists with no pull request; a previous run likely failed after creating it",
				"proposed_repo", patch.Repo, "file_path", patch.FilePath, "issue", issue, "err", gerr)
		default:
			log.Error("github pr failed", "err", gerr)
		}
	}

	if derr := p.discord.Notify(ctx, a, an, issue, prLink, patch); derr != nil {
		log.Error("discord notify failed", "err", derr)
	}
	return nil
}
