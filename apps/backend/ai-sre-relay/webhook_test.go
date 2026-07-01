package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type stubHandler struct{ calls int }

func (s *stubHandler) Handle(_ context.Context, _ Alert) error { s.calls++; return nil }

func TestHealthz(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	newRouter(&stubHandler{}).ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("healthz = %d, want 200", rr.Code)
	}
}
