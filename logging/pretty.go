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

type prettyKind string

const (
	kindGeneric    prettyKind = "generic"
	kindRequest    prettyKind = "request"
	kindDependency prettyKind = "dependency"
	kindDB         prettyKind = "db"
	kindPhase      prettyKind = "phase"
	kindStartup    prettyKind = "startup"
)

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
	message := entry.Message

	kind := classifyPrettyKind(entry, fieldMap, hasSQL)
	delete(fieldMap, "kind")

	buf.AppendString(renderPrettyHeader(entry, kind, color))
	buf.AppendByte('\n')

	lines := renderPrettyLines(kind, message, fieldMap, sqlText, hasSQL, color)
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		buf.AppendString("  ")
		buf.AppendString(line)
		buf.AppendByte('\n')
	}

	if hasStacktrace {
		for _, line := range strings.Split(strings.TrimRight(stacktrace, "\n"), "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			buf.AppendString("  ")
			buf.AppendString(colorize(line, ansiRed, color))
			buf.AppendByte('\n')
		}
	}

	return buf.String()
}

func renderPrettyHeader(entry zapcore.Entry, kind prettyKind, color bool) string {
	parts := []string{
		entry.Time.UTC().Format("2006-01-02T15:04:05.000Z"),
		padRight(formatPrettyLevel(entry.Level, color), 5),
		padRight(colorize(string(kind), prettyKindColor(kind), color), 10),
	}
	if entry.Caller.Defined {
		parts = append(parts, colorize(entry.Caller.TrimmedPath(), ansiCyan, color))
	}
	return strings.Join(parts, " ")
}

func renderPrettyLines(kind prettyKind, message string, fieldMap map[string]any, sqlText string, hasSQL bool, color bool) []string {
	switch kind {
	case kindRequest:
		return renderPrettyRequest(message, fieldMap, color)
	case kindDependency:
		return renderPrettyDependency(message, fieldMap, color)
	case kindDB:
		return renderPrettyDB(message, fieldMap, sqlText, hasSQL, color)
	case kindPhase:
		return renderPrettyPhase(message, fieldMap, color)
	case kindStartup:
		return renderPrettyGeneric(message, fieldMap, color)
	default:
		return renderPrettyGeneric(message, fieldMap, color)
	}
}

func renderPrettyRequest(message string, fieldMap map[string]any, color bool) []string {
	method, _ := popString(fieldMap, "method")
	route, _ := popString(fieldMap, "route")
	if route == "" {
		route, _ = popString(fieldMap, "path")
	}
	statusValue, hasStatus := popInt(fieldMap, "status")
	durationText := popFormattedDuration(fieldMap)

	summaryParts := []string{}
	if method != "" {
		summaryParts = append(summaryParts, colorize(method, ansiBlue, color))
	}
	if route != "" {
		summaryParts = append(summaryParts, route)
	}
	if hasStatus {
		summaryParts = append(summaryParts, colorize(strconv.Itoa(statusValue), statusColor(statusValue), color))
	}
	if durationText != "" {
		summaryParts = append(summaryParts, colorize(durationText, ansiMagenta, color))
	}

	lines := []string{}
	if len(summaryParts) > 0 {
		lines = append(lines, strings.Join(summaryParts, " "))
	}

	correlation := formatCompactFields(fieldMap, color,
		"service", "request_id", "trace_id", "span_id", "trace_sampled",
		"tenant_id", "project_id", "user_id", "workflow_id", "run_id",
		"instance_id", "server_id", "bytes", "remote_addr", "real_ip",
		"forwarded_for", "user_agent",
	)
	if correlation != "" {
		lines = append(lines, correlation)
	}

	if details := formatCompactFields(fieldMap, color, "error_code", "dependency", "error_message"); details != "" {
		lines = append(lines, details)
	}

	if showRequestMessage(message) {
		lines = append(lines, message)
	}

	if extras := formatRemainingFields(fieldMap, color); extras != "" {
		lines = append(lines, extras)
	}
	return lines
}

