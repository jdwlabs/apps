package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"
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
	if e.Title != "🔥 FIRING · KubePodCrashLooping" {
		t.Fatalf("title = %q", e.Title)
	}
	if e.URL != "https://jdwlabs.atlassian.net/browse/JDWLABS-123" {
		t.Fatalf("url = %q, want the Jira issue", e.URL)
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

func embedChars(e discordEmbed) int {
	n := utf8.RuneCountInString(e.Title) + utf8.RuneCountInString(e.Description)
	for _, f := range e.Fields {
		n += utf8.RuneCountInString(f.Name) + utf8.RuneCountInString(f.Value)
	}
	return n
}

func TestEmbedTitleCarriesState(t *testing.T) {
	cases := []struct {
		status, want string
	}{
		{"firing", "🔥 FIRING · KubePodNotReady"},
		{"", "🔥 FIRING · KubePodNotReady"},
		{"resolved", "✅ RESOLVED · KubePodNotReady"},
	}
	d := NewDiscordNotifier("http://x", "https://j.example", nil)
	for _, tc := range cases {
		a := Alert{Status: tc.status, Labels: map[string]string{"alertname": "KubePodNotReady"}}
		if got := d.buildEmbed(a, Analysis{}, "", nil, nil).Title; got != tc.want {
			t.Errorf("status %q: title = %q, want %q", tc.status, got, tc.want)
		}
	}
}

func TestEmbedURLOnlyWhenIssueKnown(t *testing.T) {
	d := NewDiscordNotifier("http://x", "https://j.example", nil)
	a := Alert{Labels: map[string]string{"alertname": "X"}}
	if got := d.buildEmbed(a, Analysis{}, "", nil, nil).URL; got != "" {
		t.Errorf("url = %q without an issue", got)
	}
	if got := d.buildEmbed(a, Analysis{}, "OPS-1", nil, nil).URL; got != "https://j.example/browse/OPS-1" {
		t.Errorf("url = %q", got)
	}
}

func TestTablesToCodeBlocks(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{
			"aligned table with inline markup stripped",
			"Findings:\n\n| Pod | **State** |\n|---|:-:|\n| `api-1` | Pending |\n| worker | ✅ OK |\n\nDone.",
			"Findings:\n\n" + codeFence + "\nPod     State\n---------------\napi-1   Pending\nworker  ✅ OK\n" + codeFence + "\n\nDone.",
		},
		{
			"table inside an existing fence is untouched",
			codeFence + "\n| a | b |\n|---|---|\n| 1 | 2 |\n" + codeFence,
			codeFence + "\n| a | b |\n|---|---|\n| 1 | 2 |\n" + codeFence,
		},
		{
			"pipes without a separator row are prose",
			"| not | a table |\nnext",
			"| not | a table |\nnext",
		},
		{"no table", "plain text", "plain text"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tablesToCodeBlocks(tc.in); got != tc.want {
				t.Fatalf("got:\n%s\nwant:\n%s", got, tc.want)
			}
		})
	}
}

