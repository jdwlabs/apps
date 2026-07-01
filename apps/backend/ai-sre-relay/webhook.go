package main

import (
	"context"
	"encoding/json"
	"net/http"
)

// handler is the pipeline seam; webhook depends only on this narrow interface.
type handler interface {
	Handle(ctx context.Context, a Alert) error
}

func newRouter(h handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("POST /webhook", func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Alerts []Alert `json:"alerts"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "bad payload", http.StatusBadRequest)
			return
		}
		for _, a := range payload.Alerts {
			if a.Status != "firing" {
				continue
			}
			a := a
			// Async: return 202 fast so Alertmanager is not held open for the
			// full investigation (which can take tens of seconds).
			go func() {
				_ = h.Handle(context.Background(), a)
			}()
		}
		w.WriteHeader(http.StatusAccepted)
	})
	return mux
}
