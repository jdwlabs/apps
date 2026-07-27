package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestInitTracingDisabledWithoutEndpoint(t *testing.T) {
	env := mockEnvGetter{envs: map[string]string{}}

	shutdown, err := initTracing(context.Background(), env)
	if err != nil {
		t.Fatalf("initTracing returned an error with no endpoint configured: %v", err)
	}
	if shutdown == nil {
		t.Fatal("initTracing returned a nil shutdown func")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Errorf("shutdown returned an error when tracing was disabled: %v", err)
	}
}

func TestInitTracingRejectsUnknownProtocol(t *testing.T) {
	env := mockEnvGetter{envs: map[string]string{
		"OTEL_EXPORTER_OTLP_ENDPOINT": "http://tempo:4317",
		"OTEL_EXPORTER_OTLP_PROTOCOL": "carrier-pigeon",
	}}

	shutdown, err := initTracing(context.Background(), env)
	if err == nil {
		t.Fatal("initTracing accepted an unsupported protocol")
	}
	// A configuration error must still hand back a usable shutdown, because the
	// caller defers it before inspecting the error.
	if shutdown == nil {
		t.Fatal("initTracing returned a nil shutdown alongside its error")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Errorf("fallback shutdown returned an error: %v", err)
	}
}

// The outer wrapper starts the span before ServeMux has matched a route, so
// without tracedRoute every request shares one opaque span name.
func TestTracedRouteNamesSpanAfterPattern(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })

	ctx, span := provider.Tracer("test").Start(context.Background(), "placeholder")

	called := false
	handler := tracedRoute("GET /api/remotes", func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/remotes", nil).WithContext(ctx)
	handler(httptest.NewRecorder(), req)
	span.End()

	if !called {
		t.Fatal("tracedRoute did not invoke the wrapped handler")
	}

	ended := recorder.Ended()
	if len(ended) != 1 {
		t.Fatalf("expected exactly 1 recorded span, got %d", len(ended))
	}
	if got := ended[0].Name(); got != "GET /api/remotes" {
		t.Errorf("span name = %q, want %q", got, "GET /api/remotes")
	}

	var route string
	for _, attr := range ended[0].Attributes() {
		if attr.Key == "http.route" {
			route = attr.Value.AsString()
		}
	}
	if route != "GET /api/remotes" {
		t.Errorf("http.route attribute = %q, want %q", route, "GET /api/remotes")
	}
}

// tracedRoute must not require an active span: with tracing disabled the
// context carries a non-recording span, and setting a name on it is a no-op
// rather than a panic.
func TestTracedRouteWithoutActiveSpan(t *testing.T) {
	handler := tracedRoute("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	w := httptest.NewRecorder()
	handler(w, httptest.NewRequest(http.MethodGet, "/health", nil))

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

// The tracing wrapper sits outside CORS and access logging, so a regression
// there would take the whole handler chain down rather than just telemetry.
func TestTracedServerStillServesRoutes(t *testing.T) {
	env := mockEnvGetter{envs: map[string]string{"SD_PORT": "9090"}}
	server := initServer(env)

	for _, path := range []string{"/", "/health", "/api/remotes", "/api/micro-frontends"} {
		w := httptest.NewRecorder()
		server.Handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))

		if w.Code != http.StatusOK {
			t.Errorf("GET %s through traced handler = %d, want %d", path, w.Code, http.StatusOK)
		}
	}
}
