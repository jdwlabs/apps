package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeHolmes struct {
	an  Analysis
	err error
}

func (f fakeHolmes) Investigate(context.Context, Alert) (Analysis, error) { return f.an, f.err }

type fakePatcher struct{ p *Patch }

func (f fakePatcher) Generate(context.Context, Analysis) (*Patch, error) { return f.p, nil }

type fakeJira struct {
	key      IssueKey
	called   bool
	upserts  int
	closed   bool // IsOpen reports the issue as Done
	refires  []int
	openErr  error
	refireOn bool // records that IsOpen was consulted at all

	resolveNotes []time.Time
	closes       []time.Time
	// closeErr fails the transition, closeCommentErr the closing comment, so
	// a test can tell a ticket left open from one closed without its note.
	closeErr        error
	closeCommentErr error
	closeComments   int
	reopens         int

	// findKey is the ticket a fingerprint search recovers, standing in for a
	// ticket this process did not file itself.
	findKey IssueKey
	findErr error
	finds   int

	// onIsOpen runs inside IsOpen, so a test can land a concurrent state
	// change at the exact point the pipeline has read the ticket status.
	onIsOpen func()

	noteResolvedCtxErr error
	closeCtxErr        error
}

func (f *fakeJira) Upsert(context.Context, Alert, Analysis) (IssueKey, error) {
	f.called = true
	f.upserts++
	return f.key, nil
}

func (f *fakeJira) IsOpen(context.Context, IssueKey) (bool, error) {
	f.refireOn = true
	if f.onIsOpen != nil {
		f.onIsOpen()
	}
	if f.openErr != nil {
		return false, f.openErr
	}
	return !f.closed, nil
}

func (f *fakeJira) FindOpenByFingerprint(_ context.Context, _ Alert) (IssueKey, error) {
	f.finds++
	if f.findErr != nil {
		return "", f.findErr
	}
	return f.findKey, nil
}

func (f *fakeJira) Reopen(context.Context, IssueKey) error {
	f.reopens++
	f.closed = false
	return nil
}

func (f *fakeJira) NoteRefire(_ context.Context, _ IssueKey, count int) error {
	f.refires = append(f.refires, count)
	return nil
}

func (f *fakeJira) NoteResolved(ctx context.Context, _ IssueKey, at time.Time) error {
	f.noteResolvedCtxErr = ctx.Err()
	f.resolveNotes = append(f.resolveNotes, at)
	return nil
}

// Close mirrors the real client: the transition lands first, and a ticket
// transitioned but not commented reports ErrClosedWithoutNote rather than a
// plain failure, so the caller does not retry a Done ticket forever.
func (f *fakeJira) Close(ctx context.Context, _ IssueKey, at time.Time) error {
	f.closeCtxErr = ctx.Err()
	if f.closeErr != nil {
		return f.closeErr
	}
	f.closes = append(f.closes, at)
	f.closed = true
	if f.closeCommentErr != nil {
		return fmt.Errorf("%w: %v", ErrClosedWithoutNote, f.closeCommentErr)
	}
	f.closeComments++
	return nil
}

type fakeGH struct{ called bool }

func (f *fakeGH) OpenPR(context.Context, Patch, IssueKey) (PRLink, error) {
	f.called = true
	return "http://pr/1", nil
}

type fakeDiscord struct {
	called       bool
	pr           *PRLink
	notifies     int
	lastAlert    Alert
	lastAnalysis Analysis
}

func (f *fakeDiscord) Notify(_ context.Context, a Alert, an Analysis, _ IssueKey, pr *PRLink, _ *Patch) error {
	f.called = true
	f.notifies++
	f.lastAlert = a
	f.lastAnalysis = an
	f.pr = pr
	return nil
}

func silentLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestPipelineHappyPathOpensPR(t *testing.T) {
	j, gh, d := &fakeJira{key: "JDWLABS-1"}, &fakeGH{}, &fakeDiscord{}
	p := NewPipeline(fakeHolmes{an: Analysis{RootCause: "x"}}, fakePatcher{p: &Patch{Repo: "a/b"}}, j, gh, d, silentLogger())
	if err := p.Handle(context.Background(), Alert{Fingerprint: "f"}); err != nil {
		t.Fatal(err)
	}
	if !j.called || !gh.called || !d.called || d.pr == nil {
		t.Fatalf("steps: jira=%v gh=%v discord=%v pr=%v", j.called, gh.called, d.called, d.pr)
	}
}

func TestPipelineNoPatchSkipsPR(t *testing.T) {
	gh, d := &fakeGH{}, &fakeDiscord{}
	p := NewPipeline(fakeHolmes{an: Analysis{RootCause: "x"}}, fakePatcher{p: nil}, &fakeJira{key: "JDWLABS-2"}, gh, d, silentLogger())
	_ = p.Handle(context.Background(), Alert{})
	if gh.called {
		t.Fatal("PR opened despite nil patch")
	}
	if !d.called || d.pr != nil {
		t.Fatal("discord should fire with nil PR link")
	}
}

func TestPipelineHolmesFailureStillNotifies(t *testing.T) {
	d := &fakeDiscord{}
	p := NewPipeline(fakeHolmes{err: errors.New("down")}, fakePatcher{}, &fakeJira{}, &fakeGH{}, d, silentLogger())
	_ = p.Handle(context.Background(), Alert{})
	if !d.called {
		t.Fatal("expected failure notice to Discord")
	}
}

type ctxCheckingDiscord struct {
	called bool
	ctxErr error
}

func (f *ctxCheckingDiscord) Notify(ctx context.Context, _ Alert, _ Analysis, _ IssueKey, _ *PRLink, _ *Patch) error {
	f.called = true
	f.ctxErr = ctx.Err()
	return f.ctxErr
}

// The usual way Holmes fails is the per-alert deadline expiring mid
// investigation — the same dead context must not also kill the notice that
// tells humans about it.
func TestPipelineFailureNoticeSurvivesDeadContext(t *testing.T) {
	d := &ctxCheckingDiscord{}
	p := NewPipeline(fakeHolmes{err: errors.New("deadline exceeded")}, fakePatcher{}, &fakeJira{}, &fakeGH{}, d, silentLogger())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_ = p.Handle(ctx, Alert{Fingerprint: "f"})
	if !d.called || d.ctxErr != nil {
		t.Fatalf("failure notice must run on a live context: called=%v ctxErr=%v", d.called, d.ctxErr)
	}
}

type fakeJiraErr struct{}

func (fakeJiraErr) Upsert(context.Context, Alert, Analysis) (IssueKey, error) {
	return "", errors.New("jira down")
}

func (fakeJiraErr) IsOpen(context.Context, IssueKey) (bool, error) {
	return false, errors.New("jira down")
}

func (fakeJiraErr) NoteRefire(context.Context, IssueKey, int) error {
	return errors.New("jira down")
}

func (fakeJiraErr) NoteResolved(context.Context, IssueKey, time.Time) error {
	return errors.New("jira down")
}

func (fakeJiraErr) Close(context.Context, IssueKey, time.Time) error {
	return errors.New("jira down")
}

func (fakeJiraErr) FindOpenByFingerprint(context.Context, Alert) (IssueKey, error) {
	return "", errors.New("jira down")
}

func (fakeJiraErr) Reopen(context.Context, IssueKey) error {
	return errors.New("jira down")
}