func TestTruncateIsRuneSafe(t *testing.T) {
	cases := []struct {
		name, in string
		n        int
		want     string
	}{
		{"short unchanged", "héllo", 10, "héllo"},
		{"exact fit unchanged", "日本語", 3, "日本語"},
		{"cuts on runes not bytes", "日本語テキスト", 4, "日本語…"},
		{"emoji not split", "🔥🔥🔥🔥", 3, "🔥🔥…"},
	}
	for _, tc := range cases {
		got := truncate(tc.in, tc.n)
		if got != tc.want || !utf8.ValidString(got) {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestTruncateMarkdown(t *testing.T) {
	para := func(c string, n int) string { return strings.Repeat(c, n) }
	cases := []struct {
		name  string
		in    string
		n     int
		issue IssueKey
		check func(t *testing.T, got string)
	}{
		{"fits untouched", "short", 100, "OPS-9", func(t *testing.T, got string) {
			if got != "short" {
				t.Fatalf("got %q", got)
			}
		}},
		{"cuts at a paragraph boundary with a Jira pointer", para("a", 40) + "\n\n" + para("b", 40) + "\n\n" + para("c", 40), 120, "OPS-9", func(t *testing.T, got string) {
			want := para("a", 40) + "\n\n" + para("b", 40) + "\n\n…continued in Jira OPS-9"
			if got != want {
				t.Fatalf("got %q, want %q", got, want)
			}
		}},
		{"falls back to a line boundary", para("a", 40) + "\n" + para("b", 40) + "\n" + para("c", 40), 120, "OPS-9", func(t *testing.T, got string) {
			if !strings.HasPrefix(got, para("a", 40)+"\n"+para("b", 40)+"\n\n…") || strings.Contains(got, "ccccc") {
				t.Fatalf("got %q", got)
			}
		}},
		{"no issue gives a generic pointer", para("a", 40) + "\n\n" + para("b", 80), 80, "", func(t *testing.T, got string) {
			if !strings.HasSuffix(got, "…truncated") || strings.Contains(got, "Jira") {
				t.Fatalf("got %q", got)
			}
		}},
		{"multibyte text is cut on a rune", strings.Repeat("日本語 ", 100), 50, "OPS-9", func(t *testing.T, got string) {
			if !utf8.ValidString(got) || utf8.RuneCountInString(got) > 50 {
				t.Fatalf("invalid or oversized: %d runes %q", utf8.RuneCountInString(got), got)
			}
		}},
		{"an open code fence is closed", codeFence + "\n" + strings.Repeat("row\n", 60), 100, "OPS-9", func(t *testing.T, got string) {
			if fenceOpen(strings.TrimSuffix(got, "\n\n…continued in Jira OPS-9")) {
				t.Fatalf("fence left open: %q", got)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := truncateMarkdown(tc.in, tc.n, tc.issue)
			if utf8.RuneCountInString(got) > tc.n {
				t.Fatalf("%d runes exceeds budget %d", utf8.RuneCountInString(got), tc.n)
			}
			tc.check(t, got)
		})
	}
}

func TestEmbedStaysWithinDiscordLimits(t *testing.T) {
	d := NewDiscordNotifier("http://x", "https://j.example", nil)
	table := "| Pod | Node | State |\n|---|---|---|\n| api | n1 | Pending |\n"
	long := strings.Repeat("日本語の長い根本原因の説明。\n\n"+table+"\n", 400)
	pr := PRLink("https://github.com/jdwlabs/platform/pull/" + strings.Repeat("9", 1500))
	patch := &Patch{Confidence: 0.5, Rationale: strings.Repeat("理由", 2000)}
	a := Alert{Labels: map[string]string{"alertname": "A", "namespace": strings.Repeat("n", 2000), "severity": "critical"}}
	e := d.buildEmbed(a, Analysis{RootCause: long}, "OPS-7", &pr, patch)

	if n := utf8.RuneCountInString(e.Description); n > maxEmbedDescription {
		t.Errorf("description = %d chars, over %d", n, maxEmbedDescription)
	}
	if n := embedChars(e); n > maxEmbedTotal {
		t.Errorf("embed = %d chars, over %d", n, maxEmbedTotal)
	}
	for _, f := range e.Fields {
		if n := utf8.RuneCountInString(f.Value); n > maxEmbedFieldValue {
			t.Errorf("field %q = %d chars, over %d", f.Name, n, maxEmbedFieldValue)
		}
	}
	if !strings.HasSuffix(e.Description, "…continued in Jira OPS-7") {
		t.Errorf("missing Jira pointer: %q", e.Description[len(e.Description)-60:])
	}
	if raw, _ := json.Marshal(e); !utf8.Valid(raw) {
		t.Error("embed JSON is not valid UTF-8")
	}
}

func TestEmbedRendersHolmesDriftAnalysis(t *testing.T) {
	d := NewDiscordNotifier("http://x", "https://jdwlabs.atlassian.net", nil)
	rc := "## Root cause\n\nThe Application `platform-holmes` is OutOfSync.\n\n| Resource | Status |\n|---|---|\n| Deployment/holmes | OutOfSync |\n| ConfigMap/holmes-config | Synced |\n\nRe-sync after the config change."
	a := Alert{Status: "firing", Labels: map[string]string{"alertname": "ArgoCDAppOutOfSync", "namespace": "argocd", "severity": "warning"}}
	e := d.buildEmbed(a, Analysis{RootCause: rc}, "OPS-322", nil, nil)
	if strings.Contains(e.Description, "|---") || !strings.Contains(e.Description, codeFence+"\nResource") {
		t.Fatalf("table not converted:\n%s", e.Description)
	}
	if e.Color != colorOrange {
		t.Fatalf("color = %#x", e.Color)
	}
	if testing.Verbose() {
		raw, _ := json.MarshalIndent(e, "", "  ")
		t.Logf("%s", raw)
	}
}
