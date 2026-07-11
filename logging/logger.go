package logging

import (
	"context"
	"fmt"
	"math"
	"os"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	defaultLoggerMu sync.RWMutex
	defaultLogger   = zap.NewNop()
)

func New(cfg Config) (*zap.Logger, error) {
	level := zap.NewAtomicLevel()
	if err := level.UnmarshalText([]byte(defaultLevel(cfg.Level))); err != nil {
		return nil, fmt.Errorf("parse log level: %w", err)
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     rfc3339MillisUTCEncoder,
		EncodeDuration: zapcore.MillisDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var encoder zapcore.Encoder
	var core zapcore.Core
	switch cfg.normalizedFormat() {
	case FormatJSON:
		encoder = zapcore.NewJSONEncoder(encoderConfig)
		output := cfg.Output
		if output == nil {
			output = os.Stdout
		}
		core = zapcore.NewCore(encoder, zapcore.AddSync(output), level)
	case FormatPrettyText:
		output := cfg.Output
		if output == nil {
			output = os.Stdout
		}
		core = newPrettyCore(level, zapcore.AddSync(output), prettyColorEnabled(output))
	default:
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
		output := cfg.Output
		if output == nil {
			output = os.Stdout
		}
		core = zapcore.NewCore(encoder, zapcore.AddSync(output), level)
	}

	options := []zap.Option{
		zap.AddStacktrace(cfg.stacktraceAt()),
	}
	if cfg.shouldAddCaller() {
		options = append(options, zap.AddCaller())
	}

	logger := zap.New(core, options...)
	if cfg.Service != "" {
		logger = logger.With(Service(cfg.Service))
	}
	return logger, nil
}

func SetDefault(logger *zap.Logger) {
	if logger == nil {
		logger = zap.NewNop()
	}
	defaultLoggerMu.Lock()
	defaultLogger = logger
	defaultLoggerMu.Unlock()
	zap.ReplaceGlobals(logger)
}

func L() *zap.Logger {
	defaultLoggerMu.RLock()
	defer defaultLoggerMu.RUnlock()
	return defaultLogger
}

func Debug(msg string, args ...any) { write(nil, LevelDebug, msg, args...) }
func Info(msg string, args ...any)  { write(nil, LevelInfo, msg, args...) }
func Warn(msg string, args ...any)  { write(nil, LevelWarn, msg, args...) }
func Error(msg string, args ...any) { write(nil, LevelError, msg, args...) }

func DebugContext(ctx context.Context, msg string, args ...any) { write(ctx, LevelDebug, msg, args...) }
func InfoContext(ctx context.Context, msg string, args ...any)  { write(ctx, LevelInfo, msg, args...) }
func WarnContext(ctx context.Context, msg string, args ...any)  { write(ctx, LevelWarn, msg, args...) }
func ErrorContext(ctx context.Context, msg string, args ...any) { write(ctx, LevelError, msg, args...) }

// write is the core dispatch function.
//
// For context-aware calls it does two things:
//  1. Emits a structured log that includes both correlation fields (trace_id,
//     span_id, tenant_id, …) and the caller's own fields.
//  2. Mirrors the call onto the active OTel span as a named event so the
//     message and its fields are visible directly in the trace UI — the same
//     pattern as the "logs" entries in a Jaeger trace.
func write(ctx context.Context, level zapcore.Level, msg string, args ...any) {
	logger := L()
	if logger == nil {
		return
	}

	// Split caller fields out so they can be sent to the span separately.
	callerFields := AttrsToFields(args...)

	// Structured log: correlation fields + caller fields.
	logFields := make([]zap.Field, 0, len(callerFields)+8)
	if ctx != nil {
		logFields = append(logFields, CorrelationFields(ctx)...)
	}
	logFields = append(logFields, callerFields...)
	writeFields(logger, level, msg, logFields...)

	// Span event: caller fields only — trace_id/span_id live on the span
	// context already; repeating them in every event would be noise.
	if ctx != nil {
		addSpanLogEvent(ctx, msg, callerFields)
	}
}

func writeFields(logger *zap.Logger, level zapcore.Level, msg string, fields ...zap.Field) {
	if logger == nil {
		return
	}
	if ce := logger.Check(level, msg); ce != nil {
		ce.Write(fields...)
	}
}

// addSpanLogEvent adds the log message as a named event on the active span so
// that every InfoContext / ErrorContext / … call appears in the trace exactly
// like the "logs" entries visible in the Jaeger UI.
func addSpanLogEvent(ctx context.Context, msg string, fields []zap.Field) {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return
	}
	attrs := make([]attribute.KeyValue, 0, len(fields))
	for _, f := range fields {
		if kv, ok := fieldToAttribute(f); ok {
			attrs = append(attrs, kv)
		}
	}
	span.AddEvent(msg, trace.WithAttributes(attrs...))
}

// fieldToAttribute converts a single zap.Field to an OTel attribute.KeyValue.
// Returns false for field types that have no meaningful attribute representation
// (e.g. namespace markers, binary blobs).
func fieldToAttribute(f zap.Field) (attribute.KeyValue, bool) {
	switch f.Type {
	case zapcore.StringType:
		return attribute.String(f.Key, f.String), true

	case zapcore.ErrorType:
		if f.Interface != nil {
			return attribute.String(f.Key, f.Interface.(error).Error()), true
		}

	case zapcore.Int64Type, zapcore.Int32Type, zapcore.Int16Type, zapcore.Int8Type:
		return attribute.Int64(f.Key, f.Integer), true

	case zapcore.Uint64Type:
		return attribute.Int64(f.Key, int64(uint64(f.Integer))), true

	case zapcore.Uint32Type, zapcore.Uint16Type, zapcore.Uint8Type:
		return attribute.Int64(f.Key, f.Integer), true

	case zapcore.Float64Type:
		return attribute.Float64(f.Key, math.Float64frombits(uint64(f.Integer))), true

	case zapcore.Float32Type:
		return attribute.Float64(f.Key, float64(math.Float32frombits(uint32(f.Integer)))), true

	case zapcore.BoolType:
		return attribute.Bool(f.Key, f.Integer == 1), true

	case zapcore.DurationType:
		return attribute.Int64(f.Key+"_ms", time.Duration(f.Integer).Milliseconds()), true

	case zapcore.StringerType:
		if s, ok := f.Interface.(fmt.Stringer); ok {
			return attribute.String(f.Key, s.String()), true
		}

	case zapcore.ReflectType:
		return attribute.String(f.Key, fmt.Sprintf("%v", f.Interface)), true
	}
	return attribute.KeyValue{}, false
}

func defaultLevel(level string) string {
	if level == "" {
		return "info"
	}
	return level
}

func rfc3339MillisUTCEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.UTC().Format("2006-01-02T15:04:05.000Z"))
}
