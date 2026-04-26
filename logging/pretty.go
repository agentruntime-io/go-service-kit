package logging

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

type prettyCore struct {
	levelEnabler zapcore.LevelEnabler
	output       zapcore.WriteSyncer
	color        bool
	staticFields []zap.Field
}

func newPrettyCore(levelEnabler zapcore.LevelEnabler, output zapcore.WriteSyncer, color bool) zapcore.Core {
	return &prettyCore{
		levelEnabler: levelEnabler,
		output:       output,
		color:        color,
	}
}

func (c *prettyCore) Enabled(level zapcore.Level) bool {
	return c.levelEnabler.Enabled(level)
}

func (c *prettyCore) With(fields []zap.Field) zapcore.Core {
	next := *c
	next.staticFields = append(append([]zap.Field{}, c.staticFields...), fields...)
	return &next
}

func (c *prettyCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return ce.AddCore(entry, c)
	}
	return ce
}

func (c *prettyCore) Write(entry zapcore.Entry, fields []zap.Field) error {
	allFields := append(append([]zap.Field{}, c.staticFields...), fields...)
	line := formatPrettyEntry(entry, allFields, c.color)
	if _, err := c.output.Write([]byte(line)); err != nil {
		return err
	}
	if entry.Level >= zapcore.ErrorLevel {
		_ = c.output.Sync()
	}
	return nil
}

func (c *prettyCore) Sync() error {
	return c.output.Sync()
}

func formatPrettyEntry(entry zapcore.Entry, fields []zap.Field, color bool) string {
	buf := buffer.NewPool().Get()
	defer buf.Free()

	fieldMap := fieldsToMap(fields)
	sqlText, hasSQL := popString(fieldMap, "sql")
	stacktrace, hasStacktrace := popString(fieldMap, "stacktrace")

	prefixParts := []string{
		entry.Time.UTC().Format("2006-01-02T15:04:05.000Z"),
		formatPrettyLevel(entry.Level, color),
	}
	if entry.Caller.Defined {
		prefixParts = append(prefixParts, colorize(entry.Caller.TrimmedPath(), ansiCyan, color))
	}

	fieldParts := make([]string, 0, len(fieldMap)+1)
	for _, key := range preferredFieldOrder(fieldMap) {
		fieldParts = append(fieldParts, formatPrettyField(key, fieldMap[key], color))
		delete(fieldMap, key)
	}
	if len(fieldMap) > 0 {
		keys := make([]string, 0, len(fieldMap))
		for key := range fieldMap {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			fieldParts = append(fieldParts, formatPrettyField(key, fieldMap[key], color))
		}
	}
	fieldParts = append(fieldParts, formatPrettyField("msg", entry.Message, color))

	buf.AppendString(strings.Join(prefixParts, " "))
	if len(fieldParts) > 0 {
		buf.AppendByte(' ')
		buf.AppendString(strings.Join(fieldParts, " "))
	}
	buf.AppendByte('\n')

	if hasSQL {
		buf.AppendString("    ")
		buf.AppendString(colorize("sql", ansiYellow, color))
		buf.AppendString("=")
		buf.AppendString(strconv.Quote(sqlText))
		buf.AppendByte('\n')
	}
	if hasStacktrace {
		buf.AppendString(stacktrace)
		if !strings.HasSuffix(stacktrace, "\n") {
			buf.AppendByte('\n')
		}
	}
	return buf.String()
}

func fieldsToMap(fields []zap.Field) map[string]any {
	encoder := zapcore.NewMapObjectEncoder()
	for _, field := range dedupeFields(fields) {
		field.AddTo(encoder)
	}
	return encoder.Fields
}

func popString(fieldMap map[string]any, key string) (string, bool) {
	value, ok := fieldMap[key]
	if !ok {
		return "", false
	}
	delete(fieldMap, key)
	switch typed := value.(type) {
	case string:
		return typed, true
	default:
		return fmt.Sprint(typed), true
	}
}

func preferredFieldOrder(fieldMap map[string]any) []string {
	order := []string{
		"service",
		"request_id",
		"trace_id",
		"span_id",
		"trace_sampled",
		"component",
		"phase",
		"dependency",
		"operation",
		"method",
		"route",
		"path",
		"status",
		"duration_ms",
		"slow_threshold_ms",
		"rows",
		"tenant_id",
		"project_id",
		"user_id",
		"workflow_id",
		"run_id",
		"instance_id",
		"server_id",
		"connection_id",
		"profile_id",
		"error_code",
		"error_message",
		"error",
	}
	result := make([]string, 0, len(order))
	for _, key := range order {
		if _, ok := fieldMap[key]; ok {
			result = append(result, key)
		}
	}
	return result
}

func formatPrettyField(key string, value any, color bool) string {
	keyText := colorize(key, ansiGray, color)
	switch typed := value.(type) {
	case string:
		return keyText + "=" + quoteIfNeeded(typed)
	case bool:
		return keyText + "=" + strconv.FormatBool(typed)
	case int:
		return keyText + "=" + strconv.Itoa(typed)
	case int64:
		return keyText + "=" + strconv.FormatInt(typed, 10)
	case float64:
		return keyText + "=" + strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		return keyText + "=" + quoteIfNeeded(fmt.Sprint(typed))
	}
}

func quoteIfNeeded(value string) string {
	if value == "" {
		return `""`
	}
	for _, r := range value {
		if r == ' ' || r == '"' || r == '\n' || r == '\t' || r == '=' {
			return strconv.Quote(value)
		}
	}
	return value
}

func formatPrettyLevel(level zapcore.Level, color bool) string {
	text := level.String()
	switch level {
	case zapcore.DebugLevel:
		return colorize(text, ansiBlue, color)
	case zapcore.InfoLevel:
		return colorize(text, ansiGreen, color)
	case zapcore.WarnLevel:
		return colorize(text, ansiYellow, color)
	case zapcore.ErrorLevel, zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
		return colorize(text, ansiRed, color)
	default:
		return text
	}
}

func prettyColorEnabled(output io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	file, ok := output.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	if info.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	if strings.EqualFold(os.Getenv("TERM"), "dumb") {
		return false
	}
	return true
}

const (
	ansiReset  = "\x1b[0m"
	ansiRed    = "\x1b[31m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
	ansiBlue   = "\x1b[34m"
	ansiCyan   = "\x1b[36m"
	ansiGray   = "\x1b[90m"
)

func colorize(text, code string, enabled bool) string {
	if !enabled || text == "" {
		return text
	}
	return code + text + ansiReset
}
