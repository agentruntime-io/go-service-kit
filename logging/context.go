package logging

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type contextKey string

const (
	requestIDKey  contextKey = "request_id"
	tenantIDKey   contextKey = "tenant_id"
	projectIDKey  contextKey = "project_id"
	userIDKey     contextKey = "user_id"
	workflowIDKey contextKey = "workflow_id"
	runIDKey      contextKey = "run_id"
	instanceIDKey contextKey = "instance_id"
	serverIDKey   contextKey = "server_id"
)

type ContextFieldExtractor func(context.Context) []zap.Field

var (
	contextFieldExtractorsMu sync.RWMutex
	contextFieldExtractors   []ContextFieldExtractor
)

func WithRequestID(ctx context.Context, value string) context.Context {
	return context.WithValue(ctx, requestIDKey, value)
}

func WithTenantID(ctx context.Context, value string) context.Context {
	return context.WithValue(ctx, tenantIDKey, value)
}

func WithProjectID(ctx context.Context, value string) context.Context {
	return context.WithValue(ctx, projectIDKey, value)
}

func WithUserID(ctx context.Context, value string) context.Context {
	return context.WithValue(ctx, userIDKey, value)
}

func WithWorkflowID(ctx context.Context, value string) context.Context {
	return context.WithValue(ctx, workflowIDKey, value)
}

func WithRunID(ctx context.Context, value string) context.Context {
	return context.WithValue(ctx, runIDKey, value)
}

func WithInstanceID(ctx context.Context, value string) context.Context {
	return context.WithValue(ctx, instanceIDKey, value)
}

func WithServerID(ctx context.Context, value string) context.Context {
	return context.WithValue(ctx, serverIDKey, value)
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
	if value, ok := valueFromContext(ctx, requestIDKey); ok {
		fields = append(fields, RequestID(value))
	}
	if value, ok := valueFromContext(ctx, tenantIDKey); ok {
		fields = append(fields, TenantID(value))
	}
	if value, ok := valueFromContext(ctx, projectIDKey); ok {
		fields = append(fields, ProjectID(value))
	}
	if value, ok := valueFromContext(ctx, userIDKey); ok {
		fields = append(fields, UserID(value))
	}
	if value, ok := valueFromContext(ctx, workflowIDKey); ok {
		fields = append(fields, WorkflowID(value))
	}
	if value, ok := valueFromContext(ctx, runIDKey); ok {
		fields = append(fields, RunID(value))
	}
	if value, ok := valueFromContext(ctx, instanceIDKey); ok {
		fields = append(fields, InstanceID(value))
	}
	if value, ok := valueFromContext(ctx, serverIDKey); ok {
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

func valueFromContext(ctx context.Context, key contextKey) (string, bool) {
	value, ok := ctx.Value(key).(string)
	if !ok || value == "" {
		return "", false
	}
	return value, true
}
