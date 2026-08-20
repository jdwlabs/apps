package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestRefreshAll_OneRepoFailureLeavesOthersIntact exercises the invariant
// refreshAll's own doc comment states: a failing repo must not blank out
// series a previous successful cycle already populated for a different repo.
func TestRefreshAll_OneRepoFailureLeavesOthersIntact(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generating test key: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/access_tokens"):
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]string{"token": "tok", "expires_at": "2026-08-20T13:00:00Z"})
		case strings.Contains(r.URL.Path, "/repos/jdwlabs/platform/actions/runs"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"workflow_runs": []map[string]any{
					{"name": "CI", "status": "completed", "conclusion": "success"},
				},
			})
		case strings.Contains(r.URL.Path, "/repos/jdwlabs/apps/actions/runs"):
			w.WriteHeader(http.StatusInternalServerError)
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	gh := NewGitHubClient(srv.URL, "1", "999", key, srv.Client())
	gauges := newCIHealthGauges()
	nullLog := slog.New(slog.NewTextHandler(io.Discard, nil))

	// First cycle: platform succeeds and is recorded.
	refreshAll(context.Background(), gh, "jdwlabs", []string{"platform"}, gauges, nullLog)

	var before strings.Builder
	gauges.writeTo(&before)
	if !strings.Contains(before.String(), `repo="platform",workflow="CI"`) {
		t.Fatalf("platform series missing after its own successful refresh:\n%s", before.String())
	}

	// Second cycle: apps fails, platform is not part of this cycle's repo
	// list at all (simulating a config change) — platform's series from the
	// prior cycle must still be present afterward.
	refreshAll(context.Background(), gh, "jdwlabs", []string{"apps"}, gauges, nullLog)

	var after strings.Builder
	gauges.writeTo(&after)
	if !strings.Contains(after.String(), `repo="platform",workflow="CI"`) {
		t.Error("platform series was lost after an unrelated repo's failed refresh")
	}
	if !strings.Contains(after.String(), `jdwlabs_repo_health_exporter_refreshes_total{outcome="failed"} 1`) {
		t.Errorf("expected exactly one failed-refresh count, got:\n%s", after.String())
	}
}
