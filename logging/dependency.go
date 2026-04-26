package logging

import (
	"context"
	"net/http"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type DependencyFailure struct {
	Message    string
	Dependency string
	Operation  string
	Activity   string
	Method     string
	URLHost    string
	URLPath    string
	Status     int
	Duration   time.Duration
	ErrorCode  string
	Err        error
	Fields     []any
}

func LogDependencyFailure(logger *zap.Logger, ctx context.Context, event DependencyFailure) {
	if logger == nil {
		return
	}

	fields := make([]zap.Field, 0, 12+len(event.Fields))
	fields = append(fields, CorrelationFields(ctx)...)
	fields = append(fields, Kind("dependency"))
	if event.Dependency != "" {
		fields = append(fields, Dependency(event.Dependency))
	}
	if event.Operation != "" {
		fields = append(fields, Operation(event.Operation))
	}
	if event.Method != "" {
		fields = append(fields, Method(event.Method))
	}
	if event.URLHost != "" {
		fields = append(fields, URLHost(event.URLHost))
	}
	if event.URLPath != "" {
		fields = append(fields, URLPath(event.URLPath))
	}
	if event.Status > 0 {
		fields = append(fields, Status(event.Status))
	}
	if event.Duration > 0 {
		fields = append(fields, DurationMSFloat(event.Duration))
	}
	if event.ErrorCode != "" {
		fields = append(fields, ErrorCode(event.ErrorCode))
	}
	if event.Err != nil {
		fields = append(fields, ErrorField(event.Err))
	}
	fields = append(fields, AttrsToFields(event.Fields...)...)

	writeFields(logger, dependencyFailureLevel(event.Status, event.Err), dependencyFailureMessage(event), dedupeFields(fields)...)
}

func ObserveHTTPDependency(dependency, operation, activity string, req *http.Request, status int, duration time.Duration, err error) DependencyFailure {
	host, path := SanitizeURLRequest(req)
	method := ""
	if req != nil {
		method = req.Method
	}
	return DependencyFailure{
		Dependency: dependency,
		Operation:  operation,
		Activity:   activity,
		Method:     method,
		URLHost:    host,
		URLPath:    path,
		Status:     status,
		Duration:   duration,
		Err:        err,
	}
}

func dependencyFailureMessage(event DependencyFailure) string {
	if event.Message != "" {
		return event.Message
	}
	dependency := event.Dependency
	if dependency == "" {
		dependency = "dependency"
	}
	operation := event.Operation
	if operation == "" {
		operation = "request"
	}
	if event.Activity != "" {
		return dependency + " " + operation + " failed while " + event.Activity
	}
	return dependency + " " + operation + " failed"
}

func dependencyFailureLevel(status int, err error) zapcore.Level {
	switch {
	case err != nil || status >= http.StatusInternalServerError:
		return LevelError
	case status >= http.StatusBadRequest:
		return LevelWarn
	default:
		return LevelInfo
	}
}