func renderPrettyDependency(message string, fieldMap map[string]any, color bool) []string {
	lines := []string{message}

	detailsParts := []string{}
	if text := popFormattedField(fieldMap, "dependency", color); text != "" {
		detailsParts = append(detailsParts, text)
	}
	if text := popFormattedField(fieldMap, "operation", color); text != "" {
		detailsParts = append(detailsParts, text)
	}
	if text := popFormattedField(fieldMap, "method", color); text != "" {
		detailsParts = append(detailsParts, text)
	}
	if text := popFormattedField(fieldMap, "status", color); text != "" {
		detailsParts = append(detailsParts, text)
	}
	if durationText := popFormattedDuration(fieldMap); durationText != "" {
		detailsParts = append(detailsParts, colorize(durationText, ansiMagenta, color))
	}
	if text := popFormattedField(fieldMap, "url_host", color); text != "" {
		detailsParts = append(detailsParts, text)
	}
	if text := popFormattedField(fieldMap, "url_path", color); text != "" {
		detailsParts = append(detailsParts, text)
	}
	if text := popFormattedField(fieldMap, "error_code", color); text != "" {
		detailsParts = append(detailsParts, text)
	}
	if details := strings.Join(detailsParts, " "); details != "" {
		lines = append(lines, details)
	}

	if correlation := formatCompactFields(fieldMap, color,
		"service", "request_id", "trace_id", "span_id", "trace_sampled",
		"tenant_id", "project_id", "user_id", "workflow_id", "run_id",
		"instance_id", "server_id",
	); correlation != "" {
		lines = append(lines, correlation)
	}

	if errLine := formatCompactFields(fieldMap, color, "error_message", "error"); errLine != "" {
		lines = append(lines, errLine)
	}
	if extras := formatRemainingFields(fieldMap, color); extras != "" {
		lines = append(lines, extras)
	}
	return lines
}

func renderPrettyDB(message string, fieldMap map[string]any, sqlText string, hasSQL bool, color bool) []string {
	lines := []string{}

	durationText := popFormattedDuration(fieldMap)
	rowsText := popFormattedField(fieldMap, "rows", color)
	thresholdText := popThreshold(fieldMap, color)
	errorText := popFormattedField(fieldMap, "error", color)

	summaryParts := []string{}
	if strings.Contains(strings.ToLower(message), "slow") {
		summaryParts = append(summaryParts, colorize("slow query", ansiYellow, color)+":")
	} else if message != "" {
		summaryParts = append(summaryParts, message)
	}
	if durationText != "" {
		summaryParts = append(summaryParts, colorize(durationText, ansiMagenta, color))
	}
	if rowsText != "" {
		summaryParts = append(summaryParts, rowsText)
	}
	if thresholdText != "" {
		summaryParts = append(summaryParts, thresholdText)
	}
	if errorText != "" {
		summaryParts = append(summaryParts, errorText)
	}
	if len(summaryParts) > 0 {
		lines = append(lines, strings.Join(summaryParts, " "))
	}

	if correlation := formatCompactFields(fieldMap, color,
		"service", "request_id", "trace_id", "span_id", "trace_sampled",
		"tenant_id", "workflow_id", "run_id", "instance_id", "server_id",
	); correlation != "" {
		lines = append(lines, correlation)
	}

	if hasSQL {
		lines = append(lines, renderSQLLines(sqlText, color)...)
	}

	if extras := formatRemainingFields(fieldMap, color); extras != "" {
		lines = append(lines, extras)
	}
	return lines
}

func renderPrettyPhase(message string, fieldMap map[string]any, color bool) []string {
	lines := []string{message}
	if details := formatCompactFields(fieldMap, color,
		"phase", "component", "decision", "reason", "status",
		"tenant_id", "project_id", "user_id", "workflow_id", "run_id",
		"instance_id", "server_id", "request_id", "trace_id", "span_id",
	); details != "" {
		lines = append(lines, details)
	}
	if extras := formatRemainingFields(fieldMap, color); extras != "" {
		lines = append(lines, extras)
	}
	return lines
}