func TestPipelineJiraFailureStillNotifiesDiscord(t *testing.T) {
	d := &fakeDiscord{}
	p := NewPipeline(fakeHolmes{an: Analysis{RootCause: "x"}}, fakePatcher{p: nil}, fakeJiraErr{}, &fakeGH{}, d, silentLogger())
	if err := p.Handle(context.Background(), Alert{Fingerprint: "fp"}); err != nil {
		t.Fatal(err)
	}
	if !d.called {
		t.Fatal("discord must fire even when jira fails")
	}
}

type fakeGHErr struct{}

func (fakeGHErr) OpenPR(context.Context, Patch, IssueKey) (PRLink, error) {
	return "", errors.New("github down")
}

func TestPipelineGithubFailureStillNotifiesDiscord(t *testing.T) {
	d := &fakeDiscord{}
	patch := &Patch{Repo: "a/b", FilePath: "f", NewContent: "c", Rationale: "r", Confidence: 0.9}
	p := NewPipeline(fakeHolmes{an: Analysis{RootCause: "x"}}, fakePatcher{p: patch}, &fakeJira{key: "JDWLABS-3"}, fakeGHErr{}, d, silentLogger())
	if err := p.Handle(context.Background(), Alert{Fingerprint: "fp"}); err != nil {
		t.Fatal(err)
	}
	if !d.called {
		t.Fatal("discord must fire even when github fails")
	}
	if d.pr != nil {
		t.Fatalf("PR link must be nil when github fails, got %v", d.pr)
	}
}

// countingHolmes records how many investigations actually ran.
type countingHolmes struct{ n int }

func (c *countingHolmes) Investigate(context.Context, Alert) (Analysis, error) {
	c.n++
	return Analysis{RootCause: "x"}, nil
}

func repeatAlert() Alert {
	return Alert{Fingerprint: "fp", StartsAt: "2026-07-28T03:02:59Z", Labels: map[string]string{"alertname": "TempoNoSpansReceived"}}
}

// The 4h repeat of a still-firing alert must not pay for a second
// investigation; it lands on the existing ticket instead.
func TestPipelineSkipsRepeatWithOpenTicket(t *testing.T) {
	h := &countingHolmes{}
	j := &fakeJira{key: "JDWLABS-229"}
	p := NewPipeline(h, fakePatcher{p: nil}, j, &fakeGH{}, &fakeDiscord{}, silentLogger())

	for range 3 {
		if err := p.Handle(context.Background(), repeatAlert()); err != nil {
			t.Fatal(err)
		}
	}
	if h.n != 1 {
		t.Fatalf("investigations = %d, want 1", h.n)
	}
	if j.upserts != 1 {
		t.Fatalf("jira upserts = %d, want 1", j.upserts)
	}
	// A skipped repeat is still recorded, and numbered.
	if len(j.refires) != 2 || j.refires[0] != 1 || j.refires[1] != 2 {
		t.Fatalf("refire notes = %v, want [1 2]", j.refires)
	}
	if got := p.Counters().repeatsSkipped.Load(); got != 2 {
		t.Fatalf("repeatsSkipped = %d, want 2", got)
	}
	if got := p.Counters().investigationsRun.Load(); got != 1 {
		t.Fatalf("investigationsRun = %d, want 1", got)
	}
}

// A human closing the ticket means the next refire is new work again.
func TestPipelineReinvestigatesAfterTicketClosed(t *testing.T) {
	h := &countingHolmes{}
	j := &fakeJira{key: "JDWLABS-229"}
	p := NewPipeline(h, fakePatcher{p: nil}, j, &fakeGH{}, &fakeDiscord{}, silentLogger())

	if err := p.Handle(context.Background(), repeatAlert()); err != nil {
		t.Fatal(err)
	}
	j.closed = true
	if err := p.Handle(context.Background(), repeatAlert()); err != nil {
		t.Fatal(err)
	}
	if h.n != 2 {
		t.Fatalf("investigations = %d, want 2 (closed ticket must re-investigate)", h.n)
	}
	if len(j.refires) != 0 {
		t.Fatalf("closed ticket must not get a refire note, got %v", j.refires)
	}
}

// A new firing episode shares the fingerprint but not startsAt, so it is not a
// repeat and must be investigated.
func TestPipelineInvestigatesNewFiringEpisode(t *testing.T) {
	h := &countingHolmes{}
	j := &fakeJira{key: "JDWLABS-229"}
	p := NewPipeline(h, fakePatcher{p: nil}, j, &fakeGH{}, &fakeDiscord{}, silentLogger())

	first := repeatAlert()
	if err := p.Handle(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	second := first
	second.StartsAt = "2026-07-29T09:00:00Z"
	if err := p.Handle(context.Background(), second); err != nil {
		t.Fatal(err)
	}
	if h.n != 2 {
		t.Fatalf("investigations = %d, want 2 (new episode must re-investigate)", h.n)
	}
}

// An unreadable ticket state must fail toward doing the work, not toward
// silently dropping the alert.
func TestPipelineInvestigatesWhenTicketStateUnknown(t *testing.T) {
	h := &countingHolmes{}
	j := &fakeJira{key: "JDWLABS-229"}
	p := NewPipeline(h, fakePatcher{p: nil}, j, &fakeGH{}, &fakeDiscord{}, silentLogger())

	if err := p.Handle(context.Background(), repeatAlert()); err != nil {
		t.Fatal(err)
	}
	j.openErr = errors.New("jira unreachable")
	if err := p.Handle(context.Background(), repeatAlert()); err != nil {
		t.Fatal(err)
	}
	if h.n != 2 {
		t.Fatalf("investigations = %d, want 2 (unknown state must re-investigate)", h.n)
	}
}

type repoRejectingGH struct{}

func (repoRejectingGH) OpenPR(_ context.Context, p Patch, _ IssueKey) (PRLink, error) {
	return "", fmt.Errorf("%w: proposed %q, allowed [jdwlabs/platform]", ErrRepoNotAllowed, p.Repo)
}

// A remediation dropped at the allowlist is a produced-then-discarded fix, not
// a flaky API call. It must be attributable in the log and countable in the
// metrics, or a systematic run of them looks the same as no alerts at all.
func TestPipelineRepoRejectionIsLoudAndCounted(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))
	patch := &Patch{Repo: "example/gitops", FilePath: "values.yaml", NewContent: "c", Confidence: 0.9}
	d := &fakeDiscord{}
	p := NewPipeline(fakeHolmes{an: Analysis{RootCause: "x"}}, fakePatcher{p: patch},
		&fakeJira{key: "JDWLABS-312"}, repoRejectingGH{}, d, log)

	if err := p.Handle(context.Background(), Alert{Fingerprint: "fp"}); err != nil {
		t.Fatal(err)
	}
	if got := p.Counters().reposRejected.Load(); got != 1 {
		t.Fatalf("reposRejected = %d, want 1", got)
	}
	out := buf.String()
	for _, want := range []string{"not allowlisted", "example/gitops", "jdwlabs/platform"} {
		if !strings.Contains(out, want) {
			t.Fatalf("rejection log omits %q:\n%s", want, out)
		}
	}
	if d.pr != nil {
		t.Fatal("no PR link may be reported for a discarded remediation")
	}
}

