package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
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
	key    IssueKey
	called bool
}

func (f *fakeJira) Upsert(context.Context, Alert, Analysis) (IssueKey, error) {
	f.called = true
	return f.key, nil
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
	patch := &Patch{Repo: "a/b", FilePath: "f", NewContent: "c", Branch: "fix/x", Rationale: "r", Confidence: 0.9}
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
