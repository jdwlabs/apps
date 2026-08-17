package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"
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
