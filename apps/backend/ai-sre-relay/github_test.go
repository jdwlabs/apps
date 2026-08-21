package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
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

	p := Patch{Repo: "jdwlabs/platform", FilePath: "values.yaml", NewContent: "limits:\n  memory: 512Mi\n", Rationale: "raise", Confidence: 0.9}
	link, err := NewGitHubClient(srv.URL, StaticGitHubToken("ghtok"), []string{"jdwlabs/platform"}, srv.Client()).OpenPR(context.Background(), p, "JDWLABS-500")
	if err != nil {
		t.Fatal(err)
	}
	if link != "https://github.com/jdwlabs/platform/pull/9" {
		t.Fatalf("link=%q", link)
	}
}

// A repo the model invented must be refused before any request is sent — the
// allowlist is the only thing bounding where relay-authored content lands.
func TestGitHubOpenPRRejectsRepo(t *testing.T) {
	var called bool
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	defer srv.Close()

	cases := []struct {
		name    string
		allowed []string
		repo    string
	}{
		{"not allowlisted", []string{"jdwlabs/platform"}, "attacker/evil"},
		{"empty allowlist disables the arm", nil, "jdwlabs/platform"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			called = false
			p := Patch{Repo: tc.repo, FilePath: "values.yaml", NewContent: "x"}
			_, err := NewGitHubClient(srv.URL, StaticGitHubToken("ghtok"), tc.allowed, srv.Client()).OpenPR(context.Background(), p, "JDWLABS-500")
			if err == nil {
				t.Fatal("expected rejection")
			}
			if called {
				t.Fatal("rejected patch still reached the GitHub API")
			}
		})
	}
}

// A refusal that names only one side of the mismatch leaves an operator unable
// to tell a hallucinated target from a misconfigured allowlist.
func TestGitHubRepoRefusalNamesBothSides(t *testing.T) {
	g := NewGitHubClient("http://unused", StaticGitHubToken("ghtok"), []string{"jdwlabs/platform", "jdwlabs/deployments"}, nil)
	err := g.checkRepo("example/gitops")
	if !errors.Is(err, ErrRepoNotAllowed) {
		t.Fatalf("refusal is not identifiable as an allowlist rejection: %v", err)
	}
	for _, want := range []string{"example/gitops", "jdwlabs/platform", "jdwlabs/deployments"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("refusal omits %q: %v", want, err)
		}
	}
}

// An empty allowlist is a configuration state, not a bad proposal, so it must
// not masquerade as one.
func TestGitHubEmptyAllowlistIsNotARepoRejection(t *testing.T) {
	err := NewGitHubClient("http://unused", StaticGitHubToken("ghtok"), nil, nil).checkRepo("jdwlabs/platform")
	if err == nil {
		t.Fatal("expected the arm to be disabled")
	}
	if errors.Is(err, ErrRepoNotAllowed) {
		t.Fatalf("unconfigured allowlist reported as a rejected proposal: %v", err)
	}
}

func TestGitHubOpenPRRejectsFilePath(t *testing.T) {
	var called bool
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	defer srv.Close()

	for _, bad := range []string{"", "/etc/passwd", "../../../etc/passwd", ".."} {
		t.Run(bad, func(t *testing.T) {
			called = false
			p := Patch{Repo: "jdwlabs/platform", FilePath: bad, NewContent: "x"}
			_, err := NewGitHubClient(srv.URL, StaticGitHubToken("ghtok"), []string{"jdwlabs/platform"}, srv.Client()).OpenPR(context.Background(), p, "JDWLABS-500")
			if err == nil {
				t.Fatalf("expected rejection for %q", bad)
			}
			if called {
				t.Fatalf("rejected path %q still reached the GitHub API", bad)
			}
		})
	}
}

// A non-2xx on the base-ref lookup must name the status; decoding the error
// body yields an empty SHA, which used to mask the real cause.
func TestGitHubOpenPRSurfacesBaseRefStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Not Found"}`))
	}))
	defer srv.Close()

	p := Patch{Repo: "jdwlabs/platform", FilePath: "values.yaml", NewContent: "x"}
	_, err := NewGitHubClient(srv.URL, StaticGitHubToken("ghtok"), []string{"jdwlabs/platform"}, srv.Client()).OpenPR(context.Background(), p, "JDWLABS-500")
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Fatalf("error does not name the status: %v", err)
	}
}

func TestSafeFilePathEscapesSegments(t *testing.T) {
	got, err := safeFilePath("tenants/platform/my values.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if got != "tenants/platform/my%20values.yaml" {
		t.Fatalf("got %q", got)
	}
}
