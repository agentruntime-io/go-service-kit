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

func TestPrettyTextLoggerFormatsFieldsAndMessage(t *testing.T) {
	var buf bytes.Buffer
	logger, err := New(Config{
		Service: "control-service",
		Format:  FormatPrettyText,
		Level:   "info",
		Output:  &buf,
	})
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}

	logger.Info("schema status checker batch started",
		TraceID("trace-1"),
		SpanID("span-1"),
		Bool("trace_sampled", true),
		Int("server_count", 4),
	)

	out := buf.String()
	for _, want := range []string{
		"INFO",
		"generic",
		"schema status checker batch started",
		"service=control-service",
		"trace_id=trace-1",
		"span_id=span-1",
		"trace_sampled=true",
		"server_count=4",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("prettytext log missing %s in %s", want, out)
		}
	}
	if strings.Contains(out, "{") {
		t.Fatalf("prettytext should not render trailing JSON object: %s", out)
	}
	if strings.Contains(out, "msg=") {
		t.Fatalf("prettytext generic renderer should not use msg=: %s", out)
	}
}

func TestPrettyTextLoggerPrintsSQLOnSeparateLine(t *testing.T) {
	var buf bytes.Buffer
	logger, err := New(Config{
		Service: "control-service",
		Format:  FormatPrettyText,
		Level:   "warn",
		Output:  &buf,
	})
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}

	logger.Warn("db query slow",
		Kind("db"),
		DurationMSFloat(250*time.Millisecond),
		Int64("rows", 3),
		Int("slow_threshold_ms", 200),
		TraceID("trace-1"),
		SpanID("span-1"),
		String("sql", "SELECT * FROM workflows"),
	)

	out := buf.String()
	for _, want := range []string{
		"WARN",
		"db",
		"slow query:",
		"250ms",
		"rows=3",
		"threshold=200",
		"trace_id=trace-1",
		"span_id=span-1",
		"SELECT *",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("prettytext db log missing %s in %s", want, out)
		}
	}
	if strings.Contains(out, "msg=") {
		t.Fatalf("db prettytext should not use msg=: %s", out)
	}
	if !strings.Contains(out, "\n  SELECT *") {
		t.Fatalf("expected multiline SQL rendering in prettytext log: %s", out)
	}
}

func TestPrettyTextRequestRendererUsesRequestLayout(t *testing.T) {
	var buf bytes.Buffer
	logger, err := New(Config{
		Service: "control-service",
		Format:  FormatPrettyText,
		Level:   "info",
		Output:  &buf,
	})
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}

	ctx := WithRequestID(context.Background(), "req-1")
	LogRequestComplete(logger, ctx, RequestComplete{
		Service:      "control",
		Method:       http.MethodPost,
		Route:        "/internal/mcp/config",
		Status:       http.StatusForbidden,
		Duration:     42 * time.Millisecond,
		ErrorCode:    "active_profile_missing",
		ErrorMessage: "instance has no active config profile",
		Fields: []any{
			TenantID("tenant-1"),
			InstanceID("inst-1"),
		},
	})

	out := buf.String()
	for _, want := range []string{
		"request",
		"POST /internal/mcp/config 403 42ms",
		"request_id=req-1",
		"tenant_id=tenant-1",
		"instance_id=inst-1",
		"error_code=active_profile_missing",
		"error_message=\"instance has no active config profile\"",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("prettytext request log missing %s in %s", want, out)
		}
	}
	if strings.Contains(out, "control request completed") {
		t.Fatalf("request prettytext should not surface the generic completion message: %s", out)
	}
}

func TestPrettyTextDependencyRendererUsesDependencyLayout(t *testing.T) {
	var buf bytes.Buffer
	logger, err := New(Config{
		Service: "control-service",
		Format:  FormatPrettyText,
		Level:   "info",
		Output:  &buf,
	})
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}

	LogDependencyFailure(logger, context.Background(), DependencyFailure{
		Message:    "vault read failed while resolving MCP source spec values",
		Dependency: "vault",
		Operation:  "source_spec_read",
		URLHost:    "vault.dev.agentruntime.io",
		URLPath:    "/v1/secret",
		Status:     http.StatusBadGateway,
		Duration:   132 * time.Millisecond,
		ErrorCode:  "vault_read_failed",
	})

	out := buf.String()
	for _, want := range []string{
		"dependency",
		"vault read failed while resolving MCP source spec values",
		"dependency=vault",
		"operation=source_spec_read",
		"status=502",
		"132ms",
		"url_host=vault.dev.agentruntime.io",
		"url_path=/v1/secret",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("prettytext dependency log missing %s in %s", want, out)
		}
	}
}

func TestRedactedValue(t *testing.T) {
	if got := RedactedValue(); !strings.Contains(got, "REDACTED") {
		t.Fatalf("unexpected redacted marker %q", got)
	}
}
