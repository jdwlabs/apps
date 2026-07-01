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

func TestDiscordNotifyIncludesLinks(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	pr := PRLink("https://github.com/jdwlabs/platform/pull/9")
	err := NewDiscordNotifier(srv.URL, srv.Client()).Notify(context.Background(),
		Alert{Labels: map[string]string{"alertname": "KubePodCrashLooping"}},
		Analysis{RootCause: "OOM"}, IssueKey("JDWLABS-123"), &pr)
	if err != nil {
		t.Fatal(err)
	}
	content, _ := body["content"].(string)
	if !strings.Contains(content, "KubePodCrashLooping") || !strings.Contains(content, "JDWLABS-123") || !strings.Contains(content, "pull/9") {
		t.Fatalf("content missing fields: %q", content)
	}
}
