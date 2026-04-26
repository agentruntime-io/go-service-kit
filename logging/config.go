package logging

import (
	"io"

	"go.uber.org/zap/zapcore"
)

const (
	FormatText       = "text"
	FormatPrettyText = "prettytext"
	FormatJSON       = "json"
)

type Level = zapcore.Level

const (
	LevelDebug = zapcore.DebugLevel
	LevelInfo  = zapcore.InfoLevel
	LevelWarn  = zapcore.WarnLevel
	LevelError = zapcore.ErrorLevel
)

// Config controls shared logger construction for Go backend services.
type Config struct {
	Service string
	Format  string
	Level   string

	Output io.Writer

	AddCaller       bool
	DisableCaller   bool
	StacktraceLevel string
}

func (c Config) normalizedFormat() string {
	switch c.Format {
	case "", FormatText:
		return FormatText
	case FormatJSON, FormatPrettyText:
		return c.Format
	default:
		return FormatText
	}
}

func (c Config) shouldAddCaller() bool {
	if c.DisableCaller {
		return false
	}
	if c.AddCaller {
		return true
	}
	return true
}

func (c Config) stacktraceAt() zapcore.Level {
	switch c.StacktraceLevel {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "", "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.ErrorLevel
	}
}
