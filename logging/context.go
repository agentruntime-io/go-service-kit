package logging

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// ContextKey is the type used for all logging context keys.  It is exported so
// that external middleware (e.g. auth, request-ID) can store values directly
// with the same keys that CorrelationFields reads — no registered extractor or
// dual-write required.
type ContextKey string

const (
	RequestIDKey  ContextKey = "request_id"
	TenantIDKey   ContextKey = "tenant_id"
	ProjectIDKey  ContextKey = "project_id"
	UserIDKey     ContextKey = "user_id"
	WorkflowIDKey ContextKey = "workflow_id"
	RunIDKey      ContextKey = "run_id"
	InstanceIDKey ContextKey = "instance_id"
	ServerIDKey   ContextKey = "server_id"
)

// ContextFieldExtractor is a function that extracts additional zap fields from
// a context.  Register custom extractors via RegisterContextFieldExtractor.
type ContextFieldExtractor func(context.Context) []zap.Field

var (
	contextFieldExtractorsMu sync.RWMutex
	contextFieldExtractors   []ContextFieldExtractor
)

func WithRequestID(ctx context.Context, value string) context.Context {
	return context.WithValue(ctx, RequestIDKey, value)
}

func WithTenantID(ctx context.Context, value string) context.Context {
	return context.WithValue(ctx, TenantIDKey, value)
}

func WithProjectID(ctx context.Context, value string) context.Context {
	return context.WithValue(ctx, ProjectIDKey, value)
}

func WithUserID(ctx context.Context, value string) context.Context {
	return context.WithValue(ctx, UserIDKey, value)
}

func WithWorkflowID(ctx context.Context, value string) context.Context {
	return context.WithValue(ctx, WorkflowIDKey, value)
}

func WithRunID(ctx context.Context, value string) context.Context {
	return context.WithValue(ctx, RunIDKey, value)
}

func WithInstanceID(ctx context.Context, value string) context.Context {
	return context.WithValue(ctx, InstanceIDKey, value)
}

func WithServerID(ctx context.Context, value string) context.Context {
	return context.WithValue(ctx, ServerIDKey, value)
}

func RegisterContextFieldExtractor(extractor ContextFieldExtractor) {
	if extractor == nil {
		return
	}
	contextFieldExtractorsMu.Lock()
	defer contextFieldExtractorsMu.Unlock()
	contextFieldExtractors = append(contextFieldExtractors, extractor)
}

func CorrelationFields(ctx context.Context) []zap.Field {
	if ctx == nil {
		return nil
	}

	fields := make([]zap.Field, 0, 10)
	if value, ok := valueFromContext(ctx, RequestIDKey); ok {
		fields = append(fields, RequestID(value))
	}
	if value, ok := valueFromContext(ctx, TenantIDKey); ok {
		fields = append(fields, TenantID(value))
	}
	if value, ok := valueFromContext(ctx, ProjectIDKey); ok {
		fields = append(fields, ProjectID(value))
	}
	if value, ok := valueFromContext(ctx, UserIDKey); ok {
		fields = append(fields, UserID(value))
	}
	if value, ok := valueFromContext(ctx, WorkflowIDKey); ok {
		fields = append(fields, WorkflowID(value))
	}
	if value, ok := valueFromContext(ctx, RunIDKey); ok {
		fields = append(fields, RunID(value))
	}
	if value, ok := valueFromContext(ctx, InstanceIDKey); ok {
		fields = append(fields, InstanceID(value))
	}
	if value, ok := valueFromContext(ctx, ServerIDKey); ok {
		fields = append(fields, ServerID(value))
	}

	spanContext := trace.SpanContextFromContext(ctx)
	if spanContext.IsValid() {
		fields = append(fields,
			TraceID(spanContext.TraceID().String()),
			SpanID(spanContext.SpanID().String()),
			TraceSampled(spanContext.IsSampled()),
		)
	}

	contextFieldExtractorsMu.RLock()
	extractors := append([]ContextFieldExtractor(nil), contextFieldExtractors...)
	contextFieldExtractorsMu.RUnlock()
	for _, extractor := range extractors {
		fields = append(fields, extractor(ctx)...)
	}

	return dedupeFields(fields)
}

func valueFromContext(ctx context.Context, key ContextKey) (string, bool) {
	value, ok := ctx.Value(key).(string)
	if !ok || value == "" {
		return "", false
	}
	return value, true
}
