package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
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

func TestJiraNoteResolvedRecordsTheResolveTime(t *testing.T) {
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/comment"):
			raw, _ := io.ReadAll(r.Body)
			body = string(raw)
			w.WriteHeader(http.StatusCreated)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	at := time.Date(2026, 7, 28, 4, 0, 0, 0, time.UTC)
	err := NewJiraClient(srv.URL, "e@x", "tok", "JDWLABS", "Task", srv.Client()).
		NoteResolved(context.Background(), "JDWLABS-88", at)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "2026-07-28T04:00:00Z") {
		t.Fatalf("resolve note omits the resolve time: %s", body)
	}
}

// The closing transition is looked up by the target status category, like the
// reopen path: workflows name the transition differently and the id is not
// stable across projects.
func TestJiraCloseUsesTheDoneCategoryTransition(t *testing.T) {
	var comment, transitionID string
	order := []string{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/comment"):
			raw, _ := io.ReadAll(r.Body)
			comment = string(raw)
			order = append(order, "comment")
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/transitions"):
			_ = json.NewEncoder(w).Encode(map[string]any{"transitions": []map[string]any{
				{"id": "11", "name": "Back to Backlog", "to": map[string]any{"statusCategory": map[string]any{"key": "new"}}},
				{"id": "31", "name": "Ship It", "to": map[string]any{"statusCategory": map[string]any{"key": "done"}}},
			}})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/transitions"):
			var payload struct {
				Transition struct {
					ID string `json:"id"`
				} `json:"transition"`
			}
			_ = json.NewDecoder(r.Body).Decode(&payload)
			transitionID = payload.Transition.ID
			order = append(order, "transition")
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	at := time.Date(2026, 7, 28, 4, 0, 0, 0, time.UTC)
	if err := NewJiraClient(srv.URL, "e@x", "tok", "JDWLABS", "Task", srv.Client()).
		Close(context.Background(), "JDWLABS-89", at); err != nil {
		t.Fatal(err)
	}
	if transitionID != "31" {
		t.Fatalf("transition id = %q, want the Done-category transition 31", transitionID)
	}
	if !strings.Contains(comment, "2026-07-28T04:00:00Z") {
		t.Fatalf("closing comment omits the resolve time: %s", comment)
	}
	if len(order) != 2 || order[0] != "transition" {
		t.Fatalf("order = %v, want the transition before the closing comment", order)
	}
}

// A workflow with no reachable Done transition must surface an error rather
// than report a ticket closed that is still open.
func TestJiraCloseFailsWithoutDoneTransition(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/comment"):
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/transitions"):
			_ = json.NewEncoder(w).Encode(map[string]any{"transitions": []map[string]any{
				{"id": "11", "name": "Back to Backlog", "to": map[string]any{"statusCategory": map[string]any{"key": "new"}}},
			}})
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	err := NewJiraClient(srv.URL, "e@x", "tok", "JDWLABS", "Task", srv.Client()).
		Close(context.Background(), "JDWLABS-90", time.Now())
	if err == nil {
		t.Fatal("close must fail when no Done transition is available")
	}
}

// After a restart the relay has no in-process mapping, so a resolve has to
// recover the ticket the same way the firing path does: by the fingerprint
// label, with the same query, so the two paths cannot select different issues.
func TestJiraFindOpenByFingerprintSearchesTheFingerprintLabel(t *testing.T) {
	var jql string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/rest/api/3/search") {
			jql, _ = url.QueryUnescape(r.URL.Query().Get("jql"))
			_ = json.NewEncoder(w).Encode(map[string]any{"issues": []map[string]string{{"key": "JDWLABS-91"}}})
			return
		}
		t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
	}))
	defer srv.Close()

	key, err := NewJiraClient(srv.URL, "e@x", "tok", "JDWLABS", "Task", srv.Client()).
		FindOpenByFingerprint(context.Background(), Alert{Fingerprint: "fp1"})
	if err != nil || key != "JDWLABS-91" {
		t.Fatalf("key=%q err=%v", key, err)
	}
	if !strings.Contains(jql, `labels = "amfp-fp1"`) {
		t.Fatalf("search must key on the fingerprint label: %s", jql)
	}
	// The upsert has to see Done tickets in order to reopen them, so the query
	// cannot filter them out on one side only; the Done case is decided on the
	// result instead.
	if strings.Contains(jql, "statusCategory") {
		t.Fatalf("the fingerprint search must match the upsert's, unfiltered: %s", jql)
	}
}

