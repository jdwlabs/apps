package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The globs pass every path below; only the tenant.yaml on main can say that
// nothing reads it. Each must be refused with nothing written.
func TestGitHubOpenPRRefusesPathsTenantYAMLDoesNotReconcile(t *testing.T) {
	for _, tc := range []struct{ path, why string }{
		{"tenants/platform/services/parked/values.yaml", "not listed"},
		{"tenants/platform/services/never-added/postInstall/job.yaml", "not listed"},
		{"tenants/platform/services/raw-only/values.yaml", "rawManifests"},
		{"tenants/platform/services/values-only/postInstall/extra.yaml", "does not read postInstall"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			srv, calls := scriptedGitHub(t, http.StatusOK, http.StatusCreated)
			defer srv.Close()
			p := verified(Patch{Repo: "jdwlabs/platform", FilePath: tc.path, NewContent: "x: 1\n", Rationale: "r", Confidence: 0.9})
			_, err := constrainedClient(srv).OpenPR(context.Background(), p, "JDWLABS-500")
			if !errors.Is(err, ErrPathNotAllowed) {
				t.Fatalf("want ErrPathNotAllowed, got %v", err)
			}
			if !strings.Contains(err.Error(), tc.why) {
				t.Fatalf("refusal does not say why (%q): %v", tc.why, err)
			}
			assertNoWrites(t, *calls)
		})
	}
}

func TestGitHubOpenPRAcceptsPathsTenantYAMLReconciles(t *testing.T) {
	for _, path := range []string{
		"tenants/platform/services/vault/values.yaml",
		"tenants/platform/services/vault/postInstall/externalsecret.yaml",
		"tenants/platform/services/raw-only/postInstall/job.yaml",
		"tenants/platform/services/values-only/values.yaml",
	} {
		t.Run(path, func(t *testing.T) {
			srv, _ := scriptedGitHub(t, http.StatusOK, http.StatusCreated)
			defer srv.Close()
			p := verified(Patch{Repo: "jdwlabs/platform", FilePath: path, NewContent: "x: 1\n", Rationale: "r", Confidence: 0.9})
			if _, err := constrainedClient(srv).OpenPR(context.Background(), p, "JDWLABS-500"); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// Not knowing what ArgoCD reads is a refusal, never a pass.
func TestGitHubOpenPRRefusesWhenTenantYAMLIsUnreadable(t *testing.T) {
	for name, handler := range map[string]http.HandlerFunc{
		"missing": func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) },
		"garbage": func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"encoding":"base64","content":"c2VydmljZXM6IFt1bmNsb3NlZA=="}`))
		},
	} {
		t.Run(name, func(t *testing.T) {
			var calls []ghCall
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls = append(calls, ghCall{r.Method, r.URL.Path, nil})
				switch {
				case strings.HasSuffix(r.URL.Path, "/git/ref/heads/main"):
					_, _ = w.Write([]byte(`{"object":{"sha":"basesha"}}`))
				case strings.HasSuffix(r.URL.Path, "/tenant.yaml"):
					handler(w, r)
				default:
					t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
				}
			}))
			defer srv.Close()
			p := verified(Patch{Repo: "jdwlabs/platform", FilePath: "tenants/platform/services/vault/values.yaml", NewContent: "x: 1\n", Rationale: "r", Confidence: 0.9})
			_, err := constrainedClient(srv).OpenPR(context.Background(), p, "JDWLABS-500")
			if !errors.Is(err, ErrPathNotAllowed) {
				t.Fatalf("want ErrPathNotAllowed, got %v", err)
			}
			assertNoWrites(t, calls)
		})
	}
}
