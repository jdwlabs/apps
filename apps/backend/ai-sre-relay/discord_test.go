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

func newRecordingServer(t *testing.T, body *struct {
	Embeds []discordEmbed `json:"embeds"`
}) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, body)
		w.WriteHeader(http.StatusNoContent)
	}))
}

func TestDiscordNotifyBuildsEmbedWithLinks(t *testing.T) {
	var body struct {
		Embeds []discordEmbed `json:"embeds"`
	}
	srv := newRecordingServer(t, &body)
	defer srv.Close()

	pr := PRLink("https://github.com/jdwlabs/platform/pull/9")
	patch := &Patch{Confidence: 0.82, Rationale: "bump memory limit"}
	alert := Alert{
		Labels: map[string]string{"alertname": "KubePodCrashLooping", "namespace": "database", "severity": "critical"},
	}
	err := NewDiscordNotifier(srv.URL, "https://jdwlabs.atlassian.net", srv.Client()).Notify(
		context.Background(), alert, Analysis{RootCause: "OOM"}, IssueKey("JDWLABS-123"), &pr, patch)
	if err != nil {
		t.Fatal(err)
	}
	if len(body.Embeds) != 1 {
		t.Fatalf("expected 1 embed, got %d", len(body.Embeds))
	}
	e := body.Embeds[0]
	if e.Title != "KubePodCrashLooping" {
		t.Fatalf("title = %q", e.Title)
	}
	if e.Color != colorRed {
		t.Fatalf("color = %#x, want red for critical severity", e.Color)
	}
	var joined string
	for _, f := range e.Fields {
		joined += f.Name + ":" + f.Value + "\n"
	}
	if !strings.Contains(joined, "database") {
		t.Fatalf("fields missing namespace: %q", joined)
	}
	if !strings.Contains(joined, "jdwlabs.atlassian.net/browse/JDWLABS-123") {
		t.Fatalf("fields missing jira link: %q", joined)
	}
	if !strings.Contains(joined, "pull/9") {
		t.Fatalf("fields missing PR link: %q", joined)
	}
	if !strings.Contains(joined, "82% confidence") {
		t.Fatalf("fields missing patch confidence: %q", joined)
	}
}

func TestDiscordNotifyResolvedAlertIsGreenRegardlessOfSeverity(t *testing.T) {
	var body struct {
		Embeds []discordEmbed `json:"embeds"`
	}
	srv := newRecordingServer(t, &body)
	defer srv.Close()

	alert := Alert{Status: "resolved", Labels: map[string]string{"alertname": "KubePodCrashLooping", "severity": "critical"}}
	err := NewDiscordNotifier(srv.URL, "https://jdwlabs.atlassian.net", srv.Client()).Notify(
		context.Background(), alert, Analysis{RootCause: "fixed"}, "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if body.Embeds[0].Color != colorGreen {
		t.Fatalf("color = %#x, want green for resolved", body.Embeds[0].Color)
	}
}

func TestDiscordNotifyOmitsFieldsWhenAbsent(t *testing.T) {
	var body struct {
		Embeds []discordEmbed `json:"embeds"`
	}
	srv := newRecordingServer(t, &body)
	defer srv.Close()

	err := NewDiscordNotifier(srv.URL, "https://jdwlabs.atlassian.net", srv.Client()).Notify(
		context.Background(), Alert{Labels: map[string]string{"alertname": "X"}},
		Analysis{RootCause: "⚠️ investigation failed: boom"}, "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(body.Embeds[0].Fields) != 0 {
		t.Fatalf("expected no fields when namespace/severity/issue/pr/patch absent, got %+v", body.Embeds[0].Fields)
	}
}

func TestDiscordNotifyTruncatesPatchRationaleThenStayUnder1024Chars(t *testing.T) {
	var body struct {
		Embeds []discordEmbed `json:"embeds"`
	}
	srv := newRecordingServer(t, &body)
	defer srv.Close()

	// Create a patch with an extremely long rationale (2000 chars)
	longRationale := strings.Repeat("x", 2000)
	patch := &Patch{Confidence: 0.95, Rationale: longRationale}
	alert := Alert{
		Labels: map[string]string{"alertname": "TestAlert"},
	}
	err := NewDiscordNotifier(srv.URL, "https://jdwlabs.atlassian.net", srv.Client()).Notify(
		context.Background(), alert, Analysis{RootCause: "test"}, "", nil, patch)
	if err != nil {
		t.Fatal(err)
	}

	// Find the Patch field and verify its value is under 1024 chars
	if len(body.Embeds) != 1 {
		t.Fatalf("expected 1 embed, got %d", len(body.Embeds))
	}
	e := body.Embeds[0]
	var patchFieldValue string
	for _, f := range e.Fields {
		if f.Name == "Patch" {
			patchFieldValue = f.Value
			break
		}
	}
	if patchFieldValue == "" {
		t.Fatal("Patch field not found")
	}
	if len(patchFieldValue) >= 1024 {
		t.Fatalf("patch field value exceeds discord's 1024-char limit: got %d chars", len(patchFieldValue))
	}
	if !strings.Contains(patchFieldValue, "95% confidence") {
		t.Fatalf("patch field missing confidence: %q", patchFieldValue)
	}
	if !strings.Contains(patchFieldValue, "…") {
		t.Fatalf("expected truncate ellipsis in field value: %q", patchFieldValue)
	}
}
