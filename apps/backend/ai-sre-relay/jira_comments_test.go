package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type commentLog struct {
	bodies []string
}

// newCommentRecorder accepts comments and, for a transition lookup, a Done
// transition, so Close can run end to end.
func newCommentRecorder(t *testing.T, log *commentLog) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/comment"):
			log.bodies = append(log.bodies, r.URL.Path)
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/transitions"):
			_, _ = w.Write([]byte(`{"transitions":[{"id":"31","name":"Done","to":{"name":"Done","statusCategory":{"key":"done"}}}]}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/transitions"):
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
}

func newClockedJira(srv *httptest.Server) (*JiraClient, *time.Time) {
	j := NewJiraClient(srv.URL, "e@x", "tok", "JDWLABS", "Task", srv.Client())
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	j.now = func() time.Time { return at }
	return j, &at
}

func TestJiraAnalysisComments(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name  string
		steps func(j *JiraClient, at *time.Time)
		want  int
	}{
		{"a different analysis right after a note still posts", func(j *JiraClient, _ *time.Time) {
			_ = j.NoteRefire(ctx, "K-1", 1)
			_, _ = j.groupInto(ctx, "K-1", Analysis{RootCause: "cause A"})
			_, _ = j.groupInto(ctx, "K-1", Analysis{RootCause: "cause B"})
		}, 3},
		{"the same analysis is not reposted", func(j *JiraClient, at *time.Time) {
			_, _ = j.groupInto(ctx, "K-1", Analysis{RootCause: "cause A"})
			*at = at.Add(24 * time.Hour)
			_, _ = j.updateIssue(ctx, "K-1", Analysis{RootCause: "cause A"}, false)
		}, 1},
		{"a changed analysis posts after an unchanged one was withheld", func(j *JiraClient, _ *time.Time) {
			_, _ = j.updateIssue(ctx, "K-1", Analysis{RootCause: "cause A"}, false)
			_, _ = j.updateIssue(ctx, "K-1", Analysis{RootCause: "cause A"}, false)
			_, _ = j.updateIssue(ctx, "K-1", Analysis{RootCause: "cause B"}, false)
		}, 2},
		{"new analyses stop at the ceiling", func(j *JiraClient, _ *time.Time) {
			for n := range maxNoisyCommentsPerTicket + 20 {
				_, _ = j.updateIssue(ctx, "K-1", Analysis{RootCause: fmt.Sprintf("cause %d", n)}, false)
			}
		}, maxNoisyCommentsPerTicket},
		{"a notes-and-analyses mix shares one ceiling", func(j *JiraClient, at *time.Time) {
			for n := range maxNoisyCommentsPerTicket {
				*at = at.Add(noisyCommentInterval + time.Minute)
				_ = j.NoteRefire(ctx, "K-1", n)
			}
			_, _ = j.updateIssue(ctx, "K-1", Analysis{RootCause: "late analysis"}, false)
		}, maxNoisyCommentsPerTicket},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var log commentLog
			srv := newCommentRecorder(t, &log)
			defer srv.Close()
			j, at := newClockedJira(srv)
			tc.steps(j, at)
			if len(log.bodies) != tc.want {
				t.Fatalf("posted %d comments, want %d", len(log.bodies), tc.want)
			}
		})
	}
}

// A full table must evict rather than stop tracking, or every ticket past the
// cap would be unlimited.
func TestJiraNoisyTableEvictsInsteadOfFailingOpen(t *testing.T) {
	var log commentLog
	srv := newCommentRecorder(t, &log)
	defer srv.Close()
	j, at := newClockedJira(srv)
	ctx := context.Background()

	for n := range maxTrackedDedupKeys {
		*at = at.Add(time.Second)
		_ = j.NoteRefire(ctx, IssueKey(fmt.Sprintf("OLD-%d", n)), 1)
	}
	*at = at.Add(time.Second)
	_ = j.NoteRefire(ctx, "NEW-1", 1)
	if len(j.noisy) != maxTrackedDedupKeys {
		t.Fatalf("table size = %d, want it held at %d", len(j.noisy), maxTrackedDedupKeys)
	}
	if _, ok := j.noisy["OLD-0"]; ok {
		t.Fatal("the least recently used ticket was not evicted")
	}
	before := len(log.bodies)
	_ = j.NoteRefire(ctx, "NEW-1", 2)
	if len(log.bodies) != before {
		t.Fatal("a ticket added past the cap was not rate limited")
	}
}

func TestJiraCloseDropsCommentState(t *testing.T) {
	var log commentLog
	srv := newCommentRecorder(t, &log)
	defer srv.Close()
	j, at := newClockedJira(srv)
	ctx := context.Background()

	_ = j.NoteRefire(ctx, "K-1", 1)
	if err := j.Close(ctx, "K-1", *at); err != nil {
		t.Fatal(err)
	}
	if _, ok := j.noisy["K-1"]; ok {
		t.Fatal("closed ticket still holds comment state")
	}
}
