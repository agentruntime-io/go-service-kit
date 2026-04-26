package logging

import (
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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
	switch cfg.normalizedFormat() {
	case FormatJSON:
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	default:
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level)
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

func defaultLevel(level string) string {
	if level == "" {
		return "info"
	}
	return level
}

func rfc3339MillisUTCEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.UTC().Format("2006-01-02T15:04:05.000Z"))
}
