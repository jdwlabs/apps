package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestJiraUpsertCreatesWhenNoDuplicate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			t.Errorf("missing Authorization header on %s %s", r.Method, r.URL.Path)
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/rest/api/3/search/jql":
			if r.URL.Query().Get("fields") != "key,status" {
				t.Errorf("search must request the key and status fields explicitly, got %q", r.URL.RawQuery)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"issues": []any{}})
		case r.Method == http.MethodPost && r.URL.Path == "/rest/api/3/issue":
			raw, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(raw), "amfp-fp1") {
				t.Errorf("create missing dedup label: %s", raw)
			}
			if !strings.Contains(string(raw), `"name":"Task"`) {
				t.Errorf("create must use the configured issue type: %s", raw)
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]string{"key": "JDWLABS-500"})
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	key, err := NewJiraClient(srv.URL, "e@x", "tok", "JDWLABS", "Task", srv.Client()).
		Upsert(context.Background(), Alert{Fingerprint: "fp1", Labels: map[string]string{"alertname": "X"}}, Analysis{RootCause: "y"})
	if err != nil || key != "JDWLABS-500" {
		t.Fatalf("key=%q err=%v", key, err)
	}
}

func TestJiraUpsertCommentsWhenDuplicate(t *testing.T) {
	commented := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			t.Errorf("missing Authorization header on %s %s", r.Method, r.URL.Path)
		}
		switch {
		case strings.HasPrefix(r.URL.Path, "/rest/api/3/search"):
			_ = json.NewEncoder(w).Encode(map[string]any{"issues": []map[string]string{{"key": "JDWLABS-77"}}})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/comment"):
			commented = true
			w.WriteHeader(http.StatusCreated)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	key, err := NewJiraClient(srv.URL, "e@x", "tok", "JDWLABS", "Task", srv.Client()).
		Upsert(context.Background(), Alert{Fingerprint: "fp1"}, Analysis{RootCause: "y"})
	if err != nil || key != "JDWLABS-77" || !commented {
		t.Fatalf("key=%q err=%v commented=%v", key, err, commented)
	}
}

// TestJiraUpsertReopensDoneDuplicate reproduces JDWLABS-126: Alertmanager
// fingerprints are stable per labelset, so once a human closes the Jira
// ticket for a fingerprint, the same alert re-firing must reopen the
// original ticket rather than spawn a duplicate.
func TestJiraUpsertReopensDoneDuplicate(t *testing.T) {
	created, transitioned, commented := false, false, false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			t.Errorf("missing Authorization header on %s %s", r.Method, r.URL.Path)
		}
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/rest/api/3/search"):
			jql, _ := url.QueryUnescape(r.URL.Query().Get("jql"))
			if strings.Contains(jql, "statusCategory != Done") {
				// Real Jira excludes Done issues from this clause: the old
				// ticket for this fingerprint is Done, so it never matches.
				_ = json.NewEncoder(w).Encode(map[string]any{"issues": []any{}})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"issues": []map[string]any{{
				"key":    "JDWLABS-107",
				"fields": map[string]any{"status": map[string]any{"statusCategory": map[string]any{"key": "done"}}},
			}}})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/transitions"):
			_ = json.NewEncoder(w).Encode(map[string]any{"transitions": []map[string]any{
				{"id": "31", "name": "Done", "to": map[string]any{"statusCategory": map[string]any{"key": "done"}}},
				{"id": "11", "name": "Reopen", "to": map[string]any{"statusCategory": map[string]any{"key": "new"}}},
			}})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/transitions"):
			raw, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(raw), `"id":"11"`) {
				t.Errorf("expected transition to the non-Done id 11, got %s", raw)
			}
			transitioned = true
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/comment"):
			commented = true
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodPost && r.URL.Path == "/rest/api/3/issue":
			created = true
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]string{"key": "JDWLABS-500"})
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	key, err := NewJiraClient(srv.URL, "e@x", "tok", "JDWLABS", "Task", srv.Client()).
		Upsert(context.Background(), Alert{Fingerprint: "9b76534c7edc3c13", Labels: map[string]string{"alertname": "X"}}, Analysis{RootCause: "y"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created {
		t.Fatal("must reopen the Done issue, not create a duplicate")
	}
	if key != "JDWLABS-107" {
		t.Fatalf("expected the original issue key JDWLABS-107, got %q", key)
	}
	if !transitioned {
		t.Fatal("expected the Done issue to be transitioned back to an open status")
	}
	if !commented {
		t.Fatal("expected a comment on the reopened issue")
	}
}

func TestJiraUpsertSearchErrorsOnServerError(t *testing.T) {
	created := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/rest/api/3/search"):
			w.WriteHeader(http.StatusInternalServerError)
		case r.Method == http.MethodPost && r.URL.Path == "/rest/api/3/issue":
			created = true
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]string{"key": "JDWLABS-999"})
		}
	}))
	defer srv.Close()

	_, err := NewJiraClient(srv.URL, "e@x", "tok", "JDWLABS", "Task", srv.Client()).
		Upsert(context.Background(), Alert{Fingerprint: "fp1", Labels: map[string]string{"alertname": "X"}}, Analysis{RootCause: "y"})
	if err == nil {
		t.Fatal("expected error when search returns 500, got nil")
	}
	if created {
		t.Fatal("create must not be attempted after a failed search")
	}
}

func TestJiraUpsertCreateErrorsOnServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			t.Errorf("missing Authorization header on %s %s", r.Method, r.URL.Path)
		}
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/rest/api/3/search"):
			_ = json.NewEncoder(w).Encode(map[string]any{"issues": []any{}})
		case r.Method == http.MethodPost && r.URL.Path == "/rest/api/3/issue":
			w.WriteHeader(http.StatusInternalServerError)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	key, err := NewJiraClient(srv.URL, "e@x", "tok", "JDWLABS", "Task", srv.Client()).
		Upsert(context.Background(), Alert{Fingerprint: "fp1", Labels: map[string]string{"alertname": "X"}}, Analysis{RootCause: "y"})
	if err == nil {
		t.Fatalf("expected error on server failure, got key=%q", key)
	}
}

// TestJiraUpsertGroupsSameAlertnameOpenTicket replays the observed duplicate
// pair: the same alert re-fired under a different labelset, so its fingerprint
// (and dedup label) changed and the fingerprint search found nothing. With an
// open ticket for the same alertname, the relay must comment there instead of
// filing a sibling duplicate.
func TestJiraUpsertGroupsSameAlertnameOpenTicket(t *testing.T) {
	created, commented := false, false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/rest/api/3/search"):
			jql, _ := url.QueryUnescape(r.URL.Query().Get("jql"))
			if strings.Contains(jql, "amfp-") {
				// New fingerprint: no issue carries its label yet.
				_ = json.NewEncoder(w).Encode(map[string]any{"issues": []any{}})
				return
			}
			if !strings.Contains(jql, `"amalert-targetdown"`) || !strings.Contains(jql, "statusCategory != Done") {
				t.Errorf("alertname search must be label-scoped and open-only, got jql %q", jql)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"issues": []map[string]any{{
				"key":    "JDWLABS-152",
				"fields": map[string]any{"status": map[string]any{"statusCategory": map[string]any{"key": "indeterminate"}}},
			}}})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/comment"):
			commented = true
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodPost && r.URL.Path == "/rest/api/3/issue":
			created = true
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]string{"key": "JDWLABS-153"})
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	key, err := NewJiraClient(srv.URL, "e@x", "tok", "JDWLABS", "Task", srv.Client()).
		Upsert(context.Background(), Alert{Fingerprint: "7e2b7cad96ebf3c4", Labels: map[string]string{"alertname": "TargetDown"}}, Analysis{RootCause: "y"})
	if err != nil {
		t.Fatal(err)
	}
	if created {
		t.Fatal("must group into the open same-alertname ticket, not create a duplicate")
	}
	if key != "JDWLABS-152" || !commented {
		t.Fatalf("expected comment on JDWLABS-152, got key=%q commented=%v", key, commented)
	}
}