// A Done ticket has nothing left to close, so the resolve path reports
// nothing even though the search matched it.
func TestJiraFindOpenByFingerprintIgnoresADoneMatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"issues": []map[string]any{
			{"key": "JDWLABS-95", "fields": map[string]any{
				"status": map[string]any{"statusCategory": map[string]any{"key": "done"}},
			}},
		}})
	}))
	defer srv.Close()

	key, err := NewJiraClient(srv.URL, "e@x", "tok", "JDWLABS", "Task", srv.Client()).
		FindOpenByFingerprint(context.Background(), Alert{Fingerprint: "fp1"})
	if err != nil || key != "" {
		t.Fatalf("key=%q err=%v, want nothing to close for a Done match", key, err)
	}
}

func TestJiraFindOpenByFingerprintReturnsEmptyOnNoMatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"issues": []any{}})
	}))
	defer srv.Close()

	key, err := NewJiraClient(srv.URL, "e@x", "tok", "JDWLABS", "Task", srv.Client()).
		FindOpenByFingerprint(context.Background(), Alert{Fingerprint: "fp1"})
	if err != nil || key != "" {
		t.Fatalf("key=%q err=%v, want an empty key and no error", key, err)
	}
}

// The transition goes first. Commenting first means a workflow that cannot
// reach Done collects one closing comment per sweep interval, forever.
func TestJiraCloseTransitionsBeforeCommenting(t *testing.T) {
	var order []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/comment"):
			order = append(order, "comment")
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/transitions"):
			_ = json.NewEncoder(w).Encode(map[string]any{"transitions": []map[string]any{
				{"id": "31", "to": map[string]any{"statusCategory": map[string]any{"key": "done"}}},
			}})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/transitions"):
			order = append(order, "transition")
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	if err := NewJiraClient(srv.URL, "e@x", "tok", "JDWLABS", "Task", srv.Client()).
		Close(context.Background(), "JDWLABS-92", time.Now()); err != nil {
		t.Fatal(err)
	}
	if len(order) != 2 || order[0] != "transition" {
		t.Fatalf("order = %v, want the transition before the closing comment", order)
	}
}

// A ticket that transitioned but could not be commented on is closed, not
// pending. Saying so distinctly is what stops the caller retrying a Done
// ticket every sweep.
func TestJiraCloseReportsATransitionedTicketWithNoNote(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/comment"):
			w.WriteHeader(http.StatusForbidden)
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/transitions"):
			_ = json.NewEncoder(w).Encode(map[string]any{"transitions": []map[string]any{
				{"id": "31", "to": map[string]any{"statusCategory": map[string]any{"key": "done"}}},
			}})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/transitions"):
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	err := NewJiraClient(srv.URL, "e@x", "tok", "JDWLABS", "Task", srv.Client()).
		Close(context.Background(), "JDWLABS-93", time.Now())
	if !errors.Is(err, ErrClosedWithoutNote) {
		t.Fatalf("err = %v, want ErrClosedWithoutNote", err)
	}
}

// A close that raced a re-fire has to be undone, with the reason on the
// ticket: it is Done while its alert is firing.
func TestJiraReopenMovesTicketOutOfDone(t *testing.T) {
	var transitionID, comment string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Query().Get("fields") == "status":
			_ = json.NewEncoder(w).Encode(map[string]any{"fields": map[string]any{
				"status": map[string]any{"statusCategory": map[string]any{"key": "done"}},
			}})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/comment"):
			raw, _ := io.ReadAll(r.Body)
			comment = string(raw)
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/transitions"):
			_ = json.NewEncoder(w).Encode(map[string]any{"transitions": []map[string]any{
				{"id": "31", "to": map[string]any{"statusCategory": map[string]any{"key": "done"}}},
				{"id": "11", "to": map[string]any{"statusCategory": map[string]any{"key": "new"}}},
			}})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/transitions"):
			var payload struct {
				Transition struct {
					ID string `json:"id"`
				} `json:"transition"`
			}
			_ = json.NewDecoder(r.Body).Decode(&payload)
			transitionID = payload.Transition.ID
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	if err := NewJiraClient(srv.URL, "e@x", "tok", "JDWLABS", "Task", srv.Client()).
		Reopen(context.Background(), "JDWLABS-94"); err != nil {
		t.Fatal(err)
	}
	if transitionID != "11" {
		t.Fatalf("transition id = %q, want the non-Done transition 11", transitionID)
	}
	if comment == "" {
		t.Fatal("a reopen must say why the ticket came back")
	}
}

