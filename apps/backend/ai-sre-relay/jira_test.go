package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestJiraUpsertCreatesWhenNoDuplicate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/rest/api/3/search"):
			_ = json.NewEncoder(w).Encode(map[string]any{"issues": []any{}})
		case r.Method == http.MethodPost && r.URL.Path == "/rest/api/3/issue":
			raw, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(raw), "amfp-fp1") {
				t.Errorf("create missing dedup label: %s", raw)
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]string{"key": "JDWLABS-500"})
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	key, err := NewJiraClient(srv.URL, "e@x", "tok", "JDWLABS", srv.Client()).
		Upsert(context.Background(), Alert{Fingerprint: "fp1", Labels: map[string]string{"alertname": "X"}}, Analysis{RootCause: "y"})
	if err != nil || key != "JDWLABS-500" {
		t.Fatalf("key=%q err=%v", key, err)
	}
}

func TestJiraUpsertCommentsWhenDuplicate(t *testing.T) {
	commented := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/rest/api/3/search"):
			_ = json.NewEncoder(w).Encode(map[string]any{"issues": []map[string]string{{"key": "JDWLABS-77"}}})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/comment"):
			commented = true
			w.WriteHeader(http.StatusCreated)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	key, err := NewJiraClient(srv.URL, "e@x", "tok", "JDWLABS", srv.Client()).
		Upsert(context.Background(), Alert{Fingerprint: "fp1"}, Analysis{RootCause: "y"})
	if err != nil || key != "JDWLABS-77" || !commented {
		t.Fatalf("key=%q err=%v commented=%v", key, err, commented)
	}
}
