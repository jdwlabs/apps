package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func TestPatchGenerateValid(t *testing.T) {
	patch := `{"repo":"jdwlabs/platform","file_path":"values.yaml","new_content":"limits:\n  memory: 512Mi\n","branch":"fix/oom","rationale":"raise limit","confidence":0.9}`
	srv := fakeLiteLLM(t, patch)
	defer srv.Close()
	g := NewPatchGenerator(srv.URL, "test-key", "claude-sonnet", 0.7, srv.Client())
	got, err := g.Generate(context.Background(), Analysis{RootCause: "OOM"})
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.Repo != "jdwlabs/platform" || got.Confidence != 0.9 {
		t.Fatalf("bad patch: %+v", got)
	}
}

func TestPatchGenerateLowConfidenceReturnsNil(t *testing.T) {
	srv := fakeLiteLLM(t, `{"repo":"x/y","file_path":"a","new_content":"b","branch":"c","rationale":"d","confidence":0.3}`)
	defer srv.Close()
	got, err := NewPatchGenerator(srv.URL, "test-key", "m", 0.7, srv.Client()).Generate(context.Background(), Analysis{})
	if err != nil || got != nil {
		t.Fatalf("want (nil,nil), got (%+v,%v)", got, err)
	}
}

func TestPatchGenerateMalformedReturnsNil(t *testing.T) {
	srv := fakeLiteLLM(t, "sorry, I cannot produce a patch")
	defer srv.Close()
	got, err := NewPatchGenerator(srv.URL, "test-key", "m", 0.7, srv.Client()).Generate(context.Background(), Analysis{})
	if err != nil || got != nil {
		t.Fatalf("want (nil,nil), got (%+v,%v)", got, err)
	}
}
