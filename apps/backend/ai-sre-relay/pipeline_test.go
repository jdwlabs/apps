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
	closeErr     error
}

func (f *fakeJira) Upsert(context.Context, Alert, Analysis) (IssueKey, error) {
	f.called = true
	f.upserts++
	return f.key, nil
}

func (f *fakeJira) IsOpen(context.Context, IssueKey) (bool, error) {
	f.refireOn = true
	if f.openErr != nil {
		return false, f.openErr
	}
	return !f.closed, nil
}

func (f *fakeJira) NoteRefire(_ context.Context, _ IssueKey, count int) error {
	f.refires = append(f.refires, count)
	return nil
}

func (f *fakeJira) NoteResolved(_ context.Context, _ IssueKey, at time.Time) error {
	f.resolveNotes = append(f.resolveNotes, at)
	return nil
}

func (f *fakeJira) Close(_ context.Context, _ IssueKey, at time.Time) error {
	if f.closeErr != nil {
		return f.closeErr
	}
	f.closes = append(f.closes, at)
	f.closed = true
	return nil
}

type fakeGH struct{ called bool }

func (f *fakeGH) OpenPR(context.Context, Patch, IssueKey) (PRLink, error) {
	f.called = true
	return "http://pr/1", nil
}

type fakeDiscord struct {
	called bool
	pr     *PRLink
}

func (f *fakeDiscord) Notify(_ context.Context, _ Alert, _ Analysis, _ IssueKey, pr *PRLink, _ *Patch) error {
	f.called = true
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
	for range 3 {
		if err := p.Handle(context.Background(), resolvedAlert()); err != nil {
			t.Fatal(err)
		}
		clk.advance(time.Hour)
	}
	if len(j.resolveNotes) != 1 {
		t.Fatalf("resolve notes = %v, want exactly one", j.resolveNotes)
	}
	if len(j.closes) != 0 {
		t.Fatalf("ticket closed before the grace window elapsed: %v", j.closes)
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
	if err := p.Handle(context.Background(), resolvedAlert()); err != nil {
		t.Fatal(err)
	}
	clk.advance(6 * time.Hour)
	p.SweepResolved(context.Background())

	if err := p.Handle(context.Background(), repeatAlert()); err != nil {
		t.Fatal(err)
	}
	if h.n != 2 {
		t.Fatalf("investigations = %d, want 2 (a Done ticket must not suppress the repeat)", h.n)
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
