package main

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strconv"
)

// maxWebhookBytes caps the request body. Alertmanager batches are small; the
// cap guards the memory-capped replica against an oversized/malicious payload.
const maxWebhookBytes = 1 << 20 // 1 MiB

// retryAfterSeconds advertises roughly one worker-slot turnover, so a sender
// that honours the hint returns when capacity plausibly exists rather than
// hammering a queue that frees a slot only every couple of minutes. It is
// advisory only: Alertmanager ignores the header entirely and applies its own
// exponential backoff, so nothing here may depend on it being read. It still
// earns its place for the plain-curl caller, whose retry flag does honour it.
const retryAfterSeconds = 120

// enqueuer is the seam to the worker pool; the HTTP layer depends only on this.
type enqueuer interface {
	enqueue(a Alert) bool
}

func newRouter(e enqueuer, c *counters, webhookToken string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		c.writeTo(w)
	})
	mux.HandleFunc("POST /webhook", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxWebhookBytes)
		var payload struct {
			Alerts []Alert `json:"alerts"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "bad payload", http.StatusBadRequest)
			return
		}
		// Answer fast so the sender is not held open for the full
		// investigation; the worker pool does the slow work asynchronously.
		// The rest of the batch is still offered after a rejection — capacity
		// freed mid-loop should be used rather than wasted.
		accepted := true
		for _, a := range payload.Alerts {
			if a.Status == "firing" && !e.enqueue(a) {
				accepted = false
			}
		}
		// A 2xx tells the sender the batch is handled and retires its only
		// copy; answering 503 is what converts a rejection into a redelivery.
		// It must be 5xx specifically — Alertmanager's webhook receiver retries
		// 5xx but treats 429 as a permanent failure, so the intuitive
		// "too many requests" reply would discard the alert outright.
		// Re-sending the whole batch is safe because both dedup layers treat a
		// redelivered alert as a repeat: fingerprints still queued or
		// investigating are coalesced, and completed ones are matched to their
		// open ticket.
		if !accepted {
			w.Header().Set("Retry-After", strconv.Itoa(retryAfterSeconds))
			http.Error(w, "investigation queue full", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	})
	return withAuth(mux, webhookToken)
}

// withAuth enforces a bearer token on the expensive/side-effecting routes when
// a token is configured, leaving /healthz open for kubelet probes and /metrics
// open for Prometheus. The Service is ClusterIP, but the downstream (paid LLM
// calls, Jira issues, GitHub PRs) warrants defense-in-depth against any
// in-cluster caller. When no token is set the mux is returned unwrapped
// (startup logs a warning).
func withAuth(next http.Handler, token string) http.Handler {
	if token == "" {
		return next
	}
	want := []byte("Bearer " + token)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" || r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}
		// Constant-time compare to avoid leaking the token via response timing.
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), want) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
