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

// fakeLiteLLM returns an OpenAI-shaped chat completion whose message content
// is the given string (the model's raw reply).
func fakeLiteLLM(t *testing.T, content string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("missing/wrong auth: %q", r.Header.Get("Authorization"))
		}
		resp := map[string]any{"choices": []map[string]any{{"message": map[string]string{"content": content}}}}
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

// capturingLiteLLM records the system prompt the generator sent before
// replying, so the instruction the model actually receives can be asserted on.
func capturingLiteLLM(t *testing.T, content string, system *string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var req openAIRequest
		if err := json.Unmarshal(raw, &req); err != nil {
			t.Errorf("request body: %v", err)
		}
		for _, m := range req.Messages {
			if m.Role == "system" {
				*system = m.Content
			}
		}
		resp := map[string]any{"choices": []map[string]any{{"message": map[string]string{"content": content}}}}
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

func TestPatchGenerateValid(t *testing.T) {
	patch := `{"file_path":"values.yaml","new_content":"limits:\n  memory: 512Mi\n","rationale":"raise limit","confidence":0.9}`
	srv := fakeLiteLLM(t, patch)
	defer srv.Close()
	g := NewPatchGenerator(srv.URL, "test-key", "claude-sonnet", 0.7, []string{"jdwlabs/platform"}, silentLogger(), srv.Client())
	got, err := g.Generate(context.Background(), Analysis{RootCause: "OOM"})
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.Repo != "jdwlabs/platform" || got.Confidence != 0.9 {
		t.Fatalf("bad patch: %+v", got)
	}
}

func TestPatchGenerateLowConfidenceReturnsNil(t *testing.T) {
	srv := fakeLiteLLM(t, `{"file_path":"a","new_content":"b","rationale":"d","confidence":0.3}`)
	defer srv.Close()
	g := NewPatchGenerator(srv.URL, "test-key", "m", 0.7, []string{"jdwlabs/platform"}, silentLogger(), srv.Client())
	got, err := g.Generate(context.Background(), Analysis{})
	if err != nil || got != nil {
		t.Fatalf("want (nil,nil), got (%+v,%v)", got, err)
	}
}

func TestPatchGenerateMalformedReturnsNil(t *testing.T) {
	srv := fakeLiteLLM(t, "sorry, I cannot produce a patch")
	defer srv.Close()
	g := NewPatchGenerator(srv.URL, "test-key", "m", 0.7, []string{"jdwlabs/platform"}, silentLogger(), srv.Client())
	got, err := g.Generate(context.Background(), Analysis{})
	if err != nil || got != nil {
		t.Fatalf("want (nil,nil), got (%+v,%v)", got, err)
	}
}

// The single-target case is the whole fix: a repository the model invented is
// discarded in favour of the configured one, so a hallucinated name can no
// longer cost the remediation.
func TestPatchGenerateOverridesHallucinatedRepo(t *testing.T) {
	for _, invented := range []string{"example/gitops", "acme/corp-gitops", "jdwlabs/platform-infra"} {
		t.Run(invented, func(t *testing.T) {
			body := `{"repo":"` + invented + `","file_path":"values.yaml","new_content":"x","rationale":"r","confidence":0.9}`
			srv := fakeLiteLLM(t, body)
			defer srv.Close()
			g := NewPatchGenerator(srv.URL, "test-key", "m", 0.7, []string{"jdwlabs/platform"}, silentLogger(), srv.Client())
			got, err := g.Generate(context.Background(), Analysis{})
			if err != nil {
				t.Fatal(err)
			}
			if got == nil {
				t.Fatal("patch dropped; the configured target should have replaced the invented one")
			}
			if got.Repo != "jdwlabs/platform" {
				t.Fatalf("repo = %q, want the configured target", got.Repo)
			}
		})
	}
}

// With one target the model is told the destination and is not asked for it.
func TestPatchPromptStatesSingleTargetAndOmitsRepoField(t *testing.T) {
	var system string
	srv := capturingLiteLLM(t, `{"confidence":0}`, &system)
	defer srv.Close()
	g := NewPatchGenerator(srv.URL, "test-key", "m", 0.7, []string{"jdwlabs/platform"}, silentLogger(), srv.Client())
	if _, err := g.Generate(context.Background(), Analysis{RootCause: "x"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(system, "jdwlabs/platform") {
		t.Fatalf("prompt never names the target repository:\n%s", system)
	}
	if strings.Contains(system, `"repo":"owner/name"`) || strings.Contains(system, `"repo":"..."`) {
		t.Fatalf("prompt still asks the model for a repository:\n%s", system)
	}
}

// Several targets is a real choice, so it is offered — but as a closed
// enumeration naming every option, never as free text.
func TestPatchPromptEnumeratesMultipleTargets(t *testing.T) {
	var system string
	srv := capturingLiteLLM(t, `{"confidence":0}`, &system)
	defer srv.Close()
	targets := []string{"jdwlabs/platform", "jdwlabs/deployments"}
	g := NewPatchGenerator(srv.URL, "test-key", "m", 0.7, targets, silentLogger(), srv.Client())
	if _, err := g.Generate(context.Background(), Analysis{RootCause: "x"}); err != nil {
		t.Fatal(err)
	}
	for _, want := range targets {
		if !strings.Contains(system, want) {
			t.Fatalf("prompt omits target %q:\n%s", want, system)
		}
	}
}

func TestPatchGenerateMultiTargetAcceptsListedRepo(t *testing.T) {
	body := `{"repo":"jdwlabs/deployments","file_path":"values.yaml","new_content":"x","rationale":"r","confidence":0.9}`
	srv := fakeLiteLLM(t, body)
	defer srv.Close()
	g := NewPatchGenerator(srv.URL, "test-key", "m", 0.7,
		[]string{"jdwlabs/platform", "jdwlabs/deployments"}, silentLogger(), srv.Client())
	got, err := g.Generate(context.Background(), Analysis{})
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.Repo != "jdwlabs/deployments" {
		t.Fatalf("bad patch: %+v", got)
	}
}

// End to end across the two halves of the fix: a model reply naming a
// repository that does not exist now reaches the GitHub arm as the configured
// target, and a PR is opened instead of the proposal being thrown away.
func TestHallucinatedRepoStillOpensPRAgainstConfiguredTarget(t *testing.T) {
	llm := fakeLiteLLM(t, `{"repo":"example/gitops","file_path":"tenants/platform/values.yaml","new_content":"limits:\n  memory: 512Mi\n","rationale":"raise limit","confidence":0.9}`)
	defer llm.Close()

	var pullPath string
	gh := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/git/ref/heads/main"):
			_ = json.NewEncoder(w).Encode(map[string]any{"object": map[string]string{"sha": "basesha"}})
		case strings.HasSuffix(r.URL.Path, "/git/refs"):
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/contents/"):
			w.WriteHeader(http.StatusNotFound)
		case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/contents/"):
			w.WriteHeader(http.StatusCreated)
		case strings.HasSuffix(r.URL.Path, "/pulls"):
			pullPath = r.URL.Path
			_ = json.NewEncoder(w).Encode(map[string]string{"html_url": "https://github.com/jdwlabs/platform/pull/1"})
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer gh.Close()

	targets := []string{"jdwlabs/platform"}
	patch, err := NewPatchGenerator(llm.URL, "test-key", "m", 0.7, targets, silentLogger(), llm.Client()).
		Generate(context.Background(), Analysis{RootCause: "OOM"})
	if err != nil || patch == nil {
		t.Fatalf("no patch produced: (%+v, %v)", patch, err)
	}
	link, err := NewGitHubClient(gh.URL, StaticGitHubToken("ghtok"), targets, gh.Client()).
		OpenPR(context.Background(), *patch, "JDWLABS-312")
	if err != nil {
		t.Fatalf("PR refused after injection: %v", err)
	}
	if link == "" {
		t.Fatal("no PR link returned")
	}
	if pullPath != "/repos/jdwlabs/platform/pulls" {
		t.Fatalf("PR opened against %q, want the configured target", pullPath)
	}
}

// A choice the model was given cannot be answered with something outside it.
func TestPatchGenerateMultiTargetRejectsUnlistedRepo(t *testing.T) {
	body := `{"repo":"example/gitops","file_path":"values.yaml","new_content":"x","rationale":"r","confidence":0.9}`
	srv := fakeLiteLLM(t, body)
	defer srv.Close()
	g := NewPatchGenerator(srv.URL, "test-key", "m", 0.7,
		[]string{"jdwlabs/platform", "jdwlabs/deployments"}, silentLogger(), srv.Client())
	got, err := g.Generate(context.Background(), Analysis{})
	if err != nil || got != nil {
		t.Fatalf("want (nil,nil) for an unlisted repo, got (%+v,%v)", got, err)
	}
}
