package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
	"go.opentelemetry.io/otel/trace"
)

const tracerServiceName = "servicediscovery"

func noopShutdown(context.Context) error { return nil }

// traceResource builds the resource attached to every exported span.
//
// service.name has to be set explicitly. It is the key Tempo's service graph
// and Grafana's tracesToLogsV2 correlation both join on, yet nothing else here
// supplies it: WithTelemetrySDK contributes only telemetry.sdk.*, and
// sdktrace.WithResource merges against resource.Environment() rather than
// resource.Default(), so the usual unknown_service fallback never applies
// either. Spans would otherwise arrive in Tempo and be unusable.
//
// WithFromEnv is deliberately last: detectors merge in order with the later one
// winning, so OTEL_SERVICE_NAME and OTEL_RESOURCE_ATTRIBUTES still override the
// built-in name rather than being silently outranked by it.
func traceResource(ctx context.Context) (*resource.Resource, error) {
	return resource.New(ctx,
		resource.WithTelemetrySDK(),
		resource.WithAttributes(semconv.ServiceName(tracerServiceName)),
		resource.WithFromEnv(),
	)
}

// initTracing wires OTLP trace export and returns the provider's shutdown.
//
// Tracing stays off unless an endpoint is configured: the OTLP exporters
// otherwise default to localhost, so an uninstrumented environment — local runs
// and the test suite included — would retry against a collector that is not
// there. Absence of traces is covered by an alert on the collector side rather
// than by failing to boot here.
func initTracing(ctx context.Context, env envGetter) (func(context.Context) error, error) {
	endpoint := env.getEnvOrDefault("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	if endpoint == "" {
		slog.Info("Tracing disabled: OTEL_EXPORTER_OTLP_ENDPOINT is unset")
		return noopShutdown, nil
	}

	var (
		exporter *otlptrace.Exporter
		err      error
	)
	// Endpoint, headers and TLS all come from the standard OTEL_* variables that
	// the exporters read themselves, so protocol is the only real branch.
	protocol := env.getEnvOrDefault("OTEL_EXPORTER_OTLP_PROTOCOL", "grpc")
	switch protocol {
	case "grpc":
		exporter, err = otlptracegrpc.New(ctx)
	case "http/protobuf", "http":
		exporter, err = otlptracehttp.New(ctx)
	default:
		return noopShutdown, fmt.Errorf("unsupported OTEL_EXPORTER_OTLP_PROTOCOL %q", protocol)
	}
	if err != nil {
		return noopShutdown, fmt.Errorf("creating OTLP trace exporter: %w", err)
	}

	res, err := traceResource(ctx)
	if err != nil {
		return noopShutdown, fmt.Errorf("building trace resource: %w", err)
	}

	// No WithSampler on purpose. The SDK reads OTEL_TRACES_SAMPLER and
	// OTEL_TRACES_SAMPLER_ARG itself, and WithSampler would override them —
	// hardcoding a rate here would take the only dial-down knob away from
	// deployment config. Unset means ParentBased(AlwaysSample), i.e. every
	// trace, which is the intended pilot default on a single low-traffic
	// replica.
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(provider)
	// Without a propagator the service starts a fresh trace per request and any
	// caller-supplied context is dropped, which silently breaks correlation.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	slog.Info("Tracing enabled", "endpoint", endpoint, "protocol", protocol)
	return provider.Shutdown, nil
}

// traced wraps the whole middleware chain so a server span covers CORS and
// access logging too, not just handler execution.
func traced(next http.Handler) http.Handler {
	return otelhttp.NewHandler(next, tracerServiceName)
}

// tracedRoute names the active span after its ServeMux pattern.
//
// The span is started by the outer wrapper, before ServeMux has matched
// anything, so the route is not knowable at creation time — every span would
// otherwise share one opaque name. Naming per route also keeps the name set
// bounded: deriving it from the raw URL path would let unmatched requests mint
// unlimited distinct span names.
func tracedRoute(pattern string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		span := trace.SpanFromContext(r.Context())
		span.SetName(pattern)
		span.SetAttributes(attribute.String("http.route", pattern))
		handler(w, r)
	}
}
