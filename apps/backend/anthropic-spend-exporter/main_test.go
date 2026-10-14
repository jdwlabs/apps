package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// fakeAnthropic serves both endpoints from fields a test can change between
// refreshes.
type fakeAnthropic struct {
	costStatus      int
	costBody        string
	workspaceStatus int
	workspaceBody   string
}

func (f *fakeAnthropic) handler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/organizations/cost_report":
			w.WriteHeader(f.costStatus)
			fmt.Fprint(w, f.costBody)
		case "/v1/organizations/workspaces":
			w.WriteHeader(f.workspaceStatus)
			fmt.Fprint(w, f.workspaceBody)
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	}
}

func costBody(workspaceJSON, amount string) string {
	return fmt.Sprintf(`{"data":[{"results":[{"amount":%q,"currency":"USD","cost_type":"tokens","model":"claude-sonnet-5-5","service_tier":"standard","workspace_id":%s}]}],"has_more":false,"next_page":null}`, amount, workspaceJSON)
}

func newTestRefresher(t *testing.T, f *fakeAnthropic, now time.Time) *refresher {
	t.Helper()
	srv := httptest.NewServer(f.handler(t))
	t.Cleanup(srv.Close)
	return &refresher{
		client:   NewAnthropicClient(srv.URL, "sk-ant-admin-test", srv.Client()),
		view:     newSpendView(100),
		cycleDay: 15,
		names:    map[string]string{},
		now:      func() time.Time { return now },
		log:      slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

var refreshNow = time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)

func TestRefreshOnce_PublishesNamedCostsAndWindow(t *testing.T) {
	f := &fakeAnthropic{
		costStatus: 200, costBody: costBody(`"wrkspc_a"`, "1234"),
		workspaceStatus: 200, workspaceBody: `{"data":[{"id":"wrkspc_a","name":"agents"}],"has_more":false,"last_id":"wrkspc_a"}`,
	}
	r := newTestRefresher(t, f, refreshNow)
	r.refreshOnce(context.Background())

	out := render(r.view)
	for _, want := range []string{
		`jdwlabs_anthropic_cycle_cost_usd{workspace="agents",model="claude-sonnet-5-5",service_tier="standard",cost_type="tokens"} 12.34`,
		// Cycle day 15 on 20 October: 15 October to 15 November.
		`jdwlabs_anthropic_cycle_start_timestamp_seconds 1792022400`,
		`jdwlabs_anthropic_cycle_end_timestamp_seconds 1794700800`,
		`jdwlabs_anthropic_spend_exporter_refreshes_total{outcome="ok"} 1`,
		fmt.Sprintf(`jdwlabs_anthropic_spend_exporter_last_refresh_timestamp_seconds %d`, refreshNow.Unix()),
	} {
		if !strings.Contains(out, want+"\n") {
			t.Errorf("output missing %q\nfull output:\n%s", want, out)
		}
	}
}

func TestRefreshOnce_FailedCostReportKeepsPreviousView(t *testing.T) {
	f := &fakeAnthropic{
		costStatus: 200, costBody: costBody(`"wrkspc_a"`, "1234"),
		workspaceStatus: 200, workspaceBody: `{"data":[{"id":"wrkspc_a","name":"agents"}],"has_more":false,"last_id":"wrkspc_a"}`,
	}
	r := newTestRefresher(t, f, refreshNow)
	r.refreshOnce(context.Background())

	f.costStatus, f.costBody = http.StatusTooManyRequests, ""
	later := refreshNow.Add(5 * time.Minute)
	r.now = func() time.Time { return later }
	r.refreshOnce(context.Background())

	out := render(r.view)
	if !strings.Contains(out, `workspace="agents"`) || !strings.Contains(out, "} 12.34\n") {
		t.Errorf("previous costs were lost after a failed refresh:\n%s", out)
	}
	if !strings.Contains(out, `refreshes_total{outcome="failed"} 1`) {
		t.Errorf("failed refresh was not counted:\n%s", out)
	}
	// The timestamp must not advance, or the staleness alert could never
	// fire while every refresh is failing.
	if !strings.Contains(out, fmt.Sprintf("last_refresh_timestamp_seconds %d\n", refreshNow.Unix())) {
		t.Errorf("last-refresh timestamp moved on a failed refresh:\n%s", out)
	}
}

func TestRefreshOnce_WorkspaceListingFailureStillPublishesCosts(t *testing.T) {
	f := &fakeAnthropic{
		costStatus: 200, costBody: costBody(`"wrkspc_a"`, "1234"),
		workspaceStatus: 200, workspaceBody: `{"data":[{"id":"wrkspc_a","name":"agents"}],"has_more":false,"last_id":"wrkspc_a"}`,
	}
	r := newTestRefresher(t, f, refreshNow)
	r.refreshOnce(context.Background())

	f.workspaceStatus, f.workspaceBody = http.StatusInternalServerError, ""
	f.costBody = costBody(`"wrkspc_a"`, "2000")
	r.refreshOnce(context.Background())

	out := render(r.view)
	// The name from the earlier listing is kept, and the new figure lands.
	if !strings.Contains(out, `jdwlabs_anthropic_cycle_cost_usd{workspace="agents",model="claude-sonnet-5-5",service_tier="standard",cost_type="tokens"} 20`+"\n") {
		t.Errorf("costs not published with the remembered name:\n%s", out)
	}
	if !strings.Contains(out, `refreshes_total{outcome="ok"} 2`) {
		t.Errorf("a refresh with only the name listing failing must still count as ok:\n%s", out)
	}
}

func TestRefreshOnce_WorkspaceLabelFallbacks(t *testing.T) {
	f := &fakeAnthropic{
		costStatus: 200,
		costBody: `{"data":[{"results":[
			{"amount":"100","currency":"USD","cost_type":"tokens","model":"m","service_tier":"standard","workspace_id":null},
			{"amount":"200","currency":"USD","cost_type":"tokens","model":"m","service_tier":"standard","workspace_id":"wrkspc_unlisted"},
			{"amount":"300","currency":"USD","cost_type":"tokens","model":"m","service_tier":"standard","workspace_id":"wrkspc_x"},
			{"amount":"400","currency":"USD","cost_type":"tokens","model":"m","service_tier":"standard","workspace_id":"wrkspc_y"}
		]}],"has_more":false,"next_page":null}`,
		workspaceStatus: 200,
		workspaceBody:   `{"data":[{"id":"wrkspc_x","name":"shared"},{"id":"wrkspc_y","name":"shared"}],"has_more":false,"last_id":"wrkspc_y"}`,
	}
	r := newTestRefresher(t, f, refreshNow)
	r.refreshOnce(context.Background())

	out := render(r.view)
	for _, want := range []string{
		`{workspace="default",model="m",service_tier="standard",cost_type="tokens"} 1`,
		`{workspace="wrkspc_unlisted",model="m",service_tier="standard",cost_type="tokens"} 2`,
		// Two workspaces sharing a name are summed, not overwritten.
		`{workspace="shared",model="m",service_tier="standard",cost_type="tokens"} 7`,
	} {
		if !strings.Contains(out, want+"\n") {
			t.Errorf("output missing %q\nfull output:\n%s", want, out)
		}
	}
}

func TestParseCycleDay(t *testing.T) {
	for _, ok := range []string{"1", "15", "31"} {
		if _, err := parseCycleDay(ok); err != nil {
			t.Errorf("parseCycleDay(%q) rejected: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "0", "32", "-1", "1.5", "first", " 5"} {
		if _, err := parseCycleDay(bad); err == nil {
			t.Errorf("parseCycleDay(%q) accepted", bad)
		}
	}
}

func TestParseBudget(t *testing.T) {
	if usd, err := parseBudget("100"); err != nil || usd != 100 {
		t.Errorf("parseBudget(100) = %v, %v", usd, err)
	}
	if usd, err := parseBudget("199.99"); err != nil || usd != 199.99 {
		t.Errorf("parseBudget(199.99) = %v, %v", usd, err)
	}
	// Zero would make every budget ratio a division by zero in the alerts.
	for _, bad := range []string{"", "0", "-5", "NaN", "Inf", "$100", "abc"} {
		if _, err := parseBudget(bad); err == nil {
			t.Errorf("parseBudget(%q) accepted", bad)
		}
	}
}

func TestRouter(t *testing.T) {
	srv := httptest.NewServer(newRouter(newSpendView(100)))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("/healthz status = %d", resp.StatusCode)
	}

	resp, err = http.Get(srv.URL + "/metrics")
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if got := resp.Header.Get("Content-Type"); got != "text/plain; version=0.0.4" {
		t.Errorf("/metrics Content-Type = %q", got)
	}
	if !strings.Contains(string(body), "jdwlabs_anthropic_credit_budget_usd 100\n") {
		t.Errorf("/metrics body missing the budget gauge:\n%s", body)
	}
}