// TestJiraUpsertDedupSurvivesSearchIndexLag replays the create-then-refire
// race: Jira's JQL index is eventually consistent, so searches keep returning
// empty for a while after a create. Follow-up upserts — same fingerprint or a
// new fingerprint of the same alertname — must land on the issue this process
// just created, not create duplicates.
func TestJiraUpsertDedupSurvivesSearchIndexLag(t *testing.T) {
	creates, comments := 0, 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/rest/api/3/search"):
			// Simulated index lag: the created issue is never searchable.
			_ = json.NewEncoder(w).Encode(map[string]any{"issues": []any{}})
		case r.Method == http.MethodGet && r.URL.Path == "/rest/api/3/issue/JDWLABS-160":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"fields": map[string]any{"status": map[string]any{"statusCategory": map[string]any{"key": "indeterminate"}}},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/rest/api/3/issue/JDWLABS-160/comment":
			comments++
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodPost && r.URL.Path == "/rest/api/3/issue":
			creates++
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]string{"key": "JDWLABS-160"})
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	j := NewJiraClient(srv.URL, "e@x", "tok", "JDWLABS", "Task", srv.Client())
	an := Analysis{RootCause: "y"}

	key, err := j.Upsert(context.Background(), Alert{Fingerprint: "fp-a", Labels: map[string]string{"alertname": "TargetDown"}}, an)
	if err != nil || key != "JDWLABS-160" {
		t.Fatalf("first upsert: key=%q err=%v", key, err)
	}
	// Same fingerprint re-fires while the index still lags.
	key, err = j.Upsert(context.Background(), Alert{Fingerprint: "fp-a", Labels: map[string]string{"alertname": "TargetDown"}}, an)
	if err != nil || key != "JDWLABS-160" {
		t.Fatalf("same-fingerprint refire: key=%q err=%v", key, err)
	}
	// A different labelset of the same alert fires while the index still lags.
	key, err = j.Upsert(context.Background(), Alert{Fingerprint: "fp-b", Labels: map[string]string{"alertname": "TargetDown"}}, an)
	if err != nil || key != "JDWLABS-160" {
		t.Fatalf("same-alertname refire: key=%q err=%v", key, err)
	}

	if creates != 1 {
		t.Fatalf("created %d issues, want exactly 1", creates)
	}
	if comments != 2 {
		t.Fatalf("expected both refires to comment (2), got %d", comments)
	}
}

// TestJiraKnownMapIsBounded mirrors pipeline.go's maxTrackedFirings discipline
// for the active map: known must stop growing once it hits its cap rather
// than track every dedup key ever seen for the life of the process.
func TestJiraKnownMapIsBounded(t *testing.T) {
	created := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/rest/api/3/search"):
			_ = json.NewEncoder(w).Encode(map[string]any{"issues": []any{}})
		case r.Method == http.MethodPost && r.URL.Path == "/rest/api/3/issue":
			created++
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]string{"key": fmt.Sprintf("JDWLABS-%d", 9000+created)})
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	j := NewJiraClient(srv.URL, "e@x", "tok", "JDWLABS", "Task", srv.Client())
	// Pre-fill known to the cap so the next Upsert's writes exercise the
	// bound, without paying for maxTrackedDedupKeys real HTTP round trips.
	for i := 0; i < maxTrackedDedupKeys; i++ {
		j.known[fmt.Sprintf("preexisting-%d", i)] = IssueKey("JDWLABS-0")
	}

	a := Alert{Fingerprint: "new-fp", Labels: map[string]string{"alertname": "New"}}
	if _, err := j.Upsert(context.Background(), a, Analysis{RootCause: "y"}); err != nil {
		t.Fatal(err)
	}

	if len(j.known) != maxTrackedDedupKeys {
		t.Fatalf("known map grew past the cap: len=%d, want %d", len(j.known), maxTrackedDedupKeys)
	}
	if _, ok := j.known[j.label(a)]; ok {
		t.Fatal("a new dedup key must not be tracked once the cap is hit")
	}
	if _, ok := j.known[j.alertLabel(a)]; ok {
		t.Fatal("a new alertname dedup key must not be tracked once the cap is hit")
	}
}