func TestCountersExposeRepoRejections(t *testing.T) {
	var c counters
	c.reposRejected.Add(3)
	var buf bytes.Buffer
	c.writeTo(&buf)
	if !strings.Contains(buf.String(), "ai_sre_relay_repo_rejections_total 3") {
		t.Fatalf("counter not exposed:\n%s", buf.String())
	}
}

type pathRejectingGH struct{}

func (pathRejectingGH) OpenPR(_ context.Context, p Patch, _ IssueKey) (PRLink, error) {
	return "", fmt.Errorf("%w: proposed %q, allowed [tenants/*/services/*/values.yaml]", ErrPathNotAllowed, p.FilePath)
}

// A patch discarded for its file path, not its repo, must land in the
// pathsRejected counter and log line — the two allowlists diagnose different
// problems and must not be conflated.
func TestPipelinePathRejectionIsLoudAndCounted(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))
	patch := &Patch{Repo: "jdwlabs/platform", FilePath: "cluster/grafana/gitsync.yaml", NewContent: "c", Confidence: 0.9}
	d := &fakeDiscord{}
	p := NewPipeline(fakeHolmes{an: Analysis{RootCause: "x"}}, fakePatcher{p: patch},
		&fakeJira{key: "JDWLABS-313"}, pathRejectingGH{}, d, log)

	if err := p.Handle(context.Background(), Alert{Fingerprint: "fp"}); err != nil {
		t.Fatal(err)
	}
	if got := p.Counters().pathsRejected.Load(); got != 1 {
		t.Fatalf("pathsRejected = %d, want 1", got)
	}
	if got := p.Counters().reposRejected.Load(); got != 0 {
		t.Fatalf("reposRejected = %d, want 0 (this was a path rejection)", got)
	}
	out := buf.String()
	for _, want := range []string{"not a watched, existing manifest", "cluster/grafana/gitsync.yaml"} {
		if !strings.Contains(out, want) {
			t.Fatalf("rejection log omits %q:\n%s", want, out)
		}
	}
	if d.pr != nil {
		t.Fatal("no PR link may be reported for a discarded remediation")
	}
}

type branchExistsGH struct{}

func (branchExistsGH) OpenPR(_ context.Context, p Patch, _ IssueKey) (PRLink, error) {
	return "", fmt.Errorf("%w: fix/ai-sre/jdwlabs-314 (repo %s)", ErrBranchExists, p.Repo)
}

// A branch that already has a PR is a quiet, expected skip: counted as
// branchesSkipped, logged at Info, and never mistaken for the orphaned case.
func TestPipelineBranchExistsIsQuietAndCounted(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))
	patch := &Patch{Repo: "jdwlabs/platform", FilePath: "tenants/platform/services/vault/values.yaml", NewContent: "c", Confidence: 0.9}
	d := &fakeDiscord{}
	p := NewPipeline(fakeHolmes{an: Analysis{RootCause: "x"}}, fakePatcher{p: patch},
		&fakeJira{key: "JDWLABS-314"}, branchExistsGH{}, d, log)

	if err := p.Handle(context.Background(), Alert{Fingerprint: "fp"}); err != nil {
		t.Fatal(err)
	}
	if got := p.Counters().branchesSkipped.Load(); got != 1 {
		t.Fatalf("branchesSkipped = %d, want 1", got)
	}
	if got := p.Counters().branchesOrphaned.Load(); got != 0 {
		t.Fatalf("branchesOrphaned = %d, want 0 (this branch has a PR)", got)
	}
	if strings.Contains(buf.String(), `"level":"ERROR"`) {
		t.Fatalf("an already-in-review skip must not log at Error:\n%s", buf.String())
	}
}

type branchOrphanedGH struct{}

func (branchOrphanedGH) OpenPR(_ context.Context, p Patch, _ IssueKey) (PRLink, error) {
	return "", fmt.Errorf("%w: fix/ai-sre/jdwlabs-315 (repo %s)", ErrBranchOrphaned, p.Repo)
}

// An orphaned branch — one a previous run created and then failed to open a
// PR from — must be loud: counted separately from the expected
// branchesSkipped skip, and logged at Error, because nobody has been asked
// to review the commit sitting on it.
func TestPipelineBranchOrphanedIsLoudAndCounted(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))
	patch := &Patch{Repo: "jdwlabs/platform", FilePath: "tenants/platform/services/vault/values.yaml", NewContent: "c", Confidence: 0.9}
	d := &fakeDiscord{}
	p := NewPipeline(fakeHolmes{an: Analysis{RootCause: "x"}}, fakePatcher{p: patch},
		&fakeJira{key: "JDWLABS-315"}, branchOrphanedGH{}, d, log)

	if err := p.Handle(context.Background(), Alert{Fingerprint: "fp"}); err != nil {
		t.Fatal(err)
	}
	if got := p.Counters().branchesOrphaned.Load(); got != 1 {
		t.Fatalf("branchesOrphaned = %d, want 1", got)
	}
	if got := p.Counters().branchesSkipped.Load(); got != 0 {
		t.Fatalf("branchesSkipped = %d, want 0 (this branch has no PR)", got)
	}
	out := buf.String()
	if !strings.Contains(out, `"level":"ERROR"`) {
		t.Fatalf("an orphaned branch must log at Error:\n%s", out)
	}
	if !strings.Contains(out, "fix/ai-sre/jdwlabs-315") {
		t.Fatalf("orphaned-branch log omits the branch:\n%s", out)
	}
}

func TestCountersExposePathAndBranchOutcomes(t *testing.T) {
	var c counters
	c.pathsRejected.Add(2)
	c.branchesSkipped.Add(5)
	c.branchesOrphaned.Add(1)
	var buf bytes.Buffer
	c.writeTo(&buf)
	for _, want := range []string{
		"ai_sre_relay_path_rejections_total 2",
		"ai_sre_relay_branches_skipped_total 5",
		"ai_sre_relay_branches_orphaned_total 1",
	} {
		if !strings.Contains(buf.String(), want) {
			t.Fatalf("counter not exposed: %q\n%s", want, buf.String())
		}
	}
}

type fakePatcherErr struct{}

func (fakePatcherErr) Generate(context.Context, Analysis) (*Patch, error) {
	return nil, errors.New("no patch")
}

func TestPipelinePatchErrorStillContinues(t *testing.T) {
	d, gh := &fakeDiscord{}, &fakeGH{}
	p := NewPipeline(fakeHolmes{an: Analysis{RootCause: "x"}}, fakePatcherErr{}, &fakeJira{key: "ABC-1"}, gh, d, silentLogger())
	if err := p.Handle(context.Background(), Alert{}); err != nil {
		t.Fatal(err)
	}
	if gh.called {
		t.Fatal("PR must be skipped when patch generation errors")
	}
	if !d.called {
		t.Fatal("discord must still fire after a patch error")
	}
}

// fakeClock drives the resolved-since grace window without sleeping.
type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fakeClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

const testResolveTime = "2026-07-28T04:00:00Z"

func mustParseTime(t *testing.T, s string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("parse %q: %v", s, err)
	}
	return parsed
}

func resolvedAlert() Alert {
	a := repeatAlert()
	a.Status = statusResolved
	a.EndsAt = testResolveTime
	return a
}

