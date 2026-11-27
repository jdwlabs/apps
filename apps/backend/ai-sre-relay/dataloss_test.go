package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func lossAlert(name, fp, startsAt, summary string) Alert {
	return Alert{
		Fingerprint: fp, StartsAt: startsAt, Status: statusFiring,
		Labels:      map[string]string{"alertname": name, "namespace": "minecraft-fwb"},
		Annotations: map[string]string{"summary": summary},
	}
}

func TestDatalossPolicy(t *testing.T) {
	p := defaultDatalossPolicy()
	cases := []struct {
		name  string
		alert Alert
		want  bool
	}{
		{"chunks lost", lossAlert("JdwillmsenMinecraftWorldChunksLost", "a", "", ""), true},
		{"unprefixed name", lossAlert("WorldArchiveShrank", "a", "", ""), true},
		{"guard would hold", lossAlert("JdwillmsenMinecraftWorldGuardWouldHold", "a", "", ""), true},
		{"unrelated alert", lossAlert("TempoNoSpansReceived", "a", "", ""), false},
		{"opt-in label", Alert{Labels: map[string]string{"alertname": "Custom", "dataloss": "true"}}, true},
		{"opt-in annotation", Alert{Labels: map[string]string{"alertname": "Custom"}, Annotations: map[string]string{"dataloss": "True"}}, true},
		{"label false", Alert{Labels: map[string]string{"alertname": "Custom", "dataloss": "false"}}, false},
		{"no name", Alert{}, false},
	}
	for _, tc := range cases {
		if got := p.isDataLoss(tc.alert); got != tc.want {
			t.Errorf("%s: isDataLoss = %v, want %v", tc.name, got, tc.want)
		}
	}
	custom := newDatalossPolicy([]string{"OnlyThis"}, "", 0)
	if custom.isDataLoss(lossAlert("WorldChunksLost", "a", "", "")) {
		t.Error("a configured list must replace the defaults")
	}
}

func TestDatalossValueAndGrowth(t *testing.T) {
	p := defaultDatalossPolicy()
	vals := []struct {
		summary string
		want    float64
		ok      bool
	}{
		{"Minecraft FWB has lost 14105 world chunks", 14105, true},
		{"Minecraft FWB has lost 1,403 world chunks", 1403, true},
		{"has lost 1.4105e+04 world chunks", 14105, true},
		{"no number here", 0, false},
		{"", 0, false},
	}
	for _, tc := range vals {
		got, ok := p.value(lossAlert("WorldChunksLost", "a", "", tc.summary))
		if ok != tc.ok || got != tc.want {
			t.Errorf("value(%q) = %v,%v want %v,%v", tc.summary, got, ok, tc.want, tc.ok)
		}
	}
	grow := []struct {
		prev, next float64
		want       bool
	}{
		{1403, 14105, true}, {1403, 1404, false}, {1403, 1544, true}, {1403, 1403, false}, {14105, 1403, false}, {0, 1, true},
	}
	for _, tc := range grow {
		if got := p.grew(tc.prev, tc.next); got != tc.want {
			t.Errorf("grew(%v,%v) = %v, want %v", tc.prev, tc.next, got, tc.want)
		}
	}
	if got := formatValue(14105); got != "14,105" {
		t.Errorf("formatValue = %q", got)
	}
}

