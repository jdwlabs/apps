package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeEnqueuer struct{ got []Alert }

func (f *fakeEnqueuer) enqueue(a Alert) bool { f.got = append(f.got, a); return true }

func TestHealthz(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	newRouter(&fakeEnqueuer{}, &counters{}, "").ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("healthz = %d, want 200", rr.Code)
	}
	if got := rr.Body.String(); got != "ok" {
		t.Errorf("healthz body = %q, want \"ok\"", got)
	}
}

func TestWebhookEnqueuesOnlyFiringAlerts(t *testing.T) {
	e := &fakeEnqueuer{}
	const payload = `{"alerts":[
	  {"status":"firing","fingerprint":"f1","labels":{"alertname":"A"}},
	  {"status":"resolved","fingerprint":"f2","labels":{"alertname":"B"}}
	]}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(payload))
	newRouter(e, &counters{}, "").ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("code = %d, want 202", rr.Code)
	}
	if len(e.got) != 1 || e.got[0].Fingerprint != "f1" {
		t.Fatalf("enqueued = %+v, want only f1", e.got)
	}
}

func TestWebhookBadPayload(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader("{not json"))
	newRouter(&fakeEnqueuer{}, &counters{}, "").ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("code = %d, want 400", rr.Code)
	}
}

func TestWebhookAuth(t *testing.T) {
	const token = "s3cret"
	const payload = `{"alerts":[{"status":"firing","fingerprint":"f1","labels":{"alertname":"A"}}]}`

	// Missing token -> 401, nothing enqueued.
	e := &fakeEnqueuer{}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(payload))
	newRouter(e, &counters{}, token).ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("missing-token code = %d, want 401", rr.Code)
	}
	if len(e.got) != 0 {
		t.Fatalf("unauthorized request must not enqueue: %+v", e.got)
	}

	// Correct token -> 202, enqueued.
	e2 := &fakeEnqueuer{}
	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(payload))
	req2.Header.Set("Authorization", "Bearer "+token)
	newRouter(e2, &counters{}, token).ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusAccepted {
		t.Fatalf("token code = %d, want 202", rr2.Code)
	}
	if len(e2.got) != 1 {
		t.Fatalf("authorized request must enqueue one, got %+v", e2.got)
	}

	// Healthz stays open even when a token is configured (kubelet probes).
	rr3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	newRouter(&fakeEnqueuer{}, &counters{}, token).ServeHTTP(rr3, req3)
	if rr3.Code != http.StatusOK {
		t.Fatalf("healthz with token = %d, want 200", rr3.Code)
	}

	// Metrics stays open too, so Prometheus can scrape without the webhook
	// credential.
	rr4 := httptest.NewRecorder()
	req4 := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	newRouter(&fakeEnqueuer{}, &counters{}, token).ServeHTTP(rr4, req4)
	if rr4.Code != http.StatusOK {
		t.Fatalf("metrics with token = %d, want 200", rr4.Code)
	}
}

func TestMetricsExposesBothCounters(t *testing.T) {
	c := &counters{}
	c.investigationsRun.Add(3)
	c.repeatsSkipped.Add(7)

	rr := httptest.NewRecorder()
	newRouter(&fakeEnqueuer{}, c, "").ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	body := rr.Body.String()
	for _, want := range []string{
		"ai_sre_relay_investigations_run_total 3",
		"ai_sre_relay_repeats_skipped_total 7",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics missing %q in:\n%s", want, body)
		}
	}
}