func renderPrettyGeneric(message string, fieldMap map[string]any, color bool) []string {
	lines := []string{}
	if showPrettyMessage(message) {
		lines = append(lines, message)
	}
	if details := formatCompactFields(fieldMap, color,
		"service", "env", "version", "addr", "component", "phase",
		"dependency", "operation", "method", "route", "path", "status",
		"duration_ms", "tenant_id", "project_id", "user_id", "workflow_id",
		"run_id", "instance_id", "server_id", "request_id", "trace_id",
		"span_id", "trace_sampled", "error_code", "error_message", "error",
	); details != "" {
		lines = append(lines, details)
	}
	if extras := formatRemainingFields(fieldMap, color); extras != "" {
		lines = append(lines, extras)
	}
	return lines
}

func renderSQLLines(sqlText string, color bool) []string {
	sqlText = strings.TrimSpace(sqlText)
	if sqlText == "" {
		return nil
	}

	upper := strings.ToUpper(sqlText)
	for _, kw := range []string{" VALUES ", " WHERE ", " SET ", " FROM ", " ORDER BY ", " LIMIT "} {
		upperKW := strings.ToUpper(kw)
		sqlText = insertBreakPreserveCase(sqlText, upper, upperKW)
		upper = strings.ToUpper(sqlText)
	}

	rawLines := strings.Split(sqlText, "\n")
	lines := make([]string, 0, len(rawLines))
	for i, line := range rawLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if i == 0 {
			lines = append(lines, colorize(line, ansiWhite, color))
		} else {
			lines = append(lines, "  "+colorize(line, ansiWhite, color))
		}
	}
	return lines
}

func insertBreakPreserveCase(sqlText, upperText, upperKeyword string) string {
	idx := strings.Index(upperText, upperKeyword)
	if idx <= 0 {
		return sqlText
	}
	return sqlText[:idx] + "\n" + strings.TrimLeft(sqlText[idx:], " ")
}

func classifyPrettyKind(entry zapcore.Entry, fieldMap map[string]any, hasSQL bool) prettyKind {
	if kindValue, ok := fieldMap["kind"].(string); ok && kindValue != "" {
		switch prettyKind(kindValue) {
		case kindRequest, kindDependency, kindDB, kindPhase, kindStartup:
			return prettyKind(kindValue)
		}
	}
	switch {
	case hasSQL:
		return kindDB
	case hasField(fieldMap, "dependency") || hasField(fieldMap, "operation") && (hasField(fieldMap, "url_host") || hasField(fieldMap, "url_path")):
		return kindDependency
	case hasField(fieldMap, "method") && (hasField(fieldMap, "route") || hasField(fieldMap, "path")) && hasField(fieldMap, "status"):
		return kindRequest
	case hasField(fieldMap, "phase"):
		return kindPhase
	case looksLikeStartup(entry.Message):
		return kindStartup
	default:
		return kindGeneric
	}
}

func hasField(fieldMap map[string]any, key string) bool {
	_, ok := fieldMap[key]
	return ok
}

func looksLikeStartup(message string) bool {
	msg := strings.ToLower(strings.TrimSpace(message))
	switch {
	case strings.Contains(msg, "starting"),
		strings.Contains(msg, "listening"),
		strings.Contains(msg, "shutdown"),
		strings.Contains(msg, "stopped"),
		strings.Contains(msg, "initialized"):
		return true
	default:
		return false
	}
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

func popInt(fieldMap map[string]any, key string) (int, bool) {
	value, ok := fieldMap[key]
	if !ok {
		return 0, false
	}
	delete(fieldMap, key)
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	default:
		return 0, false
	}
}

func popFormattedDuration(fieldMap map[string]any) string {
	value, ok := fieldMap["duration_ms"]
	if !ok {
		return ""
	}
	delete(fieldMap, "duration_ms")
	switch typed := value.(type) {
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64) + "ms"
	case int64:
		return strconv.FormatInt(typed, 10) + "ms"
	case int:
		return strconv.Itoa(typed) + "ms"
	default:
		return fmt.Sprint(typed) + "ms"
	}
}

