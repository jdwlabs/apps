package main

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

var testStart = time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)

func newTestClient(t *testing.T, h http.HandlerFunc) *AnthropicClient {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return NewAnthropicClient(srv.URL, "sk-ant-admin-test", srv.Client())
}

func assertUSD(t *testing.T, got map[costSeries]float64, key costSeries, want float64) {
	t.Helper()
	v, ok := got[key]
	if !ok {
		t.Fatalf("series %+v missing from %v", key, got)
	}
	if math.Abs(v-want) > 1e-9 {
		t.Errorf("series %+v = %v, want %v", key, v, want)
	}
}

func TestCycleCosts_SendsAuthAndQuery(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/organizations/cost_report" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("x-api-key"); got != "sk-ant-admin-test" {
			t.Errorf("x-api-key = %q", got)
		}
		if got := r.Header.Get("anthropic-version"); got != "2023-06-01" {
			t.Errorf("anthropic-version = %q", got)
		}
		if !strings.HasPrefix(r.Header.Get("User-Agent"), "jdwlabs-anthropic-spend-exporter/") {
			t.Errorf("User-Agent = %q", r.Header.Get("User-Agent"))
		}
		q := r.URL.Query()
		if got := q.Get("starting_at"); got != "2026-10-15T00:00:00Z" {
			t.Errorf("starting_at = %q", got)
		}
		if got := q.Get("bucket_width"); got != "1d" {
			t.Errorf("bucket_width = %q", got)
		}
		if got := q.Get("limit"); got != "31" {
			t.Errorf("limit = %q", got)
		}
		if got := q["group_by[]"]; len(got) != 2 || got[0] != "workspace_id" || got[1] != "description" {
			t.Errorf("group_by[] = %v", got)
		}
		fmt.Fprint(w, `{"data":[],"has_more":false,"next_page":null}`)
	})
	costs, err := c.CycleCosts(context.Background(), testStart)
	if err != nil {
		t.Fatalf("CycleCosts: %v", err)
	}
	if len(costs) != 0 {
		t.Errorf("empty report produced series: %v", costs)
	}
}

func TestCycleCosts_SumsAcrossBucketsAndPages(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("page") {
		case "":
			fmt.Fprint(w, `{"data":[{"results":[
				{"amount":"123.45","currency":"USD","cost_type":"tokens","model":"claude-sonnet-5-5","service_tier":"standard","workspace_id":"wrkspc_a"},
				{"amount":"0","currency":"USD","cost_type":"tokens","model":"claude-haiku-5-5","service_tier":"batch","workspace_id":"wrkspc_a"},
				{"amount":"1000","currency":"USD","cost_type":"web_search","model":null,"service_tier":null,"workspace_id":null}
			]},{"results":[]}],"has_more":true,"next_page":"page_2"}`)
		case "page_2":
			fmt.Fprint(w, `{"data":[{"results":[
				{"amount":"76.55","currency":"USD","cost_type":"tokens","model":"claude-sonnet-5-5","service_tier":"standard","workspace_id":"wrkspc_a"}
			]}],"has_more":false,"next_page":null}`)
		default:
			t.Errorf("unexpected page %q", r.URL.Query().Get("page"))
		}
	})
	costs, err := c.CycleCosts(context.Background(), testStart)
	if err != nil {
		t.Fatalf("CycleCosts: %v", err)
	}
	// 123.45 + 76.55 cents across two pages is exactly two dollars.
	assertUSD(t, costs, costSeries{"wrkspc_a", "claude-sonnet-5-5", "standard", "tokens"}, 2)
	// A zero-cost line is still a series: the model was used this cycle.
	assertUSD(t, costs, costSeries{"wrkspc_a", "claude-haiku-5-5", "batch", "tokens"}, 0)
	// Null workspace, model and tier arrive as empty strings.
	assertUSD(t, costs, costSeries{"", "", "", "web_search"}, 10)
	if len(costs) != 3 {
		t.Errorf("got %d series, want 3: %v", len(costs), costs)
	}
}

