package main

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
)

// maxWebhookBytes caps the request body. Alertmanager batches are small; the
// cap guards the memory-capped replica against an oversized/malicious payload.
const maxWebhookBytes = 1 << 20 // 1 MiB

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
		// Return 202 fast so Alertmanager is not held open for the full
		// investigation; the worker pool does the slow work asynchronously.
		for _, a := range payload.Alerts {
			if a.Status == "firing" {
				e.enqueue(a)
			}
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
