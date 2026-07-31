package main

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
)

func resourceAttr(t *testing.T, key string) string {
	t.Helper()
	res, err := traceResource(context.Background())
	if err != nil {
		t.Fatalf("traceResource: %v", err)
	}
	for _, kv := range res.Attributes() {
		if string(kv.Key) == key {
			return kv.Value.AsString()
		}
	}
	return ""
}

// Tempo's service graph and Grafana's tracesToLogsV2 correlation both key on
// service.name. Neither WithTelemetrySDK nor WithFromEnv supplies it, and
// WithResource merges against resource.Environment() rather than
// resource.Default(), so without an explicit attribute every span lands
// unattributed.
func TestTraceResourceSetsServiceName(t *testing.T) {
	if got := resourceAttr(t, "service.name"); got != tracerServiceName {
		t.Fatalf("service.name = %q, want %q", got, tracerServiceName)
	}
}

// The built-in name is a default, not an override: detectors merge in order and
// the later one wins, so WithFromEnv must come last.
func TestTraceResourceEnvOverridesServiceName(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "servicediscovery-canary")
	if got := resourceAttr(t, "service.name"); got != "servicediscovery-canary" {
		t.Fatalf("service.name = %q, want the env value to win", got)
	}
}

func TestTraceResourceKeepsTelemetrySDK(t *testing.T) {
	if got := resourceAttr(t, "telemetry.sdk.language"); got != "go" {
		t.Fatalf("telemetry.sdk.language = %q, want %q", got, "go")
	}
}

// Sampling must stay configurable through the spec's OTEL_TRACES_SAMPLER, which
// the SDK reads itself. A hardcoded sdktrace.WithSampler would silently
// override it, so this pins the decision not to add one.
func TestInitTracingHonoursEnvSampler(t *testing.T) {
	prev := otel.GetTracerProvider()
	t.Cleanup(func() { otel.SetTracerProvider(prev) })

	t.Setenv("OTEL_TRACES_SAMPLER", "always_off")
	env := mockEnvGetter{envs: map[string]string{"OTEL_EXPORTER_OTLP_ENDPOINT": "http://127.0.0.1:4317"}}

	shutdown, err := initTracing(context.Background(), env)
	if err != nil {
		t.Fatalf("initTracing: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	t.Cleanup(func() { _ = shutdown(ctx) })

	_, span := otel.GetTracerProvider().Tracer("test").Start(context.Background(), "probe")
	defer span.End()
	if span.IsRecording() {
		t.Fatal("OTEL_TRACES_SAMPLER=always_off was ignored; a hardcoded WithSampler would cause this")
	}
}