func TestCycleCosts_FailureOnLaterPageReturnsNoPartialTotal(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") == "" {
			fmt.Fprint(w, `{"data":[{"results":[{"amount":"500","currency":"USD","cost_type":"tokens","model":"m","service_tier":"standard","workspace_id":"w"}]}],"has_more":true,"next_page":"p2"}`)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	})
	costs, err := c.CycleCosts(context.Background(), testStart)
	if err == nil {
		t.Fatal("expected an error when a later page fails")
	}
	if costs != nil {
		t.Errorf("partial costs returned alongside the error: %v", costs)
	}
}

func TestCycleCosts_RejectsBadPayloads(t *testing.T) {
	item := func(amount, currency string) string {
		return fmt.Sprintf(`{"data":[{"results":[{"amount":%q,"currency":%q,"cost_type":"tokens","model":"m","service_tier":"standard","workspace_id":"w"}]}],"has_more":false,"next_page":null}`, amount, currency)
	}
	cases := []struct {
		name    string
		body    string
		wantErr string
	}{
		{"non-numeric amount", item("abc", "USD"), "unparseable amount"},
		{"empty amount", item("", "USD"), "unparseable amount"},
		{"NaN amount", item("NaN", "USD"), "unparseable amount"},
		{"infinite amount", item("Inf", "USD"), "unparseable amount"},
		{"non-USD currency", item("100", "EUR"), "unexpected currency"},
		{"has_more without a cursor", `{"data":[],"has_more":true,"next_page":null}`, "has_more without next_page"},
		{"body is not JSON", `<html>bad gateway</html>`, "decoding response"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, tc.body) })
			_, err := c.CycleCosts(context.Background(), testStart)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("err = %v, want one containing %q", err, tc.wantErr)
			}
		})
	}
}

func TestCycleCosts_RepeatedCursorStopsAtPageCap(t *testing.T) {
	calls := 0
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		fmt.Fprint(w, `{"data":[],"has_more":true,"next_page":"same"}`)
	})
	_, err := c.CycleCosts(context.Background(), testStart)
	if err == nil || !strings.Contains(err.Error(), "more than 50 pages") {
		t.Errorf("err = %v, want the page-cap error", err)
	}
	if calls != maxPages {
		t.Errorf("made %d requests, want exactly %d", calls, maxPages)
	}
}

func TestGet_AuthFailureNamesTheLikelyCause(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(status) })
		_, err := c.CycleCosts(context.Background(), testStart)
		if err == nil || !strings.Contains(err.Error(), "Admin API key") || !strings.Contains(err.Error(), "individual accounts") {
			t.Errorf("status %d: err = %v, want one naming the Admin key and individual accounts", status, err)
		}
		if err != nil && strings.Contains(err.Error(), "sk-ant-admin-test") {
			t.Errorf("status %d: error leaks the key: %v", status, err)
		}
	}
}

func TestGet_OversizedBodyIsRejected(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[],"has_more":false,"pad":"`))
		_, _ = w.Write([]byte(strings.Repeat("x", maxBodyBytes)))
		_, _ = w.Write([]byte(`"}`))
	})
	_, err := c.CycleCosts(context.Background(), testStart)
	if err == nil || !strings.Contains(err.Error(), "larger than") {
		t.Errorf("err = %v, want the size-limit error", err)
	}
}

func TestWorkspaceNames_FollowsCursor(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/organizations/workspaces" {
			t.Errorf("path = %s", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("include_archived") != "true" || q.Get("limit") != "1000" {
			t.Errorf("query = %v", q)
		}
		switch q.Get("after_id") {
		case "":
			fmt.Fprint(w, `{"data":[{"id":"wrkspc_a","name":"agents"}],"has_more":true,"last_id":"wrkspc_a"}`)
		case "wrkspc_a":
			fmt.Fprint(w, `{"data":[{"id":"wrkspc_b","name":"sre"}],"has_more":false,"last_id":"wrkspc_b"}`)
		default:
			t.Errorf("unexpected after_id %q", q.Get("after_id"))
		}
	})
	names, err := c.WorkspaceNames(context.Background())
	if err != nil {
		t.Fatalf("WorkspaceNames: %v", err)
	}
	if len(names) != 2 || names["wrkspc_a"] != "agents" || names["wrkspc_b"] != "sre" {
		t.Errorf("names = %v", names)
	}
}
