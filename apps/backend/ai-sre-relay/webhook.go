package main

import (
	"context"
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
	return mux
}
