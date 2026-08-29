package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// testPathGlobs mirrors the platform repository's watched layout, which is
// what the deployment configures. tenant.yaml is deliberately not a member:
// it is reconciled, but it is also the whole per-tenant service list
// (governance-appset renders it with prune and self-heal), and no
// remediation use case needs a full-file rewrite of it — see patch.go.
var testPathGlobs = []string{
	"tenants/*/services/*/values.yaml",
	"tenants/*/services/*/postInstall/*.yaml",
}

// scriptedGitHub answers OpenPR with the given statuses for the base-branch
// file lookup and the branch create, records every call, and hands back the
// PUT/PR calls so a test can assert nothing was written. hasExistingPR
// controls what a GET .../pulls?state=all listing returns, which only
// matters when branchStatus is 422: it decides whether the 422 is read as
// "already under review" (a PR exists) or "orphaned" (it does not).
func scriptedGitHub(t *testing.T, fileStatus, branchStatus int, hasExistingPR ...bool) (*httptest.Server, *[]ghCall) {
	t.Helper()
	existingPR := len(hasExistingPR) > 0 && hasExistingPR[0]
	var calls []ghCall
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if raw, _ := io.ReadAll(r.Body); len(raw) > 0 {
			_ = json.Unmarshal(raw, &body)
		}
		calls = append(calls, ghCall{r.Method, r.URL.Path, body})
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/git/ref/heads/main"):
			_ = json.NewEncoder(w).Encode(map[string]any{"object": map[string]string{"sha": "basesha"}})
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/contents/"):
			if r.URL.Query().Get("ref") != baseBranch {
				t.Errorf("file existence checked against %q, want %s", r.URL.Query().Get("ref"), baseBranch)
			}
			w.WriteHeader(fileStatus)
			if fileStatus == http.StatusOK {
				_ = json.NewEncoder(w).Encode(map[string]string{"sha": "filesha", "type": "file"})
			}
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/git/refs"):
			w.WriteHeader(branchStatus)
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/pulls"):
			if r.URL.Query().Get("state") != "all" {
				t.Errorf("pulls listed with state=%q, want all", r.URL.Query().Get("state"))
			}
			if existingPR {
				_ = json.NewEncoder(w).Encode([]map[string]any{{"number": 9}})
			} else {
				_ = json.NewEncoder(w).Encode([]map[string]any{})
			}
		case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/contents/"):
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/pulls"):
			_ = json.NewEncoder(w).Encode(map[string]string{"html_url": "https://github.com/jdwlabs/platform/pull/9"})
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	return srv, &calls
}

func assertNoWrites(t *testing.T, calls []ghCall) {
	t.Helper()
	for _, c := range calls {
		if c.method == http.MethodPut || strings.HasSuffix(c.path, "/pulls") || strings.HasSuffix(c.path, "/git/refs") {
			t.Fatalf("refused proposal still wrote: %s %s", c.method, c.path)
		}
	}
}

// Every path the bot actually proposed before the gate existed, verbatim from
// the closed PRs, must be refused before any API call is made.
func TestGitHubOpenPRRejectsUnwatchedPaths(t *testing.T) {
	srv, calls := scriptedGitHub(t, http.StatusOK, http.StatusCreated)
	defer srv.Close()

	for _, bad := range []string{
		"argocd/values.yaml",
		"cluster/grafana/grafana-gitsync-github-app.yaml",
		"platform/vault/statefulset.yaml",
		"manifests/democratic-csi/platform-democratic-csi-node.yaml",
		"clusters/jdwlabs/terraform/network.tf",
		"apps/jdwillmsen/minecraft/fwb/prd/volume-recovery/job.yaml",
		"manifests/vault-auto-unseal-cronjob.yaml",
		"tenants/platform/values.yaml",
		"tenants/platform/tenant.yaml",
		"tenants/platform/services/vault/postInstall/nested/x.yaml",
		"README.md",
		".github/workflows/validate.yml",
	} {
		t.Run(bad, func(t *testing.T) {
			*calls = nil
			p := Patch{Repo: "jdwlabs/platform", FilePath: bad, NewContent: "x", Rationale: "r", Confidence: 0.9}
			_, err := constrainedClient(srv).OpenPR(context.Background(), p, "JDWLABS-500")
			if !errors.Is(err, ErrPathNotAllowed) {
				t.Fatalf("want ErrPathNotAllowed, got %v", err)
			}
			if len(*calls) != 0 {
				t.Fatalf("unwatched path %q reached the GitHub API: %+v", bad, *calls)
			}
		})
	}
}

