package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestCorrelationFieldsUseCanonicalNames(t *testing.T) {
	ctx := context.Background()
	ctx = WithRequestID(ctx, "req-1")
	ctx = WithTenantID(ctx, "tenant-1")
	ctx = WithWorkflowID(ctx, "wf-1")
	ctx = trace.ContextWithSpanContext(ctx, trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{1},
		SpanID:     trace.SpanID{2},
		TraceFlags: trace.FlagsSampled,
	}))

	fields := CorrelationFields(ctx)
	got := map[string]bool{}
	for _, field := range fields {
		got[field.Key] = true
	}

	for _, key := range []string{"request_id", "tenant_id", "workflow_id", "trace_id", "span_id"} {
		if !got[key] {
			t.Fatalf("expected field %q in correlation fields", key)
		}
	}
}

func TestLogRequestCompleteJSON(t *testing.T) {
	var buf bytes.Buffer
	logger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(zapcore.EncoderConfig{
			TimeKey:     "time",
			LevelKey:    "level",
			MessageKey:  "msg",
			EncodeTime:  zapcore.ISO8601TimeEncoder,
			EncodeLevel: zapcore.LowercaseLevelEncoder,
		}),
		zapcore.AddSync(&buf),
		zapcore.InfoLevel,
	))

	ctx := WithRequestID(context.Background(), "req-1")
	LogRequestComplete(logger, ctx, RequestComplete{
		Service:    "control",
		Method:     http.MethodPost,
		Route:      "/internal/mcp/config",
		Status:     http.StatusForbidden,
		Duration:   42 * time.Millisecond,
		ErrorCode:  "active_profile_missing",
		Dependency: "control_db",
	})

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal log: %v", err)
	}

	if got["msg"] != "control request completed" {
		t.Fatalf("unexpected message: %v", got["msg"])
	}
	for _, key := range []string{"request_id", "method", "route", "status", "duration_ms", "error_code", "dependency"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("expected key %q in log", key)
		}
	}
}

func TestDependencyFailureMessageIncludesActivity(t *testing.T) {
	msg := dependencyFailureMessage(DependencyFailure{
		Dependency: "vault",
		Operation:  "read",
		Activity:   "resolving MCP source spec values",
	})
	want := "vault read failed while resolving MCP source spec values"
	if msg != want {
		t.Fatalf("got %q want %q", msg, want)
	}
}

func TestSanitizeURL(t *testing.T) {
	host, path := SanitizeURL("https://example.com/api/v1/config?token=secret")
	if host != "example.com" {
		t.Fatalf("unexpected host %q", host)
	}
	if path != "/api/v1/config" {
		t.Fatalf("unexpected path %q", path)
	}
}

func TestObserveHTTPDependencySanitizesURL(t *testing.T) {
	rawURL, err := url.Parse("https://example.com/api/v1/config?token=secret")
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	req := &http.Request{Method: http.MethodGet, URL: rawURL}
	event := ObserveHTTPDependency("jwks", "fetch", "verifying the bearer token", req, http.StatusBadGateway, 10*time.Millisecond, nil)

	if event.URLHost != "example.com" || event.URLPath != "/api/v1/config" {
		t.Fatalf("unexpected sanitized URL: host=%q path=%q", event.URLHost, event.URLPath)
	}
}

func TestNewTextLoggerIncludesServiceField(t *testing.T) {
	logger, err := New(Config{
		Service: "control-service",
		Format:  FormatText,
		Level:   "info",
	})
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}
	defer logger.Sync()

	if logger == nil {
		t.Fatal("expected logger")
	}
}

func TestRedactedValue(t *testing.T) {
	if got := RedactedValue(); !strings.Contains(got, "REDACTED") {
		t.Fatalf("unexpected redacted marker %q", got)
	}
}
