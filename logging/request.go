package logging

import (
	"context"
	"time"

	"go.uber.org/zap"
)

type RequestComplete struct {
	Service string
	Method  string
	Route   string
	Status  int

	Duration     time.Duration
	ErrorCode    string
	ErrorMessage string
	Dependency   string

	Fields []zap.Field
}

func LogRequestComplete(logger *zap.Logger, ctx context.Context, event RequestComplete) {
	if logger == nil {
		return
	}

	fields := make([]zap.Field, 0, 10+len(event.Fields))
	fields = append(fields, CorrelationFields(ctx)...)
	if event.Method != "" {
		fields = append(fields, Method(event.Method))
	}
	if event.Route != "" {
		fields = append(fields, Route(event.Route))
	}
	if event.Status > 0 {
		fields = append(fields, Status(event.Status))
	}
	if event.Duration > 0 {
		fields = append(fields, DurationMS(event.Duration))
	}
	if event.ErrorCode != "" {
		fields = append(fields, ErrorCode(event.ErrorCode))
	}
	if event.ErrorMessage != "" {
		fields = append(fields, ErrorMessage(event.ErrorMessage))
	}
	if event.Dependency != "" {
		fields = append(fields, Dependency(event.Dependency))
	}
	fields = append(fields, event.Fields...)

	logger.Info(requestCompletedMessage(event.Service), fields...)
}

func requestCompletedMessage(service string) string {
	if service == "" {
		return "request completed"
	}
	return service + " request completed"
}
