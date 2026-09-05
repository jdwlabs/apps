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

// capacity is the number of alerts this enqueuer accepts before refusing, so a
// test can drive the overflow branch without standing up a worker pool.
// A negative capacity accepts everything.
type fakeEnqueuer struct {
	got      []Alert
	capacity int
	// refuse names fingerprints rejected regardless of capacity. The capacity
	// model alone is monotonic — once full it never accepts again — so it can
	// only ever refuse a suffix of a batch and cannot express a slot freeing
	// mid-loop.
	refuse map[string]bool
}

func (f *fakeEnqueuer) enqueue(a Alert) bool {
	if f.refuse[a.Fingerprint] {
		return false
	}
	if f.capacity >= 0 && len(f.got) >= f.capacity {
		return false
	}
	f.got = append(f.got, a)
	return true
}

func acceptAll() *fakeEnqueuer { return &fakeEnqueuer{capacity: -1} }

func TestHealthz(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	newRouter(acceptAll(), &counters{}, "").ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("healthz = %d, want 200", rr.Code)
	}
	if got := rr.Body.String(); got != "ok" {
		t.Errorf("healthz body = %q, want \"ok\"", got)
	}
}

// A resolved notification is what tells the relay its ticket can be closed, so
// it has to reach the pipeline like a firing one. Anything else Alertmanager
// might send is still ignored.
func TestWebhookEnqueuesFiringAndResolvedAlerts(t *testing.T) {
	e := acceptAll()
	const payload = `{"alerts":[
	  {"status":"firing","fingerprint":"f1","labels":{"alertname":"A"}},
	  {"status":"resolved","fingerprint":"f2","endsAt":"2026-07-28T04:00:00Z","labels":{"alertname":"B"}},
	  {"status":"suppressed","fingerprint":"f3","labels":{"alertname":"C"}}
	]}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(payload))
	newRouter(e, &counters{}, "").ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("code = %d, want 202", rr.Code)
	}
	if len(e.got) != 2 || e.got[0].Fingerprint != "f1" || e.got[1].Fingerprint != "f2" {
		t.Fatalf("enqueued = %+v, want f1 and f2", e.got)
	}
	if e.got[1].EndsAt != "2026-07-28T04:00:00Z" {
		t.Fatalf("resolved alert lost its end time: %+v", e.got[1])
	}
}

func TestWebhookBadPayload(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader("{not json"))
	newRouter(acceptAll(), &counters{}, "").ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("code = %d, want 400", rr.Code)
	}
}

func TestWebhookAuth(t *testing.T) {
	const token = "s3cret"
	const payload = `{"alerts":[{"status":"firing","fingerprint":"f1","labels":{"alertname":"A"}}]}`

	// Missing token -> 401, nothing enqueued.
	e := acceptAll()
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
	e2 := acceptAll()
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

// A full queue must be reported to the sender as a failed delivery. Answering
// 202 while discarding the alert retires the sender's only copy, which is how
// an overflow turns into permanently lost coverage rather than a retry.
func TestWebhookRefusesBatchWhenQueueFull(t *testing.T) {
	e := &fakeEnqueuer{capacity: 2}
	const payload = `{"alerts":[
	  {"status":"firing","fingerprint":"f1","labels":{"alertname":"A"}},
	  {"status":"firing","fingerprint":"f2","labels":{"alertname":"B"}},
	  {"status":"firing","fingerprint":"f3","labels":{"alertname":"C"}}
	]}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(payload))
	newRouter(e, &counters{}, "").ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("code = %d, want 503 so the sender retries the batch", rr.Code)
	}
	if got := rr.Header().Get("Retry-After"); got == "" {
		t.Error("503 must carry Retry-After")
	}
	// Capacity that existed when the batch arrived should still be used; a
	// rejection late in the batch must not discard the alerts that did fit.
	if len(e.got) != 2 {
		t.Fatalf("enqueued %d alerts, want the 2 that fit", len(e.got))
	}
}

// A refusal must not abandon the rest of the batch. A worker finishing
// mid-loop frees a slot the later alerts can still take, and stopping at the
// first refusal would discard that capacity and enlarge the retry the sender
// has to make. The refused fingerprint sits in the middle so that accepting
// what follows it cannot be explained by capacity ordering alone.
func TestWebhookOffersRestOfBatchAfterRefusal(t *testing.T) {
	e := &fakeEnqueuer{capacity: -1, refuse: map[string]bool{"f2": true}}
	const payload = `{"alerts":[
	  {"status":"firing","fingerprint":"f1","labels":{"alertname":"A"}},
	  {"status":"firing","fingerprint":"f2","labels":{"alertname":"B"}},
	  {"status":"firing","fingerprint":"f3","labels":{"alertname":"C"}}
	]}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(payload))
	newRouter(e, &counters{}, "").ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("code = %d, want 503 because one alert was refused", rr.Code)
	}
	var seen []string
	for _, a := range e.got {
		seen = append(seen, a.Fingerprint)
	}
	if len(seen) != 2 || seen[0] != "f1" || seen[1] != "f3" {
		t.Fatalf("enqueued %v, want [f1 f3] — f3 trails the refused f2", seen)
	}
}

func TestWebhookAcceptsWhenQueueHasRoom(t *testing.T) {
	e := &fakeEnqueuer{capacity: 2}
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
}

// End-to-end over the real dispatcher: the bug this guards against was not a
// wrong queue size but a return value nobody read, so the assertion has to run
// through the same wiring production uses — full queue, real refusal, HTTP
// status, and the counter that makes it visible.
func TestWebhookOverflowIsRefusedAndCounted(t *testing.T) {
	release := make(chan struct{})
	h := handlerFunc(func(_ context.Context, _ Alert) error {
		<-release // occupy the only worker for the whole test
		return nil
	})
	c := &counters{}
	d := newDispatcher(h, 1, 1, time.Second, c, silentLogger())
	// Deferred LIFO: the worker must be released before shutdown waits on it.
	defer d.shutdown(context.Background())
	defer close(release)
	router := newRouter(d, c, "")

	post := func(fingerprint string) int {
		body := fmt.Sprintf(`{"alerts":[{"status":"firing","fingerprint":%q,"labels":{"alertname":"Storm"}}]}`, fingerprint)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(body)))
		return rr.Code
	}

	if code := post("f1"); code != http.StatusAccepted { // picked up by the worker
		t.Fatalf("first post = %d, want 202", code)
	}
	time.Sleep(20 * time.Millisecond)
	if code := post("f2"); code != http.StatusAccepted { // fills the size-1 queue
		t.Fatalf("second post = %d, want 202", code)
	}
	if code := post("f3"); code != http.StatusServiceUnavailable {
		t.Fatalf("overflowing post = %d, want 503", code)
	}

	if got := c.alertsRejected.Load(); got != 1 {
		t.Fatalf("alertsRejected = %d, want 1", got)
	}
	// The refusal must also be legible in the exposition output, since that is
	// the only surface Prometheus reads.
	var buf strings.Builder
	c.writeTo(&buf)
	if !strings.Contains(buf.String(), "ai_sre_relay_alerts_rejected_total 1") {
		t.Fatalf("rejection counter missing from /metrics output:\n%s", buf.String())
	}
}