// A path that matches the layout but does not exist on the base branch is
// still an invention; the arm edits files, it never creates them.
func TestGitHubOpenPRRejectsNonexistentFile(t *testing.T) {
	srv, calls := scriptedGitHub(t, http.StatusNotFound, http.StatusCreated)
	defer srv.Close()

	p := Patch{Repo: "jdwlabs/platform", FilePath: "tenants/platform/services/nope/values.yaml", NewContent: "x", Rationale: "r", Confidence: 0.9}
	_, err := constrainedClient(srv).OpenPR(context.Background(), p, "JDWLABS-500")
	if !errors.Is(err, ErrPathNotAllowed) {
		t.Fatalf("want ErrPathNotAllowed, got %v", err)
	}
	assertNoWrites(t, *calls)
}

// A branch that already exists and already has a PR carries an earlier
// proposal under review. The arm must not push onto it: one PR, one commit.
func TestGitHubOpenPRRefusesExistingBranch(t *testing.T) {
	srv, calls := scriptedGitHub(t, http.StatusOK, http.StatusUnprocessableEntity, true)
	defer srv.Close()

	p := Patch{Repo: "jdwlabs/platform", FilePath: "tenants/platform/services/vault/values.yaml", NewContent: "x", Rationale: "r", Confidence: 0.9}
	_, err := constrainedClient(srv).OpenPR(context.Background(), p, "JDWLABS-500")
	if !errors.Is(err, ErrBranchExists) {
		t.Fatalf("want ErrBranchExists, got %v", err)
	}
	if errors.Is(err, ErrBranchOrphaned) {
		t.Fatalf("a branch with an existing PR must not also read as orphaned: %v", err)
	}
	for _, c := range *calls {
		if c.method == http.MethodPut || (c.method == http.MethodPost && strings.HasSuffix(c.path, "/pulls")) {
			t.Fatalf("wrote onto an existing branch: %s %s", c.method, c.path)
		}
	}
}

// A branch that already exists but carries no pull request in any state is
// not "already in review" — it is the remnant of a run that failed after
// creating the branch (a dead context, a GitHub 5xx on the PUT or the pulls
// create) and never opened one. This must be its own, loud outcome: silently
// folding it into the ordinary existing-branch skip would leave a real,
// unreviewed commit sitting unseen.
func TestGitHubOpenPRFlagsOrphanedBranchWhenNoPRExists(t *testing.T) {
	srv, calls := scriptedGitHub(t, http.StatusOK, http.StatusUnprocessableEntity, false)
	defer srv.Close()

	p := Patch{Repo: "jdwlabs/platform", FilePath: "tenants/platform/services/vault/values.yaml", NewContent: "x", Rationale: "r", Confidence: 0.9}
	_, err := constrainedClient(srv).OpenPR(context.Background(), p, "JDWLABS-500")
	if !errors.Is(err, ErrBranchOrphaned) {
		t.Fatalf("want ErrBranchOrphaned, got %v", err)
	}
	if errors.Is(err, ErrBranchExists) {
		t.Fatalf("an orphaned branch must not also read as already-in-review: %v", err)
	}
	for _, c := range *calls {
		if c.method == http.MethodPut || (c.method == http.MethodPost && strings.HasSuffix(c.path, "/pulls")) {
			t.Fatalf("wrote onto an orphaned branch: %s %s", c.method, c.path)
		}
	}
}

// The scenario that produces an orphaned branch in the first place: the
// branch is created fresh this run (201), the file is written, and only the
// pull request creation fails. The commit is real and left behind; a later
// retry of the same ticket is what TestGitHubOpenPRFlagsOrphanedBranchWhenNoPRExists
// covers.
func TestGitHubOpenPRErrorsWhenPRCreationFailsAfterCommit(t *testing.T) {
	var calls []ghCall
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, ghCall{r.Method, r.URL.Path, nil})
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/git/ref/heads/main"):
			_ = json.NewEncoder(w).Encode(map[string]any{"object": map[string]string{"sha": "basesha"}})
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/contents/"):
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"sha": "filesha", "type": "file"})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/git/refs"):
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/contents/"):
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/pulls"):
			w.WriteHeader(http.StatusInternalServerError)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	p := Patch{Repo: "jdwlabs/platform", FilePath: "tenants/platform/services/vault/values.yaml", NewContent: "x", Rationale: "r", Confidence: 0.9}
	_, err := constrainedClient(srv).OpenPR(context.Background(), p, "JDWLABS-500")
	if err == nil {
		t.Fatal("expected an error when pull request creation fails")
	}
	var puts int
	for _, c := range calls {
		if c.method == http.MethodPut {
			puts++
		}
	}
	if puts != 1 {
		t.Fatalf("the commit must have landed even though no PR was opened, got %d PUTs", puts)
	}
}

