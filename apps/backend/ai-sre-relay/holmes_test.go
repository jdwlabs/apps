package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHolmesInvestigate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/investigate" || r.Method != http.MethodPost {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"analysis": "pod OOMKilled; memory limit too low"}`))
	}))
	defer srv.Close()

	a := Alert{Fingerprint: "fp1", Labels: map[string]string{"alertname": "KubePodCrashLooping", "namespace": "prod"}, Annotations: map[string]string{"description": "restarting"}}
	got, err := NewHolmesClient(srv.URL, srv.Client()).Investigate(context.Background(), a)
	if err != nil {
		t.Fatal(err)
	}
	if got.RootCause != "pod OOMKilled; memory limit too low" {
		t.Fatalf("RootCause = %q", got.RootCause)
	}
}

func TestHolmesInvestigateServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	if _, err := NewHolmesClient(srv.URL, srv.Client()).Investigate(context.Background(), Alert{}); err == nil {
		t.Fatal("want error on 500, got nil")
	}
}