// newTestPipeline wires a pipeline whose grace window is driven by clk.
func newTestPipeline(t *testing.T, h holmesInvestigator, j jiraUpserter, grace time.Duration, clk *fakeClock) *Pipeline {
	t.Helper()
	clk.t = mustParseTime(t, testResolveTime)
	return NewPipeline(h, fakePatcher{p: nil}, j, &fakeGH{}, &fakeDiscord{}, silentLogger(),
		withCloseGrace(grace), withClock(clk.now))
}

// A resolve notification is recorded on the ticket immediately, but the close
// waits out the grace window: an alert that resolves and re-fires an hour
// later never had a fixed condition.
func TestPipelineResolvedAlertIsNotedWithoutClosing(t *testing.T) {
	clk := &fakeClock{}
	j := &fakeJira{key: "JDWLABS-400"}
	p := newTestPipeline(t, &countingHolmes{}, j, 6*time.Hour, clk)

	if err := p.Handle(context.Background(), repeatAlert()); err != nil {
		t.Fatal(err)
	}
	if err := p.Handle(context.Background(), resolvedAlert()); err != nil {
		t.Fatal(err)
	}

	want := mustParseTime(t, testResolveTime)
	if len(j.resolveNotes) != 1 || !j.resolveNotes[0].Equal(want) {
		t.Fatalf("resolve notes = %v, want one note at %v", j.resolveNotes, want)
	}
	if len(j.closes) != 0 {
		t.Fatalf("ticket closed inside the grace window: %v", j.closes)
	}
}