// The happy path updates in place: the PUT carries the blob SHA read from the
// base branch, which is what makes the Contents API edit rather than create.
func TestGitHubOpenPRUpdatesExistingFileInPlace(t *testing.T) {
	srv, calls := scriptedGitHub(t, http.StatusOK, http.StatusCreated)
	defer srv.Close()

	p := Patch{Repo: "jdwlabs/platform", FilePath: "tenants/platform/services/vault/postInstall/externalsecret.yaml", NewContent: "x", Rationale: "r", Confidence: 0.9}
	if _, err := constrainedClient(srv).OpenPR(context.Background(), p, "JDWLABS-500"); err != nil {
		t.Fatal(err)
	}
	var puts int
	for _, c := range *calls {
		if c.method == http.MethodPut {
			puts++
			if sha, _ := c.body["sha"].(string); sha != "filesha" {
				t.Fatalf("PUT without the existing blob sha: %+v", c.body)
			}
		}
	}
	if puts != 1 {
		t.Fatalf("want exactly one file write, got %d", puts)
	}
}

// The denylist is a known exception to the allowlist: a path an allow glob
// matches (a dormant release's values.yaml, typically) that is not actually
// reconciled. It must win even though the allow glob also matches, and it
// must be distinguishable from an ordinary allowlist rejection.
func TestGitHubOpenPRDenylistWinsOverAllowlist(t *testing.T) {
	srv, calls := scriptedGitHub(t, http.StatusOK, http.StatusCreated)
	defer srv.Close()

	deny := []string{"tenants/*/services/arc-systems/*.yaml", "tenants/*/services/arc-systems/postInstall/*.yaml"}
	g := NewGitHubClient(srv.URL, StaticGitHubToken("ghtok"), []string{"jdwlabs/platform"}, testPathGlobs, deny, srv.Client())
	p := Patch{Repo: "jdwlabs/platform", FilePath: "tenants/platform/services/arc-systems/values.yaml", NewContent: "x", Rationale: "r", Confidence: 0.9}
	_, err := g.OpenPR(context.Background(), p, "JDWLABS-500")
	if !errors.Is(err, ErrPathDenied) {
		t.Fatalf("want ErrPathDenied, got %v", err)
	}
	if errors.Is(err, ErrPathNotAllowed) {
		t.Fatalf("a denylist hit must not also read as a plain allowlist rejection: %v", err)
	}
	if len(*calls) != 0 {
		t.Fatalf("denied path reached the GitHub API: %+v", *calls)
	}
}

// No path allowlist means no PR arm, the same fail-closed posture as the repo
// allowlist, and it must not read as a rejected proposal.
func TestGitHubOpenPRDisabledWithoutPathAllowlist(t *testing.T) {
	srv, calls := scriptedGitHub(t, http.StatusOK, http.StatusCreated)
	defer srv.Close()

	p := Patch{Repo: "jdwlabs/platform", FilePath: "tenants/platform/services/vault/values.yaml", NewContent: "x"}
	_, err := NewGitHubClient(srv.URL, StaticGitHubToken("ghtok"), []string{"jdwlabs/platform"}, nil, nil, srv.Client()).OpenPR(context.Background(), p, "JDWLABS-500")
	if err == nil || errors.Is(err, ErrPathNotAllowed) {
		t.Fatalf("want a disabled-arm error, got %v", err)
	}
	if len(*calls) != 0 {
		t.Fatalf("disabled arm reached the GitHub API: %+v", *calls)
	}
}

// The model is told the watched layout and told not to invent one; the gate
// above is the boundary, but the instruction is what keeps proposals useful.
func TestPatchSystemPromptNamesWatchedLayout(t *testing.T) {
	for _, targets := range [][]string{nil, {"jdwlabs/platform"}, {"jdwlabs/platform", "jdwlabs/deployments"}} {
		got := patchSystemPrompt(targets)
		for _, want := range []string{
			"tenants/<tenant>/services/<release>/values.yaml",
			"tenants/<tenant>/services/<release>/postInstall/<name>.yaml",
			"NEVER create a new file",
			"NEVER propose tenants/<tenant>/tenant.yaml itself",
		} {
			if !strings.Contains(got, want) {
				t.Fatalf("targets %v: prompt lacks %q:\n%s", targets, want, got)
			}
		}
	}
}
