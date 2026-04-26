package logging

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

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

func write(ctx context.Context, level zapcore.Level, msg string, args ...any) {
	logger := L()
	if logger == nil {
		return
	}

	fields := make([]zap.Field, 0, len(args)+4)
	if ctx != nil {
		fields = append(fields, CorrelationFields(ctx)...)
	}
	fields = append(fields, AttrsToFields(args...)...)
	writeFields(logger, level, msg, fields...)
}

func writeFields(logger *zap.Logger, level zapcore.Level, msg string, fields ...zap.Field) {
	if logger == nil {
		return
	}
	if ce := logger.Check(level, msg); ce != nil {
		ce.Write(fields...)
	}
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