func popFormattedField(fieldMap map[string]any, key string, color bool) string {
	value, ok := fieldMap[key]
	if !ok {
		return ""
	}
	delete(fieldMap, key)
	return formatPrettyField(key, value, color)
}

func popThreshold(fieldMap map[string]any, color bool) string {
	value, ok := fieldMap["slow_threshold_ms"]
	if !ok {
		return ""
	}
	delete(fieldMap, "slow_threshold_ms")
	switch typed := value.(type) {
	case float64:
		return colorize("threshold="+strconv.FormatFloat(typed, 'f', -1, 64)+"ms", ansiGray, color)
	case int64:
		return colorize("threshold="+strconv.FormatInt(typed, 10)+"ms", ansiGray, color)
	case int:
		return colorize("threshold="+strconv.Itoa(typed)+"ms", ansiGray, color)
	default:
		return colorize("threshold="+fmt.Sprint(typed)+"ms", ansiGray, color)
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

func formatCompactFields(fieldMap map[string]any, color bool, keys ...string) string {
	parts := []string{}
	for _, key := range keys {
		if value, ok := fieldMap[key]; ok {
			parts = append(parts, formatPrettyField(key, value, color))
			delete(fieldMap, key)
		}
	}
	return strings.Join(parts, " ")
}

func formatRemainingFields(fieldMap map[string]any, color bool) string {
	keys := preferredFieldOrder(fieldMap)
	seen := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		seen[key] = struct{}{}
	}
	rest := make([]string, 0, len(fieldMap))
	for _, key := range keys {
		if value, ok := fieldMap[key]; ok {
			rest = append(rest, formatPrettyField(key, value, color))
			delete(fieldMap, key)
		}
	}
	if len(fieldMap) > 0 {
		keys := make([]string, 0, len(fieldMap))
		for key := range fieldMap {
			if _, ok := seen[key]; ok {
				continue
			}
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			rest = append(rest, formatPrettyField(key, fieldMap[key], color))
			delete(fieldMap, key)
		}
	}
	return strings.Join(rest, " ")
}

func showPrettyMessage(message string) bool {
	return strings.TrimSpace(message) != ""
}

func showRequestMessage(message string) bool {
	msg := strings.ToLower(strings.TrimSpace(message))
	if msg == "" {
		return false
	}
	return !strings.HasSuffix(msg, "request completed")
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
	text := strings.ToUpper(level.String())
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

func prettyKindColor(kind prettyKind) string {
	switch kind {
	case kindRequest:
		return ansiCyan
	case kindDependency:
		return ansiRed
	case kindDB:
		return ansiYellow
	case kindPhase:
		return ansiBlue
	case kindStartup:
		return ansiGreen
	default:
		return ansiGray
	}
}

func statusColor(status int) string {
	switch {
	case status >= 500:
		return ansiRed
	case status >= 400:
		return ansiYellow
	default:
		return ansiGreen
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

func padRight(text string, width int) string {
	plain := stripANSI(text)
	if len(plain) >= width {
		return text
	}
	return text + strings.Repeat(" ", width-len(plain))
}

func stripANSI(text string) string {
	text = strings.ReplaceAll(text, ansiReset, "")
	for _, code := range []string{ansiRed, ansiGreen, ansiYellow, ansiBlue, ansiCyan, ansiGray, ansiMagenta, ansiWhite} {
		text = strings.ReplaceAll(text, code, "")
	}
	return text
}

const (
	ansiReset   = "\x1b[0m"
	ansiRed     = "\x1b[31m"
	ansiGreen   = "\x1b[32m"
	ansiYellow  = "\x1b[33m"
	ansiBlue    = "\x1b[34m"
	ansiMagenta = "\x1b[35m"
	ansiCyan    = "\x1b[36m"
	ansiWhite   = "\x1b[37m"
	ansiGray    = "\x1b[90m"
)

func colorize(text, code string, enabled bool) string {
	if !enabled || text == "" {
		return text
	}
	return code + text + ansiReset
}
