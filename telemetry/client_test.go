package telemetry_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/agentruntime-io/go-service-kit/telemetry"
)

// setupTracer registers a SpanRecorder-backed TracerProvider as the global
// OTel provider and restores the no-op provider after the test.
func setupTracer(t *testing.T) *tracetest.SpanRecorder {
	t.Helper()
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(tp)
	t.Cleanup(func() {
		otel.SetTracerProvider(sdktrace.NewTracerProvider())
		_ = tp.Shutdown(context.Background())
	})
	return sr
}

func TestStartClientSpan_SpanName(t *testing.T) {
	sr := setupTracer(t)

	_, span := telemetry.StartClientSpan(
		context.Background(),
		"test/tracer", "wheelhouse", "billing_status",
		http.MethodGet, "/internal/billing/status/t1",
	)
	span.End()

	ended := sr.Ended()
	if len(ended) == 0 {
		t.Fatal("no spans recorded")
	}
	if got := ended[0].Name(); got != "wheelhouse/billing_status" {
		t.Errorf("span name = %q, want %q", got, "wheelhouse/billing_status")
	}
}

func TestStartClientSpan_ExtraAttrs(t *testing.T) {
	sr := setupTracer(t)

	_, span := telemetry.StartClientSpan(
		context.Background(),
		"test/tracer", "wheelhouse", "credit_check",
		http.MethodPost, "/internal/billing/credit-check",
		attribute.String("billing.tenant_id", "tenant-1"),
	)
	span.End()

	for _, a := range sr.Ended()[0].Attributes() {
		if string(a.Key) == "billing.tenant_id" && a.Value.AsString() == "tenant-1" {
			return
		}
	}
	t.Error("span missing attribute billing.tenant_id")
}

func TestObserveHTTPOutcome_TransportError(t *testing.T) {
	sr := setupTracer(t)

	ctx, span := telemetry.StartClientSpan(
		context.Background(),
		"test/tracer", "wheelhouse", "billing_status",
		http.MethodGet, "/internal/billing/status/t1",
	)
	telemetry.ObserveHTTPOutcome(ctx, span, "wheelhouse", "billing_status", 0, 10*time.Millisecond, errors.New("dial timeout"))
	span.End()

	if s := sr.Ended()[0]; s.Status().Code != codes.Error {
		t.Errorf("span status = %v, want Error", s.Status().Code)
	}
}

func TestObserveHTTPOutcome_5xx(t *testing.T) {
	sr := setupTracer(t)

	ctx, span := telemetry.StartClientSpan(
		context.Background(),
		"test/tracer", "wheelhouse", "billing_status",
		http.MethodGet, "/internal/billing/status/t1",
	)
	telemetry.ObserveHTTPOutcome(ctx, span, "wheelhouse", "billing_status", http.StatusInternalServerError, 5*time.Millisecond, nil)
	span.End()

	if s := sr.Ended()[0]; s.Status().Code != codes.Error {
		t.Errorf("span status = %v, want Error for 500", s.Status().Code)
	}
}

func TestObserveHTTPOutcome_200_NoError(t *testing.T) {
	sr := setupTracer(t)

	ctx, span := telemetry.StartClientSpan(
		context.Background(),
		"test/tracer", "wheelhouse", "billing_status",
		http.MethodGet, "/internal/billing/status/t1",
	)
	telemetry.ObserveHTTPOutcome(ctx, span, "wheelhouse", "billing_status", http.StatusOK, 5*time.Millisecond, nil)
	span.End()

	if s := sr.Ended()[0]; s.Status().Code == codes.Error {
		t.Errorf("span status = Error, want not error for 200")
	}
}