// A day-by-day replay of the November 2026 world loss: the ticket opens, the
// evidence window expires and the alert resolves, the loss grows tenfold under
// the same alert identity, and nothing is ever closed by the relay.
func TestDatalossNovemberTimeline(t *testing.T) {
	const (
		lostName  = "JdwillmsenMinecraftWorldChunksLost"
		guardName = "JdwillmsenMinecraftWorldGuardWouldHold"
		started   = "2026-11-13T21:30:00Z"
	)
	ctx := context.Background()
	clk := &fakeClock{}
	j := &fakeJira{key: "OPS-385"}
	h := &countingHolmes{}
	d := &fakeDiscord{}
	p := NewPipeline(h, fakePatcher{}, j, &fakeGH{}, d, silentLogger(),
		withCloseGrace(6*time.Hour), withClock(clk.now))
	clk.t = mustParseTime(t, started)

	step := func(name string, at string, a Alert, wantInvestigations int, check func()) {
		t.Helper()
		clk.t = mustParseTime(t, at)
		if err := p.Handle(ctx, a); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if h.n != wantInvestigations {
			t.Fatalf("%s: investigations = %d, want %d", name, h.n, wantInvestigations)
		}
		if check != nil {
			check()
		}
	}

	lost := func(n string) Alert {
		return lossAlert(lostName, "fp-lost", started, "Minecraft FWB has lost "+n+" world chunks")
	}
	guard := lossAlert(guardName, "fp-guard", "2026-11-16T06:45:00Z", "world guard would hold")

	step("open at T", started, lost("1403"), 1, nil)
	step("same value repeats are skipped", "2026-11-14T03:30:00Z", lost("1403"), 1, func() {
		if len(j.refires) != 1 {
			t.Fatalf("refire notes = %v, want one", j.refires)
		}
	})

	step("guard alert opens", "2026-11-16T06:51:00Z", guard, 2, nil)
	g := guard
	g.Status, g.EndsAt = statusResolved, "2026-11-16T13:22:00Z"
	step("window expires, guard resolves", "2026-11-16T13:22:00Z", g, 2, func() {
		if len(j.datalossNotes) != 1 || !j.datalossNotes[0].Equal(mustParseTime(t, g.EndsAt)) {
			t.Fatalf("data-loss resolve notes = %v", j.datalossNotes)
		}
		if len(j.resolveNotes) != 0 {
			t.Fatalf("data-loss resolve took the auto-close path: %v", j.resolveNotes)
		}
	})
	clk.advance(30 * 24 * time.Hour)
	p.SweepResolved(ctx)
	if len(j.closes) != 0 {
		t.Fatalf("data-loss ticket auto-closed: %v", j.closes)
	}
	// A duplicate resolve notification must not stack another note.
	step("duplicate resolve", "2026-12-16T13:22:00Z", g, 2, func() {
		if len(j.datalossNotes) != 1 {
			t.Fatalf("duplicate resolve noted again: %v", j.datalossNotes)
		}
	})

	step("loss grows 1,403 -> 14,105", "2026-11-16T06:50:00Z", lost("14105"), 3, func() {
		if !strings.Contains(d.lastAnalysis.RootCause, "from 1,403 to 14,105") {
			t.Fatalf("growth not reported: %q", d.lastAnalysis.RootCause)
		}
		if len(j.refires) != 1 {
			t.Fatalf("a grown value was recorded as a skipped repeat: %v", j.refires)
		}
	})
	step("same new value repeats are skipped again", "2026-11-16T10:50:00Z", lost("14105"), 3, func() {
		if len(j.refires) != 2 {
			t.Fatalf("refire notes = %v, want two", j.refires)
		}
	})

	r := lost("14105")
	r.Status, r.EndsAt = statusResolved, "2026-11-20T16:00:00Z"
	step("loss alert resolves", "2026-11-20T16:00:00Z", r, 3, nil)
	clk.advance(30 * 24 * time.Hour)
	p.SweepResolved(ctx)
	if len(j.closes) != 0 || j.closed {
		t.Fatalf("ticket closed after resolve: closes=%v", j.closes)
	}
	if len(j.datalossNotes) != 2 {
		t.Fatalf("resolve notes = %v, want one per alert", j.datalossNotes)
	}
}

// The same lifecycle for an ordinary alert still closes, so the class is the
// only thing that changed.
func TestDatalossDoesNotChangeOrdinaryAutoClose(t *testing.T) {
	clk := &fakeClock{}
	j := &fakeJira{key: "JDWLABS-9"}
	p := newTestPipeline(t, &countingHolmes{}, j, time.Hour, clk)
	if err := p.Handle(context.Background(), repeatAlert()); err != nil {
		t.Fatal(err)
	}
	if err := p.Handle(context.Background(), resolvedAlert()); err != nil {
		t.Fatal(err)
	}
	clk.advance(2 * time.Hour)
	p.SweepResolved(context.Background())
	if len(j.closes) != 1 || len(j.datalossNotes) != 0 {
		t.Fatalf("closes=%v datalossNotes=%v", j.closes, j.datalossNotes)
	}
}

