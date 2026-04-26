package logging

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type PhaseLog struct {
	Message string
	Phase   string
	Level   Level
	Fields  []any
}

func LogPhase(logger *zap.Logger, ctx context.Context, event PhaseLog) {
	if logger == nil {
		return
	}

	fields := make([]zap.Field, 0, 8+len(event.Fields))
	fields = append(fields, CorrelationFields(ctx)...)
	if event.Phase != "" {
		fields = append(fields, Phase(event.Phase))
	}
	fields = append(fields, AttrsToFields(event.Fields...)...)

	level := event.Level
	if event.Level == 0 {
		level = LevelInfo
	}
	writeFields(logger, level, event.Message, dedupeFields(fields)...)

	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		attrs := []attribute.KeyValue{}
		if event.Phase != "" {
			attrs = append(attrs, attribute.String("phase", event.Phase))
		}
		span.AddEvent(event.Message, trace.WithAttributes(attrs...))
	}
}
