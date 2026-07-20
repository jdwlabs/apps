package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHolmesInvestigate(t *testing.T) {
	var gotReq chatRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" || r.Method != http.MethodPost {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotReq); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"analysis": "pod OOMKilled; memory limit too low"}`))
	}))
	defer srv.Close()

	a := Alert{Fingerprint: "fp1", Labels: map[string]string{"alertname": "KubePodCrashLooping", "namespace": "prod", "pod": "api-0", "severity": "critical"}, Annotations: map[string]string{"description": "restarting"}}
	got, err := NewHolmesClient(srv.URL, "claude-sonnet", srv.Client()).Investigate(context.Background(), a)
	if err != nil {
		t.Fatal(err)
	}
	if got.RootCause != "pod OOMKilled; memory limit too low" {
		t.Fatalf("RootCause = %q", got.RootCause)
	}
	if gotReq.Model != "claude-sonnet" {
		t.Errorf("Model = %q, want claude-sonnet", gotReq.Model)
	}
	if gotReq.Stream {
		t.Error("Stream = true, want false: relay consumes a single JSON response")
	}
	for _, want := range []string{"KubePodCrashLooping", "prod", "api-0", "critical", "restarting"} {
		if !strings.Contains(gotReq.Ask, want) {
			t.Errorf("Ask missing %q: %q", want, gotReq.Ask)
		}
	}
}

// A first response of raw tool-call markup (a real failure mode of weaker
// models) must be re-asked, and the clean second answer used — never filed.
func TestHolmesInvestigateRetriesOnToolCallResponse(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		if calls == 1 {
			_ = json.NewEncoder(w).Encode(map[string]string{
				"analysis": "<tool_call>\n<function=execute_prometheus_instant_query>\n<parameter=query>\nup\n</parameter>\n</function>\n</tool_call>",
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"analysis": "## Root Cause\n\nscrape target down"})
	}))
	defer srv.Close()

	got, err := NewHolmesClient(srv.URL, "claude-sonnet", srv.Client()).Investigate(context.Background(), Alert{Fingerprint: "fp1"})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("expected one retry (2 calls), got %d", calls)
	}
	if strings.Contains(got.RootCause, "<tool_call>") {
		t.Fatalf("tool-call markup leaked into the analysis: %q", got.RootCause)
	}
	if !strings.Contains(got.RootCause, "scrape target down") {
		t.Fatalf("retry answer not used: %q", got.RootCause)
	}
}

// Persistent garbage must become an error — the pipeline then reports a failed
// investigation instead of filing a ticket with no analysis content.
func TestHolmesInvestigateErrorsWhenAnalysisStaysUnusable(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"analysis": `{"name":"kubectl_get","arguments":{"kind":"pod"}}`})
	}))
	defer srv.Close()

	_, err := NewHolmesClient(srv.URL, "claude-sonnet", srv.Client()).Investigate(context.Background(), Alert{Fingerprint: "fp1"})
	if err == nil {
		t.Fatal("want error when every attempt returns tool-call output, got nil")
	}
	if calls != investigateAttempts {
		t.Fatalf("expected %d attempts, got %d", investigateAttempts, calls)
	}
}

// Conversational narration around the analysis must be stripped before the
// result reaches any output.
func TestHolmesInvestigateStripsConversationalFiller(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"analysis": "Now I have a complete picture. Let me summarize the investigation findings.\n\n## Root Cause Analysis\n\netcd degraded under OOM pressure.",
		})
	}))
	defer srv.Close()

	got, err := NewHolmesClient(srv.URL, "claude-sonnet", srv.Client()).Investigate(context.Background(), Alert{Fingerprint: "fp1"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got.RootCause, "## Root Cause Analysis") {
		t.Fatalf("filler preamble not stripped: %q", got.RootCause)
	}
}

func TestHolmesInvestigateServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	if _, err := NewHolmesClient(srv.URL, "claude-sonnet", srv.Client()).Investigate(context.Background(), Alert{}); err == nil {
		t.Fatal("want error on 500, got nil")
	}
}
