// Package telemetry provides reusable helpers for creating and observing
// outgoing service-call spans that follow the Qlub distributed tracing
// conventions.
//
// Every HTTP client that calls another internal service should use
// StartClientSpan + ObserveHTTPOutcome instead of rolling its own span logic.
// The resulting trace shape in Jaeger is:
//
//	<incoming request span>               (otelchi middleware)
//	  └── <Service>/<operation>           (StartClientSpan — SpanKindClient)
//	        ├── peer.service = "..."
//	        ├── http.status_code = 200
//	        ├── log: "<service> <op> ok"  (span event via svcLogging.*Context)
//	        └── HTTP <METHOD> <path>      (otelhttp.NewTransport)
package telemetry

import (
	"context"
	"fmt"
	"time"

	svcLogging "github.com/agentruntime-io/go-service-kit/logging"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// StartClientSpan starts a SpanKindClient span for one outgoing service call.
//
//   - tracerName should be the full Go package import path of the calling
//     package (e.g. "github.com/agentruntime-io/work-service/internal/services/wheelhouseclient").
//     This sets the instrumentation scope visible in trace backends.
//   - service is the downstream service name (e.g. "wheelhouse").
//   - operation is the short call name (e.g. "billing_status").
//     The span will be named "<service>/<operation>".
//   - method and path are the HTTP method and route template.
//   - extraAttrs are optional additional span attributes to set at start time
//     (e.g. business-level IDs that are known before the call).
//
// The caller must end the span — call defer span.End() immediately after
// calling StartClientSpan.
//
// Pass the returned ctx to http.NewRequestWithContext so that
// otelhttp.NewTransport creates the HTTP-transport child span underneath
// this business span.
func StartClientSpan(
	ctx context.Context,
	tracerName, service, operation, method, path string,
	extraAttrs ...attribute.KeyValue,
) (context.Context, trace.Span) {
	attrs := make([]attribute.KeyValue, 0, 3+len(extraAttrs))
	attrs = append(attrs,
		attribute.String("peer.service", service),
		attribute.String("http.method", method),
		attribute.String("http.route", path),
	)
	attrs = append(attrs, extraAttrs...)

	return otel.Tracer(tracerName).Start(ctx, service+"/"+operation,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attrs...),
	)
}

// ObserveHTTPOutcome records the result of an outgoing HTTP call on the span
// and emits a structured log via svcLogging.*Context.
//
// Because svcLogging.*Context mirrors every log message as a named event on
// the active span, calling ObserveHTTPOutcome also produces the "log" entries
// visible in Jaeger's trace UI — no additional span.AddEvent call is needed.
//
// Log levels follow this convention:
//   - transport error (err != nil) → Error
//   - HTTP 5xx                     → Error
//   - HTTP 4xx                     → Warn
//   - HTTP 2xx / 3xx               → Info
func ObserveHTTPOutcome(
	ctx context.Context,
	span trace.Span,
	service, operation string,
	status int,
	dur time.Duration,
	err error,
) {
	span.SetAttributes(
		attribute.Int("http.status_code", status),
		attribute.Int64("duration_ms", dur.Milliseconds()),
	)

	switch {
	case err != nil:
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		svcLogging.ErrorContext(ctx, service+" "+operation+" failed",
			"dependency", service,
			"operation", operation,
			"error", err,
			"duration_ms", dur.Milliseconds(),
		)

	case status >= 500:
		span.SetStatus(codes.Error, fmt.Sprintf("upstream %d", status))
		svcLogging.ErrorContext(ctx, service+" "+operation+" server error",
			"dependency", service,
			"operation", operation,
			"status", status,
			"duration_ms", dur.Milliseconds(),
		)

	case status >= 400:
		span.SetStatus(codes.Error, fmt.Sprintf("upstream %d", status))
		svcLogging.WarnContext(ctx, service+" "+operation+" client error",
			"dependency", service,
			"operation", operation,
			"status", status,
			"duration_ms", dur.Milliseconds(),
		)

	default:
		svcLogging.InfoContext(ctx, service+" "+operation+" ok",
			"dependency", service,
			"operation", operation,
			"status", status,
			"duration_ms", dur.Milliseconds(),
		)
	}
}