// Alertmanager re-sends a resolved notification while the alert stays
// resolved; the grace clock starts at the first one and the ticket collects
// one note, not one per notification.
func TestPipelineRepeatedResolveNotesOnce(t *testing.T) {
	clk := &fakeClock{}
	j := &fakeJira{key: "JDWLABS-401"}
	p := newTestPipeline(t, &countingHolmes{}, j, 6*time.Hour, clk)

	if err := p.Handle(context.Background(), repeatAlert()); err != nil {
		t.Fatal(err)
	}
	// No end time, so each notification resolves at the moment it is received
	// and a later one would visibly push the close out if it were allowed to.
	first := resolvedAlert()
	first.EndsAt = ""
	if err := p.Handle(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	clk.advance(5 * time.Hour)
	if err := p.Handle(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	clk.advance(time.Hour)
	p.SweepResolved(context.Background())

	if len(j.resolveNotes) != 1 {
		t.Fatalf("resolve notes = %v, want exactly one", j.resolveNotes)
	}
	if len(j.closes) != 1 {
		t.Fatalf("closes = %v, want one: a repeated resolve must not restart the grace window", j.closes)
	}
}

// Once the alert has stayed resolved for the whole grace window the relay
// closes its own ticket, naming the resolve time.
func TestPipelineClosesTicketAfterGrace(t *testing.T) {
	clk := &fakeClock{}
	j := &fakeJira{key: "JDWLABS-402"}
	p := newTestPipeline(t, &countingHolmes{}, j, 6*time.Hour, clk)

	if err := p.Handle(context.Background(), repeatAlert()); err != nil {
		t.Fatal(err)
	}
	if err := p.Handle(context.Background(), resolvedAlert()); err != nil {
		t.Fatal(err)
	}
	clk.advance(6 * time.Hour)
	p.SweepResolved(context.Background())

	want := mustParseTime(t, testResolveTime)
	if len(j.closes) != 1 || !j.closes[0].Equal(want) {
		t.Fatalf("closes = %v, want one close naming %v", j.closes, want)
	}
	if got := p.Counters().ticketsAutoClosed.Load(); got != 1 {
		t.Fatalf("ticketsAutoClosed = %d, want 1", got)
	}
	// A second sweep must not comment on the ticket again.
	p.SweepResolved(context.Background())
	if len(j.closes) != 1 {
		t.Fatalf("closes = %v, want the close to happen exactly once", j.closes)
	}
}

// A re-fire inside the grace window means the condition was never fixed: the
// pending close is cancelled and the ticket stays open.
func TestPipelineRefireCancelsPendingClose(t *testing.T) {
	clk := &fakeClock{}
	j := &fakeJira{key: "JDWLABS-403"}
	p := newTestPipeline(t, &countingHolmes{}, j, 6*time.Hour, clk)

	if err := p.Handle(context.Background(), repeatAlert()); err != nil {
		t.Fatal(err)
	}
	if err := p.Handle(context.Background(), resolvedAlert()); err != nil {
		t.Fatal(err)
	}
	// The grace window elapses while the re-fire is being investigated, which
	// is the case the cancellation has to survive: investigations run for
	// minutes, and a sweep inside one must not close the ticket it is about
	// to write to.
	clk.advance(12 * time.Hour)
	refire := repeatAlert()
	refire.StartsAt = "2026-07-28T05:00:00Z"
	if err := p.Handle(context.Background(), refire); err != nil {
		t.Fatal(err)
	}
	p.SweepResolved(context.Background())

	if len(j.closes) != 0 {
		t.Fatalf("re-fired alert had its ticket closed anyway: %v", j.closes)
	}
	if got := p.Counters().ticketsAutoClosed.Load(); got != 0 {
		t.Fatalf("ticketsAutoClosed = %d, want 0", got)
	}
}

// sweepingHolmes runs a sweep from inside the investigation, standing in for
// the periodic sweep landing while a re-fire is being investigated.
type sweepingHolmes struct {
	p *Pipeline
	n int
}

func (h *sweepingHolmes) Investigate(ctx context.Context, _ Alert) (Analysis, error) {
	h.n++
	h.p.SweepResolved(ctx)
	return Analysis{RootCause: "x"}, nil
}

// The pending close must be withdrawn before the investigation starts, not
// after it finishes: an investigation runs for minutes and a sweep landing
// mid-way would otherwise close the ticket the alert is still firing on.
func TestPipelineRefireCancelsPendingCloseBeforeInvestigating(t *testing.T) {
	clk := &fakeClock{}
	j := &fakeJira{key: "JDWLABS-410"}
	h := &sweepingHolmes{}
	p := newTestPipeline(t, h, j, 6*time.Hour, clk)
	h.p = p

	if err := p.Handle(context.Background(), repeatAlert()); err != nil {
		t.Fatal(err)
	}
	if err := p.Handle(context.Background(), resolvedAlert()); err != nil {
		t.Fatal(err)
	}
	clk.advance(12 * time.Hour)

	refire := repeatAlert()
	refire.StartsAt = "2026-07-28T05:00:00Z"
	if err := p.Handle(context.Background(), refire); err != nil {
		t.Fatal(err)
	}
	if len(j.closes) != 0 {
		t.Fatalf("ticket closed by a sweep during the re-fire investigation: %v", j.closes)
	}
}

// After the relay closes a ticket, the same fingerprint firing again is new
// work: the Done ticket must not suppress the investigation. Jira's upsert is
// what routes that investigation back onto the closed ticket by reopening it.
func TestPipelineAutoClosedTicketDoesNotSuppressNextFiring(t *testing.T) {
	clk := &fakeClock{}
	h := &countingHolmes{}
	j := &fakeJira{key: "JDWLABS-404"}
	p := newTestPipeline(t, h, j, 6*time.Hour, clk)

	if err := p.Handle(context.Background(), repeatAlert()); err != nil {
		t.Fatal(err)
	}
	if err := p.Handle(context.Background(), resolvedAlert()); err != nil {
		t.Fatal(err)
	}
	clk.advance(6 * time.Hour)
	p.SweepResolved(context.Background())
	if len(j.closes) != 1 {
		t.Fatalf("closes = %v, want the ticket closed before the re-fire", j.closes)
	}

	refire := repeatAlert()
	refire.StartsAt = "2026-07-29T09:00:00Z"
	if err := p.Handle(context.Background(), refire); err != nil {
		t.Fatal(err)
	}
	if h.n != 2 {
		t.Fatalf("investigations = %d, want 2 (a closed ticket must not absorb a new firing)", h.n)
	}
	if j.upserts != 2 {
		t.Fatalf("jira upserts = %d, want 2 (the re-fire reopens the ticket via upsert)", j.upserts)
	}
	if len(j.refires) != 0 {
		t.Fatalf("a closed ticket must not collect refire notes, got %v", j.refires)
	}
}

// Even a repeat of the same firing episode must not be absorbed once the
// ticket is Done — the dedup decision is fingerprint plus ticket status.
func TestPipelineDoneTicketNeverSuppressesRepeat(t *testing.T) {
	clk := &fakeClock{}
	h := &countingHolmes{}
	j := &fakeJira{key: "JDWLABS-405"}
	p := newTestPipeline(t, h, j, 6*time.Hour, clk)

	if err := p.Handle(context.Background(), repeatAlert()); err != nil {
		t.Fatal(err)
	}
	// The mapping stays in place — a human closed the ticket, nothing swept
	// it away — so the status check, not a missing entry, is what has to
	// decide that this repeat is new work.
	j.closed = true
	j.refireOn = false
	if err := p.Handle(context.Background(), repeatAlert()); err != nil {
		t.Fatal(err)
	}
	if !j.refireOn {
		t.Fatal("the repeat was decided without consulting the ticket status")
	}
	if h.n != 2 {
		t.Fatalf("investigations = %d, want 2 (a Done ticket must not suppress the repeat)", h.n)
	}
	if len(j.refires) != 0 {
		t.Fatalf("a Done ticket must not collect refire notes, got %v", j.refires)
	}
}

// A ticket a human closed during the grace window is left alone: the relay
// drops its pending close instead of commenting on a finished ticket.
func TestPipelineSkipsCloseWhenTicketAlreadyDone(t *testing.T) {
	clk := &fakeClock{}
	j := &fakeJira{key: "JDWLABS-406"}
	p := newTestPipeline(t, &countingHolmes{}, j, 6*time.Hour, clk)

	if err := p.Handle(context.Background(), repeatAlert()); err != nil {
		t.Fatal(err)
	}
	if err := p.Handle(context.Background(), resolvedAlert()); err != nil {
		t.Fatal(err)
	}
	j.closed = true
	clk.advance(6 * time.Hour)
	p.SweepResolved(context.Background())

	if len(j.closes) != 0 {
		t.Fatalf("relay closed a ticket that was already Done: %v", j.closes)
	}
	if got := p.Counters().ticketsAutoClosed.Load(); got != 0 {
		t.Fatalf("ticketsAutoClosed = %d, want 0", got)
	}
}

// A failed close must stay pending: dropping it would leave the ticket open
// forever with nothing scheduled to try again.
func TestPipelineRetriesCloseAfterFailure(t *testing.T) {
	clk := &fakeClock{}
	j := &fakeJira{key: "JDWLABS-407", closeErr: errors.New("jira down")}
	p := newTestPipeline(t, &countingHolmes{}, j, 6*time.Hour, clk)

	if err := p.Handle(context.Background(), repeatAlert()); err != nil {
		t.Fatal(err)
	}
	if err := p.Handle(context.Background(), resolvedAlert()); err != nil {
		t.Fatal(err)
	}
	clk.advance(6 * time.Hour)
	p.SweepResolved(context.Background())
	if len(j.closes) != 0 {
		t.Fatalf("closes = %v, want none while Jira is failing", j.closes)
	}

	j.closeErr = nil
	p.SweepResolved(context.Background())
	if len(j.closes) != 1 {
		t.Fatalf("closes = %v, want the pending close retried once Jira recovers", j.closes)
	}
}

// A resolve for an alert this process never filed a ticket for is a no-op, not
// an error: the relay restarted, or the alert predates it.
func TestPipelineResolvedAlertWithoutTicketIsIgnored(t *testing.T) {
	clk := &fakeClock{}
	j := &fakeJira{key: "JDWLABS-408"}
	p := newTestPipeline(t, &countingHolmes{}, j, 6*time.Hour, clk)

	if err := p.Handle(context.Background(), resolvedAlert()); err != nil {
		t.Fatal(err)
	}
	if len(j.resolveNotes) != 0 || len(j.closes) != 0 {
		t.Fatalf("untracked resolve touched Jira: notes=%v closes=%v", j.resolveNotes, j.closes)
	}
	if j.upserts != 0 {
		t.Fatalf("resolve must never open a ticket, upserts = %d", j.upserts)
	}
}

// A resolve notification carrying no usable end time still starts the grace
// clock, from the moment the relay saw it.
func TestPipelineResolvedAlertWithoutEndsAtUsesReceiptTime(t *testing.T) {
	clk := &fakeClock{}
	j := &fakeJira{key: "JDWLABS-409"}
	p := newTestPipeline(t, &countingHolmes{}, j, 6*time.Hour, clk)

	if err := p.Handle(context.Background(), repeatAlert()); err != nil {
		t.Fatal(err)
	}
	a := resolvedAlert()
	a.EndsAt = ""
	if err := p.Handle(context.Background(), a); err != nil {
		t.Fatal(err)
	}
	if len(j.resolveNotes) != 1 || !j.resolveNotes[0].Equal(clk.now()) {
		t.Fatalf("resolve notes = %v, want one note at receipt time %v", j.resolveNotes, clk.now())
	}
	clk.advance(6 * time.Hour)
	p.SweepResolved(context.Background())
	if len(j.closes) != 1 {
		t.Fatalf("closes = %v, want the grace window to run from receipt time", j.closes)
	}
}

func TestCountersExposeTicketsAutoClosed(t *testing.T) {
	var c counters
	c.ticketsAutoClosed.Add(4)
	var buf bytes.Buffer
	c.writeTo(&buf)
	if !strings.Contains(buf.String(), "ai_sre_relay_tickets_auto_closed_total 4") {
		t.Fatalf("counter not exposed:\n%s", buf.String())
	}
}

// A resolve arriving after a restart, with no firing in between, has no
// in-process mapping to work from. The firing path already recovers the ticket
// from its fingerprint label; the resolve path must do the same, or the ticket
// the relay filed before the restart can never be closed by it.
func TestPipelineResolveAdoptsTicketAfterRestart(t *testing.T) {
	clk := &fakeClock{}
	j := &fakeJira{key: "JDWLABS-411", findKey: "JDWLABS-411"}
	p := newTestPipeline(t, &countingHolmes{}, j, 6*time.Hour, clk)

	if err := p.Handle(context.Background(), resolvedAlert()); err != nil {
		t.Fatal(err)
	}
	if j.finds != 1 {
		t.Fatalf("fingerprint searches = %d, want 1", j.finds)
	}
	if len(j.resolveNotes) != 1 {
		t.Fatalf("resolve notes = %v, want the adopted ticket to be noted", j.resolveNotes)
	}
	clk.advance(6 * time.Hour)
	p.SweepResolved(context.Background())
	if len(j.closes) != 1 {
		t.Fatalf("closes = %v, want the adopted ticket closed after the grace window", j.closes)
	}
}

// The search is a fallback, not the primary path: a fingerprint this process
// already tracks must not pay for a JQL round-trip on every resolve.
func TestPipelineResolveSkipsSearchWhenAlreadyTracked(t *testing.T) {
	clk := &fakeClock{}
	j := &fakeJira{key: "JDWLABS-412", findKey: "JDWLABS-412"}
	p := newTestPipeline(t, &countingHolmes{}, j, 6*time.Hour, clk)

	if err := p.Handle(context.Background(), repeatAlert()); err != nil {
		t.Fatal(err)
	}
	if err := p.Handle(context.Background(), resolvedAlert()); err != nil {
		t.Fatal(err)
	}
	if j.finds != 0 {
		t.Fatalf("fingerprint searches = %d, want 0 for a tracked fingerprint", j.finds)
	}
}

// A search that finds nothing must stay a no-op — the relay never opens a
// ticket for a resolve.
func TestPipelineResolveWithNoRecoverableTicketIsIgnored(t *testing.T) {
	clk := &fakeClock{}
	j := &fakeJira{key: "JDWLABS-413"}
	p := newTestPipeline(t, &countingHolmes{}, j, 6*time.Hour, clk)

	if err := p.Handle(context.Background(), resolvedAlert()); err != nil {
		t.Fatal(err)
	}
	if len(j.resolveNotes) != 0 || len(j.closes) != 0 || j.upserts != 0 {
		t.Fatalf("unrecoverable resolve touched Jira: notes=%v closes=%v upserts=%d",
			j.resolveNotes, j.closes, j.upserts)
	}
}

// A re-fire that lands after the sweep took its snapshot but before the close
// went out must still stop it: the alert is firing right now, and the snapshot
// is the only thing that says otherwise.
func TestPipelineSweepAbandonsCloseCancelledAfterSnapshot(t *testing.T) {
	clk := &fakeClock{}
	j := &fakeJira{key: "JDWLABS-414"}
	p := newTestPipeline(t, &countingHolmes{}, j, 6*time.Hour, clk)

	if err := p.Handle(context.Background(), repeatAlert()); err != nil {
		t.Fatal(err)
	}
	if err := p.Handle(context.Background(), resolvedAlert()); err != nil {
		t.Fatal(err)
	}
	clk.advance(6 * time.Hour)

	// IsOpen is the sweep's last read before it commits; a re-fire landing
	// exactly there is the narrowest form of the race.
	j.onIsOpen = func() { p.cancelPendingClose(repeatAlert()) }
	p.SweepResolved(context.Background())

	if len(j.closes) != 0 {
		t.Fatalf("closed a ticket whose alert re-fired mid-sweep: %v", j.closes)
	}
	if got := p.Counters().ticketsAutoClosed.Load(); got != 0 {
		t.Fatalf("ticketsAutoClosed = %d, want 0", got)
	}
}

// closeRacingJira re-fires the alert while the close itself is in flight,
// which is the one interleaving no pre-check can catch.
type closeRacingJira struct {
	*fakeJira
	race func()
}

func (j *closeRacingJira) Close(ctx context.Context, key IssueKey, at time.Time) error {
	j.race()
	return j.fakeJira.Close(ctx, key, at)
}

// Losing that race leaves a Done ticket for an alert that is firing. The relay
// has to put it back rather than wait for a human to notice.
func TestPipelineReopensTicketClosedByARacedRefire(t *testing.T) {
	clk := &fakeClock{}
	base := &fakeJira{key: "JDWLABS-415"}
	j := &closeRacingJira{fakeJira: base}
	p := newTestPipeline(t, &countingHolmes{}, j, 6*time.Hour, clk)
	j.race = func() { p.cancelPendingClose(repeatAlert()) }

	if err := p.Handle(context.Background(), repeatAlert()); err != nil {
		t.Fatal(err)
	}
	if err := p.Handle(context.Background(), resolvedAlert()); err != nil {
		t.Fatal(err)
	}
	clk.advance(6 * time.Hour)
	p.SweepResolved(context.Background())

	if len(base.closes) != 1 {
		t.Fatalf("closes = %v, want the close to have landed before the cancel was seen", base.closes)
	}
	if base.reopens != 1 {
		t.Fatalf("reopens = %d, want the raced close undone", base.reopens)
	}
}

// A resolve must never wait on a sweep that is already talking to Jira: the
// dispatcher worker running it is one of four, and a slow Jira would otherwise
// park them all and turn alerts into 503s.
func TestPipelineResolveDoesNotBlockOnARunningSweep(t *testing.T) {
	clk := &fakeClock{}
	j := &fakeJira{key: "JDWLABS-416"}
	p := newTestPipeline(t, &countingHolmes{}, j, time.Duration(0), clk)

	if err := p.Handle(context.Background(), repeatAlert()); err != nil {
		t.Fatal(err)
	}

	p.sweepMu.Lock()
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = p.Handle(context.Background(), resolvedAlert())
	}()
	blocked := false
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		blocked = true
	}
	// Release before failing and wait for the goroutine either way: a late
	// write from it must not race the test's teardown.
	p.sweepMu.Unlock()
	<-done
	if blocked {
		t.Fatal("resolve handling blocked behind an in-progress sweep")
	}

	if len(j.resolveNotes) != 1 {
		t.Fatalf("resolve notes = %v, want the resolve recorded despite the busy sweep", j.resolveNotes)
	}
	// The ticker's sweep is what picks the skipped close up.
	p.SweepResolved(context.Background())
	if len(j.closes) != 1 {
		t.Fatalf("closes = %v, want the deferred close picked up by the next sweep", j.closes)
	}
}

