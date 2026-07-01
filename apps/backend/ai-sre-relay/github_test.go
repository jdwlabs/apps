package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGitHubOpenPR(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/git/ref/heads/main"):
			_ = json.NewEncoder(w).Encode(map[string]any{"object": map[string]string{"sha": "basesha"}})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/git/refs"):
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/contents/"):
			w.WriteHeader(http.StatusNotFound) // new file
		case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/contents/"):
			raw, _ := io.ReadAll(r.Body)
			var m map[string]any
			_ = json.Unmarshal(raw, &m)
			if _, err := base64.StdEncoding.DecodeString(m["content"].(string)); err != nil {
				t.Errorf("content not base64: %v", err)
			}
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/pulls"):
			_ = json.NewEncoder(w).Encode(map[string]string{"html_url": "https://github.com/jdwlabs/platform/pull/9"})
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	p := Patch{Repo: "jdwlabs/platform", FilePath: "values.yaml", NewContent: "limits:\n  memory: 512Mi\n", Branch: "fix/oom", Rationale: "raise", Confidence: 0.9}
	link, err := NewGitHubClient(srv.URL, "ghtok", srv.Client()).OpenPR(context.Background(), p, "JDWLABS-500")
	if err != nil {
		t.Fatal(err)
	}
	if link != "https://github.com/jdwlabs/platform/pull/9" {
		t.Fatalf("link=%q", link)
	}
}
