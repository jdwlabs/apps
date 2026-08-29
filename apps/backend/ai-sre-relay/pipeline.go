package main

import (
	"context"
	"errors"
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
}

type Pipeline struct {
	holmes  holmesInvestigator
	patch   patcher
	jira    jiraUpserter
	github  prOpener
	discord discordNotifier
	log     *slog.Logger

	counters counters

	mu     sync.Mutex
	active map[string]firing
}

func NewPipeline(h holmesInvestigator, pg patcher, j jiraUpserter, gh prOpener, d discordNotifier, log *slog.Logger) *Pipeline {
	return &Pipeline{
		holmes: h, patch: pg, jira: j, github: gh, discord: d, log: log,
		active: map[string]firing{},
	}
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
func (p *Pipeline) countRepeat(a Alert) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	f := p.active[a.Fingerprint]
	f.repeats++
	p.active[a.Fingerprint] = f
	return f.repeats
}

func (p *Pipeline) remember(a Alert, issue IssueKey) {
	if a.Fingerprint == "" || issue == "" {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, known := p.active[a.Fingerprint]; !known && len(p.active) >= maxTrackedFirings {
		return
	}
	p.active[a.Fingerprint] = firing{startsAt: a.StartsAt, issue: issue}
}

func (p *Pipeline) forget(a Alert) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.active, a.Fingerprint)
}

// Handle runs one alert through the pipeline. Each output is independent: a
// failure is logged with the fingerprint and never suppresses later outputs.
func (p *Pipeline) Handle(ctx context.Context, a Alert) error {
	log := p.log.With("fingerprint", a.Fingerprint, "alert", a.Name())

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
			n := p.countRepeat(a)
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
		case errors.Is(gerr, ErrPathNotAllowed):
			// Same shape as the repo refusal: a complete patch aimed at a
			// file the GitOps controller does not read, or that does not
			// exist. Counted separately so a model that keeps inventing
			// layouts shows up as a trend rather than as flaky GitHub calls.
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
		default:
			log.Error("github pr failed", "err", gerr)
		}
	}

	if derr := p.discord.Notify(ctx, a, an, issue, prLink, patch); derr != nil {
		log.Error("discord notify failed", "err", derr)
	}
	return nil
}
