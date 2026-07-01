package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type stubHandler struct{ calls int }

func (s *stubHandler) Handle(_ context.Context, _ Alert) error { s.calls++; return nil }

type handlerFunc func(context.Context, Alert) error

func (f handlerFunc) Handle(ctx context.Context, a Alert) error { return f(ctx, a) }

func TestHealthz(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	newRouter(&stubHandler{}).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("healthz = %d, want 200", rr.Code)
	}
	if got := rr.Body.String(); got != "ok" {
		t.Errorf("healthz body = %q, want \"ok\"", got)
	}
}

func TestWebhookRecoversFromPanic(t *testing.T) {
	started := make(chan struct{})
	h := handlerFunc(func(_ context.Context, a Alert) error {
		close(started) // signal goroutine is running before we panic
		panic("induced panic for " + a.Fingerprint)
	})

	const payload = `{"alerts":[{"status":"firing","fingerprint":"fp-panic","labels":{"alertname":"Crasher"}}]}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(payload))
	newRouter(h).ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("code = %d, want 202", rr.Code)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("goroutine did not start within 1s")
	}
	// Wait for the deferred recover to complete after the panic.
	time.Sleep(50 * time.Millisecond)
}

func TestWebhookDispatchesFiringAlerts(t *testing.T) {
	done := make(chan Alert, 2)
	h := handlerFunc(func(_ context.Context, a Alert) error { done <- a; return nil })

	const payload = `{"alerts":[
	  {"status":"firing","fingerprint":"f1","labels":{"alertname":"A"}},
	  {"status":"resolved","fingerprint":"f2","labels":{"alertname":"B"}}
	]}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(payload))
	newRouter(h).ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("code = %d, want 202", rr.Code)
	}
	select {
	case got := <-done:
		if got.Fingerprint != "f1" {
			t.Fatalf("dispatched wrong alert: %s", got.Fingerprint)
		}
	case <-time.After(time.Second):
		t.Fatal("firing alert not dispatched")
	}
	select {
	case got := <-done:
		t.Fatalf("resolved alert should not dispatch: %s", got.Fingerprint)
	case <-time.After(100 * time.Millisecond):
	}
}
