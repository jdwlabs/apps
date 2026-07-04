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

func TestHolmesInvestigateServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	if _, err := NewHolmesClient(srv.URL, "claude-sonnet", srv.Client()).Investigate(context.Background(), Alert{}); err == nil {
		t.Fatal("want error on 500, got nil")
	}
}