// Reopening a ticket that is not Done would push it to whatever open status
// the workflow happens to list first and comment about a reopen that never
// happened. The close being undone may simply never have landed.
func TestJiraReopenLeavesAnOpenTicketAlone(t *testing.T) {
	touched := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Query().Get("fields") == "status" {
			_ = json.NewEncoder(w).Encode(map[string]any{"fields": map[string]any{
				"status": map[string]any{"statusCategory": map[string]any{"key": "indeterminate"}},
			}})
			return
		}
		touched = true
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	if err := NewJiraClient(srv.URL, "e@x", "tok", "JDWLABS", "Task", srv.Client()).
		Reopen(context.Background(), "JDWLABS-96"); err != nil {
		t.Fatal(err)
	}
	if touched {
		t.Fatal("an already-open ticket must not be transitioned or commented on")
	}
}

// "Won't Do" and "Duplicate" sit in the Done category too. Closing an incident
// as a duplicate says something the relay does not mean, so the named status
// wins over whichever Done-category transition the workflow lists first.
func TestJiraClosePrefersTheNamedDoneStatus(t *testing.T) {
	var transitionID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/comment"):
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/transitions"):
			_ = json.NewEncoder(w).Encode(map[string]any{"transitions": []map[string]any{
				{"id": "41", "name": "Won't Do", "to": map[string]any{"name": "Won't Do", "statusCategory": map[string]any{"key": "done"}}},
				{"id": "51", "name": "Duplicate", "to": map[string]any{"name": "Duplicate", "statusCategory": map[string]any{"key": "done"}}},
				{"id": "31", "name": "Finish", "to": map[string]any{"name": "Done", "statusCategory": map[string]any{"key": "done"}}},
			}})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/transitions"):
			var payload struct {
				Transition struct {
					ID string `json:"id"`
				} `json:"transition"`
			}
			_ = json.NewDecoder(r.Body).Decode(&payload)
			transitionID = payload.Transition.ID
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	if err := NewJiraClient(srv.URL, "e@x", "tok", "JDWLABS", "Task", srv.Client()).
		Close(context.Background(), "JDWLABS-97", time.Now()); err != nil {
		t.Fatal(err)
	}
	if transitionID != "31" {
		t.Fatalf("transition id = %q, want 31 (target status \"Done\"), not the first Done-category one", transitionID)
	}
}

// A workflow that calls it something else still closes, by category.
func TestJiraCloseFallsBackToTheDoneCategory(t *testing.T) {
	var transitionID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/comment"):
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/transitions"):
			_ = json.NewEncoder(w).Encode(map[string]any{"transitions": []map[string]any{
				{"id": "61", "name": "Complete", "to": map[string]any{"name": "Completed", "statusCategory": map[string]any{"key": "done"}}},
			}})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/transitions"):
			var payload struct {
				Transition struct {
					ID string `json:"id"`
				} `json:"transition"`
			}
			_ = json.NewDecoder(r.Body).Decode(&payload)
			transitionID = payload.Transition.ID
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	if err := NewJiraClient(srv.URL, "e@x", "tok", "JDWLABS", "Task", srv.Client()).
		Close(context.Background(), "JDWLABS-98", time.Now()); err != nil {
		t.Fatal(err)
	}
	if transitionID != "61" {
		t.Fatalf("transition id = %q, want the Done-category fallback 61", transitionID)
	}
}

// The status name is configurable: not every project calls it "Done".
func TestJiraCloseHonoursAConfiguredDoneStatus(t *testing.T) {
	var transitionID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/comment"):
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/transitions"):
			_ = json.NewEncoder(w).Encode(map[string]any{"transitions": []map[string]any{
				{"id": "31", "name": "Finish", "to": map[string]any{"name": "Done", "statusCategory": map[string]any{"key": "done"}}},
				{"id": "71", "name": "Ship", "to": map[string]any{"name": "Shipped", "statusCategory": map[string]any{"key": "done"}}},
			}})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/transitions"):
			var payload struct {
				Transition struct {
					ID string `json:"id"`
				} `json:"transition"`
			}
			_ = json.NewDecoder(r.Body).Decode(&payload)
			transitionID = payload.Transition.ID
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	if err := NewJiraClient(srv.URL, "e@x", "tok", "JDWLABS", "Task", srv.Client(), withDoneStatus("Shipped")).
		Close(context.Background(), "JDWLABS-99", time.Now()); err != nil {
		t.Fatal(err)
	}
	if transitionID != "71" {
		t.Fatalf("transition id = %q, want the configured status's transition 71", transitionID)
	}
}