// A data-loss resolve that arrives after a restart finds its ticket in Jira
// and still leaves it open.
func TestDatalossResolveAfterRestartAdoptsTicket(t *testing.T) {
	clk := &fakeClock{}
	j := &fakeJira{key: "OPS-1", findKey: "OPS-404"}
	p := newTestPipeline(t, &countingHolmes{}, j, time.Hour, clk)
	a := lossAlert("JdwillmsenMinecraftWorldArchiveShrank", "fp-a", "2026-11-17T04:20:00Z", "")
	a.Status, a.EndsAt = statusResolved, "2026-11-20T16:00:00Z"
	if err := p.Handle(context.Background(), a); err != nil {
		t.Fatal(err)
	}
	clk.advance(48 * time.Hour)
	p.SweepResolved(context.Background())
	if j.finds != 1 || len(j.datalossNotes) != 1 || len(j.closes) != 0 {
		t.Fatalf("finds=%d notes=%v closes=%v", j.finds, j.datalossNotes, j.closes)
	}
}

type issueCall struct{ method, path, body string }

func newEscalationServer(t *testing.T, priority string, calls *[]issueCall) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		*calls = append(*calls, issueCall{r.Method, r.URL.Path, string(b)})
		switch {
		case r.Method == http.MethodGet && r.URL.Query().Get("fields") == "priority":
			_, _ = w.Write([]byte(`{"fields":{"priority":{"name":"` + priority + `"}}}`))
		case r.Method == http.MethodPut:
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/comment"):
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/search/jql"), strings.Contains(r.URL.Path, "/search"):
			_, _ = w.Write([]byte(`{"issues":[{"key":"OPS-385","fields":{"status":{"statusCategory":{"key":"new"}}}}]}`))
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"fields":{"status":{"statusCategory":{"key":"new"}}}}`))
		}
	}))
}

func TestJiraEscalatesDatalossTickets(t *testing.T) {
	cases := []struct {
		name         string
		alert        Alert
		priority     string
		wantPut      bool
		wantPriority bool
	}{
		{"medium is raised to High", lossAlert("WorldChunksLost", "fp", "", ""), "Medium", true, true},
		{"already Highest is not lowered", lossAlert("WorldChunksLost", "fp", "", ""), "Highest", true, false},
		{"ordinary alert is untouched", lossAlert("TempoNoSpansReceived", "fp", "", ""), "Medium", false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var calls []issueCall
			srv := newEscalationServer(t, tc.priority, &calls)
			defer srv.Close()
			j := NewJiraClient(srv.URL, "e@x", "tok", "OPS", "Task", srv.Client())
			if _, err := j.Upsert(context.Background(), tc.alert, Analysis{RootCause: "rc"}); err != nil {
				t.Fatal(err)
			}
			// A second upsert must not escalate again.
			if _, err := j.Upsert(context.Background(), tc.alert, Analysis{RootCause: "rc2"}); err != nil {
				t.Fatal(err)
			}
			puts := 0
			for _, c := range calls {
				if c.method != http.MethodPut {
					continue
				}
				puts++
				var body struct {
					Fields map[string]map[string]string `json:"fields"`
					Update struct {
						Labels []map[string]string `json:"labels"`
					} `json:"update"`
				}
				if err := json.Unmarshal([]byte(c.body), &body); err != nil {
					t.Fatal(err)
				}
				if got := body.Fields["priority"]["name"] == "High"; got != tc.wantPriority {
					t.Errorf("priority set = %v, want %v (%s)", got, tc.wantPriority, c.body)
				}
				if !strings.Contains(c.body, `"dataloss"`) || !strings.Contains(c.body, `"needs-human"`) {
					t.Errorf("labels missing: %s", c.body)
				}
			}
			if (puts == 1) != tc.wantPut || puts > 1 {
				t.Errorf("PUT count = %d, wantPut %v", puts, tc.wantPut)
			}
		})
	}
}

func TestJiraNoteDataLossResolvedLeavesTicketOpen(t *testing.T) {
	var log commentLog
	srv := newCommentRecorder(t, &log)
	defer srv.Close()
	j, _ := newClockedJira(srv)
	at := time.Date(2026, 11, 16, 13, 22, 0, 0, time.UTC)
	if err := j.NoteDataLossResolved(context.Background(), "OPS-404", at); err != nil {
		t.Fatal(err)
	}
	if len(log.bodies) != 1 {
		t.Fatalf("comments = %v, want exactly one and no transition", log.bodies)
	}
}