// The per-alert deadline is usually spent by the time an alert reaches here,
// and a resolve that arrives on a dead context must still be recorded — the
// alternative is a ticket that never closes and a runbook line pointing at a
// Jira outage that never happened.
func TestPipelineResolveSurvivesDeadContext(t *testing.T) {
	clk := &fakeClock{}
	j := &fakeJira{key: "JDWLABS-417"}
	p := newTestPipeline(t, &countingHolmes{}, j, time.Duration(0), clk)

	if err := p.Handle(context.Background(), repeatAlert()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := p.Handle(ctx, resolvedAlert()); err != nil {
		t.Fatal(err)
	}
	if len(j.resolveNotes) != 1 || j.noteResolvedCtxErr != nil {
		t.Fatalf("resolve note must run on a live context: notes=%v ctxErr=%v",
			j.resolveNotes, j.noteResolvedCtxErr)
	}
	if len(j.closes) != 1 || j.closeCtxErr != nil {
		t.Fatalf("close must run on a live context: closes=%v ctxErr=%v", j.closes, j.closeCtxErr)
	}
}

// A transition that keeps failing must not re-post the closing comment every
// sweep. Before this, a workflow with no Done transition left a ticket
// collecting one "Closed automatically" comment per sweep interval, forever.
func TestPipelineFailedCloseDoesNotRepeatTheClosingComment(t *testing.T) {
	clk := &fakeClock{}
	j := &fakeJira{key: "JDWLABS-418", closeErr: errors.New("no Done transition available")}
	p := newTestPipeline(t, &countingHolmes{}, j, 6*time.Hour, clk)

	if err := p.Handle(context.Background(), repeatAlert()); err != nil {
		t.Fatal(err)
	}
	if err := p.Handle(context.Background(), resolvedAlert()); err != nil {
		t.Fatal(err)
	}
	clk.advance(6 * time.Hour)
	for range 3 {
		p.SweepResolved(context.Background())
		clk.advance(10 * time.Minute)
	}
	if j.closeComments != 0 {
		t.Fatalf("closing comments = %d, want 0 while the transition keeps failing", j.closeComments)
	}
	if got := p.Counters().ticketsAutoClosed.Load(); got != 0 {
		t.Fatalf("ticketsAutoClosed = %d, want 0", got)
	}
}

// The reverse failure — the ticket transitioned but its explanation did not
// post — is a closed ticket, not a pending one. Retrying would transition a
// Done ticket again, so it is counted and dropped, loudly.
func TestPipelineCloseWithoutNoteIsNotRetried(t *testing.T) {
	clk := &fakeClock{}
	var buf bytes.Buffer
	j := &fakeJira{key: "JDWLABS-419", closeCommentErr: errors.New("comment rejected")}
	clk.t = mustParseTime(t, testResolveTime)
	p := NewPipeline(&countingHolmes{}, fakePatcher{p: nil}, j, &fakeGH{}, &fakeDiscord{},
		slog.New(slog.NewJSONHandler(&buf, nil)), withCloseGrace(6*time.Hour), withClock(clk.now))

	if err := p.Handle(context.Background(), repeatAlert()); err != nil {
		t.Fatal(err)
	}
	if err := p.Handle(context.Background(), resolvedAlert()); err != nil {
		t.Fatal(err)
	}
	clk.advance(6 * time.Hour)
	p.SweepResolved(context.Background())
	p.SweepResolved(context.Background())

	if len(j.closes) != 1 {
		t.Fatalf("closes = %v, want the Done transition attempted exactly once", j.closes)
	}
	if got := p.Counters().ticketsAutoClosed.Load(); got != 1 {
		t.Fatalf("ticketsAutoClosed = %d, want 1 (the ticket is closed)", got)
	}
	if !strings.Contains(buf.String(), `"level":"ERROR"`) {
		t.Fatalf("a ticket closed without its explanation must be loud:\n%s", buf.String())
	}
}

// A repeat that races the sweep forgetting the fingerprint must not write the
// entry back: an entry with no issue key is unusable and holds a tracking slot
// that a real alert then cannot have.
func TestPipelineRepeatDoesNotResurrectAForgottenFingerprint(t *testing.T) {
	clk := &fakeClock{}
	h := &countingHolmes{}
	j := &fakeJira{key: "JDWLABS-420"}
	p := newTestPipeline(t, h, j, 6*time.Hour, clk)

	if err := p.Handle(context.Background(), repeatAlert()); err != nil {
		t.Fatal(err)
	}
	// Drop the mapping at the exact point the repeat path has decided the
	// ticket is open and is about to number the refire.
	j.onIsOpen = func() {
		j.onIsOpen = nil
		p.forget(repeatAlert())
	}
	if err := p.Handle(context.Background(), repeatAlert()); err != nil {
		t.Fatal(err)
	}

	p.mu.Lock()
	f, tracked := p.active[repeatAlert().Fingerprint]
	p.mu.Unlock()
	if tracked && f.issue == "" {
		t.Fatalf("repeat wrote back an entry with no issue: %+v", f)
	}
	if h.n != 2 {
		t.Fatalf("investigations = %d, want 2 (an untracked repeat is new work)", h.n)
	}
}

// The resolve is the one alert state a human watching Discord never hears
// about otherwise: the relay posts the investigation, then goes quiet.
func TestPipelineResolveNotifiesDiscord(t *testing.T) {
	clk := &fakeClock{}
	clk.t = mustParseTime(t, testResolveTime)
	j := &fakeJira{key: "JDWLABS-421"}
	d := &fakeDiscord{}
	p := NewPipeline(&countingHolmes{}, fakePatcher{p: nil}, j, &fakeGH{}, d, silentLogger(),
		withCloseGrace(6*time.Hour), withClock(clk.now))

	if err := p.Handle(context.Background(), repeatAlert()); err != nil {
		t.Fatal(err)
	}
	d.called, d.notifies = false, 0
	if err := p.Handle(context.Background(), resolvedAlert()); err != nil {
		t.Fatal(err)
	}
	if d.notifies != 1 {
		t.Fatalf("discord notifications on resolve = %d, want 1", d.notifies)
	}
	if d.lastAlert.Status != statusResolved {
		t.Fatalf("resolve notice sent with status %q", d.lastAlert.Status)
	}
	if !strings.Contains(d.lastAnalysis.RootCause, "JDWLABS-421") {
		t.Fatalf("resolve notice omits the ticket: %q", d.lastAnalysis.RootCause)
	}

	// A repeated resolve notification is not news.
	if err := p.Handle(context.Background(), resolvedAlert()); err != nil {
		t.Fatal(err)
	}
	if d.notifies != 1 {
		t.Fatalf("discord notifications = %d, want 1: a repeated resolve is not news", d.notifies)
	}
}

// resolvingHolmes delivers a resolve notification while the investigation it
// interrupts is still running. The dispatcher allows exactly this: firing and
// resolved for one fingerprint are separate in-flight keys.
type resolvingHolmes struct {
	p       *Pipeline
	resolve Alert
	n       int
}

func (h *resolvingHolmes) Investigate(ctx context.Context, _ Alert) (Analysis, error) {
	h.n++
	if h.n == 1 {
		_ = h.p.Handle(ctx, h.resolve)
	}
	return Analysis{RootCause: "x"}, nil
}

// The investigation finishing must not erase the pending close the resolve
// started. It used to: the ticket was commented and announced as resolving,
// then the close was dropped with no log, no metric and no way to tell from
// the outside that the ticket would now stay open forever.
func TestPipelineResolveDuringInvestigationSurvivesItsCompletion(t *testing.T) {
	clk := &fakeClock{}
	j := &fakeJira{key: "JDWLABS-430"}
	h := &resolvingHolmes{resolve: resolvedAlert()}
	p := newTestPipeline(t, h, j, 6*time.Hour, clk)
	h.p = p

	// A prior episode leaves the fingerprint mapped, so the resolve lands on
	// the mapping the in-flight investigation is about to overwrite.
	first := repeatAlert()
	if err := p.Handle(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	h.n = 0
	second := repeatAlert()
	second.StartsAt = "2026-07-28T05:00:00Z"
	if err := p.Handle(context.Background(), second); err != nil {
		t.Fatal(err)
	}
	if len(j.resolveNotes) != 1 {
		t.Fatalf("resolve notes = %v, want the mid-investigation resolve recorded", j.resolveNotes)
	}

	clk.advance(24 * time.Hour)
	p.SweepResolved(context.Background())
	if len(j.closes) != 1 {
		t.Fatalf("closes = %v, want the pending close to survive the investigation completing", j.closes)
	}
}

// Same loss through the adoption path: the resolve arrives before this process
// has tracked anything, recovers the ticket from Jira, and the investigation
// that was already running then overwrites it.
func TestPipelineAdoptedResolveSurvivesAConcurrentInvestigation(t *testing.T) {
	clk := &fakeClock{}
	j := &fakeJira{key: "JDWLABS-431", findKey: "JDWLABS-431"}
	h := &resolvingHolmes{resolve: resolvedAlert()}
	p := newTestPipeline(t, h, j, 6*time.Hour, clk)
	h.p = p

	if err := p.Handle(context.Background(), repeatAlert()); err != nil {
		t.Fatal(err)
	}
	if j.finds != 1 {
		t.Fatalf("fingerprint searches = %d, want the untracked resolve to have adopted", j.finds)
	}

	clk.advance(24 * time.Hour)
	p.SweepResolved(context.Background())
	if len(j.closes) != 1 {
		t.Fatalf("closes = %v, want the adopted pending close to survive the investigation", j.closes)
	}
}

// The repeat counter belongs to a firing episode. A new episode starts its
// numbering again rather than continuing the last one's.
func TestPipelineNewEpisodeRestartsRepeatNumbering(t *testing.T) {
	clk := &fakeClock{}
	j := &fakeJira{key: "JDWLABS-432"}
	p := newTestPipeline(t, &countingHolmes{}, j, 6*time.Hour, clk)

	first := repeatAlert()
	if err := p.Handle(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	if err := p.Handle(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	second := repeatAlert()
	second.StartsAt = "2026-07-28T05:00:00Z"
	if err := p.Handle(context.Background(), second); err != nil {
		t.Fatal(err)
	}
	if err := p.Handle(context.Background(), second); err != nil {
		t.Fatal(err)
	}
	if len(j.refires) != 2 || j.refires[0] != 1 || j.refires[1] != 1 {
		t.Fatalf("refire notes = %v, want [1 1]: each episode numbers its own repeats", j.refires)
	}
}

// A close that raced a re-fire is not an auto-close: the ticket is open again
// at the end of it, and counting it would overstate what the relay finished.
func TestPipelineRacedCloseIsCountedAsAReopenNotAClose(t *testing.T) {
	clk := &fakeClock{}
	base := &fakeJira{key: "JDWLABS-433"}
	j := &closeRacingJira{fakeJira: base}
	p := newTestPipeline(t, &countingHolmes{}, j, 6*time.Hour, clk)
	j.race = func() { p.cancelPendingClose(repeatAlert()) }

	if err := p.Handle(context.Background(), repeatAlert()); err != nil {
		t.Fatal(err)
	}
	if err := p.Handle(context.Background(), resolvedAlert()); err != nil {
		t.Fatal(err)
	}
	clk.advance(6 * time.Hour)
	p.SweepResolved(context.Background())

	if got := p.Counters().ticketsAutoClosed.Load(); got != 0 {
		t.Fatalf("ticketsAutoClosed = %d, want 0: the ticket was reopened", got)
	}
	if got := p.Counters().ticketReopens.Load(); got != 1 {
		t.Fatalf("ticketReopens = %d, want 1", got)
	}
}

type reopenFailingJira struct {
	*fakeJira
	race func()
}

func (j *reopenFailingJira) Close(ctx context.Context, key IssueKey, at time.Time) error {
	j.race()
	return j.fakeJira.Close(ctx, key, at)
}

func (j *reopenFailingJira) Reopen(context.Context, IssueKey) error {
	return errors.New("jira down")
}

// A reopen that fails is terminal — the ticket stays Done while its alert
// fires — so it needs its own counter to alert on.
func TestPipelineFailedReopenIsCounted(t *testing.T) {
	clk := &fakeClock{}
	base := &fakeJira{key: "JDWLABS-434"}
	j := &reopenFailingJira{fakeJira: base}
	p := newTestPipeline(t, &countingHolmes{}, j, 6*time.Hour, clk)
	j.race = func() { p.cancelPendingClose(repeatAlert()) }

	if err := p.Handle(context.Background(), repeatAlert()); err != nil {
		t.Fatal(err)
	}
	if err := p.Handle(context.Background(), resolvedAlert()); err != nil {
		t.Fatal(err)
	}
	clk.advance(6 * time.Hour)
	p.SweepResolved(context.Background())

	if got := p.Counters().ticketReopensFailed.Load(); got != 1 {
		t.Fatalf("ticketReopensFailed = %d, want 1", got)
	}
	if got := p.Counters().ticketsAutoClosed.Load(); got != 0 {
		t.Fatalf("ticketsAutoClosed = %d, want 0", got)
	}
}

// A sweep must stop when its context is done, or a shutdown waits on every
// remaining ticket's Jira round-trip.
func TestPipelineSweepStopsOnCancelledContext(t *testing.T) {
	clk := &fakeClock{}
	j := &fakeJira{key: "JDWLABS-435"}
	p := newTestPipeline(t, &countingHolmes{}, j, 6*time.Hour, clk)

	if err := p.Handle(context.Background(), repeatAlert()); err != nil {
		t.Fatal(err)
	}
	if err := p.Handle(context.Background(), resolvedAlert()); err != nil {
		t.Fatal(err)
	}
	clk.advance(6 * time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p.SweepResolved(ctx)

	if len(j.closes) != 0 {
		t.Fatalf("closes = %v, want none: the sweep's context was already done", j.closes)
	}
	if j.refireOn {
		t.Fatal("sweep called Jira despite a cancelled context")
	}
}

// A tracking table at its ceiling silently stopped tracking new fingerprints.
// Silent is the problem: every alert past the ceiling loses repeat suppression
// and auto-close at once, and nothing said so.
func TestPipelineFullTrackingTableIsVisible(t *testing.T) {
	clk := &fakeClock{}
	var buf bytes.Buffer
	clk.t = mustParseTime(t, testResolveTime)
	j := &fakeJira{key: "JDWLABS-436"}
	p := NewPipeline(&countingHolmes{}, fakePatcher{p: nil}, j, &fakeGH{}, &fakeDiscord{},
		slog.New(slog.NewJSONHandler(&buf, nil)), withCloseGrace(6*time.Hour), withClock(clk.now))

	p.mu.Lock()
	for i := range maxTrackedFirings {
		p.active[fmt.Sprintf("filler-%d", i)] = firing{issue: "JDWLABS-1"}
	}
	p.mu.Unlock()

	if err := p.Handle(context.Background(), repeatAlert()); err != nil {
		t.Fatal(err)
	}
	if got := p.Counters().fingerprintsUntracked.Load(); got != 1 {
		t.Fatalf("fingerprintsUntracked = %d, want 1", got)
	}
	if !strings.Contains(buf.String(), `"level":"WARN"`) {
		t.Fatalf("a full tracking table must be logged:\n%s", buf.String())
	}
}

func TestCountersExposeReopenAndTrackingOutcomes(t *testing.T) {
	var c counters
	c.ticketReopens.Add(2)
	c.ticketReopensFailed.Add(1)
	c.fingerprintsUntracked.Add(7)
	var buf bytes.Buffer
	c.writeTo(&buf)
	for _, want := range []string{
		`ai_sre_relay_ticket_reopens_total{result="ok"} 2`,
		`ai_sre_relay_ticket_reopens_total{result="failed"} 1`,
		"ai_sre_relay_fingerprints_untracked_total 7",
	} {
		if !strings.Contains(buf.String(), want) {
			t.Fatalf("counter not exposed: %q\n%s", want, buf.String())
		}
	}
}
