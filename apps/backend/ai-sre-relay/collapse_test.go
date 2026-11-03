package main

import (
	"context"
	"strings"
	"testing"
	"time"
)

func sev(fp, name, ns, severity string) Alert {
	return Alert{Status: statusFiring, Fingerprint: fp, StartsAt: "2026-07-28T03:02:59Z",
		Labels: map[string]string{"alertname": name, "namespace": ns, "severity": severity}}
}

func TestCollapseFiringSeverityAndSiblings(t *testing.T) {
	cases := []struct {
		name         string
		in           []Alert
		wantReps     string
		wantSiblings string // "rep:sib,sib;rep:sib"
	}{
		{"a critical after a warning is not absorbed", []Alert{
			sev("w", "X", "db", "warning"), sev("c", "X", "db", "critical"),
		}, "w,c", ""},
		{"same severity collapses and keeps siblings", []Alert{
			sev("a", "X", "db", "warning"), sev("b", "X", "db", "warning"), sev("c", "X", "db", "warning"),
		}, "a", "a:b,c"},
		{"severities collapse independently", []Alert{
			sev("a", "X", "db", "warning"), sev("c1", "X", "db", "critical"), sev("b", "X", "db", "warning"), sev("c2", "X", "db", "critical"),
		}, "a,c1", "a:b;c1:c2"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var reps, sibs []string
			for _, a := range collapseFiring(tc.in) {
				reps = append(reps, a.Fingerprint)
				if len(a.Siblings) > 0 {
					var fps []string
					for _, s := range a.Siblings {
						fps = append(fps, s.Fingerprint)
					}
					sibs = append(sibs, a.Fingerprint+":"+strings.Join(fps, ","))
				}
			}
			if got := strings.Join(reps, ","); got != tc.wantReps {
				t.Fatalf("reps = %q, want %q", got, tc.wantReps)
			}
			if got := strings.Join(sibs, ";"); got != tc.wantSiblings {
				t.Fatalf("siblings = %q, want %q", got, tc.wantSiblings)
			}
		})
	}
}

func resolvedFP(fp string) Alert {
	a := sev(fp, "X", "db", "warning")
	a.Status = statusResolved
	a.EndsAt = testResolveTime
	return a
}

// The ticket belongs to the whole collapsed group: the first pod clearing
// while the others still fire must leave it open, and the last one to clear
// closes it.
func TestPipelineKeepsTicketOpenWhileCollapsedSiblingsFire(t *testing.T) {
	clk := &fakeClock{}
	j := &fakeJira{key: "JDWLABS-500"}
	p := newTestPipeline(t, &countingHolmes{}, j, time.Hour, clk)
	ctx := context.Background()

	rep := sev("a", "X", "db", "warning")
	rep.Siblings = []Alert{sev("b", "X", "db", "warning"), sev("c", "X", "db", "warning")}
	if err := p.Handle(ctx, rep); err != nil {
		t.Fatal(err)
	}

	if err := p.Handle(ctx, resolvedFP("a")); err != nil {
		t.Fatal(err)
	}
	clk.advance(2 * time.Hour)
	p.SweepResolved(ctx)
	if len(j.closes) != 0 {
		t.Fatalf("ticket closed while siblings still fire: %v", j.closes)
	}

	// A sibling's own resolve must find the ticket rather than "nothing to close".
	if err := p.Handle(ctx, resolvedFP("b")); err != nil {
		t.Fatal(err)
	}
	if j.finds != 0 {
		t.Fatalf("sibling resolve fell back to a Jira lookup (%d); it was not tracked", j.finds)
	}
	p.SweepResolved(ctx)
	if len(j.closes) != 0 {
		t.Fatalf("ticket closed while c still fires: %v", j.closes)
	}

	if err := p.Handle(ctx, resolvedFP("c")); err != nil {
		t.Fatal(err)
	}
	clk.advance(2 * time.Hour)
	p.SweepResolved(ctx)
	if len(j.closes) != 1 {
		t.Fatalf("closes = %v, want exactly one once every fingerprint resolved", j.closes)
	}
}

// A sibling that re-fires inside the grace window must cancel its own pending
// close, even though the webhook only investigated the first alert.
func TestPipelineSiblingRefireCancelsItsPendingClose(t *testing.T) {
	clk := &fakeClock{}
	j := &fakeJira{key: "JDWLABS-501"}
	p := newTestPipeline(t, &countingHolmes{}, j, time.Hour, clk)
	ctx := context.Background()

	rep := sev("a", "X", "db", "warning")
	rep.Siblings = []Alert{sev("b", "X", "db", "warning")}
	_ = p.Handle(ctx, rep)
	_ = p.Handle(ctx, resolvedFP("b"))
	_ = p.Handle(ctx, resolvedFP("a"))

	// b fires again, collapsed behind a new representative.
	again := sev("a", "X", "db", "warning")
	again.Siblings = []Alert{sev("b", "X", "db", "warning")}
	_ = p.Handle(ctx, again)
	clk.advance(2 * time.Hour)
	p.SweepResolved(ctx)
	if len(j.closes) != 0 {
		t.Fatalf("ticket closed although the group re-fired: %v", j.closes)
	}
}
