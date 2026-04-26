package logging

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

func Service(value string) zap.Field      { return zap.String("service", value) }
func RequestID(value string) zap.Field    { return zap.String("request_id", value) }
func TraceID(value string) zap.Field      { return zap.String("trace_id", value) }
func SpanID(value string) zap.Field       { return zap.String("span_id", value) }
func TenantID(value string) zap.Field     { return zap.String("tenant_id", value) }
func ProjectID(value string) zap.Field    { return zap.String("project_id", value) }
func UserID(value string) zap.Field       { return zap.String("user_id", value) }
func WorkflowID(value string) zap.Field   { return zap.String("workflow_id", value) }
func RunID(value string) zap.Field        { return zap.String("run_id", value) }
func InstanceID(value string) zap.Field   { return zap.String("instance_id", value) }
func ServerID(value string) zap.Field     { return zap.String("server_id", value) }
func ConnectionID(value string) zap.Field { return zap.String("connection_id", value) }
func ProfileID(value string) zap.Field    { return zap.String("profile_id", value) }
func Component(value string) zap.Field    { return zap.String("component", value) }
func Phase(value string) zap.Field        { return zap.String("phase", value) }
func Operation(value string) zap.Field    { return zap.String("operation", value) }
func Dependency(value string) zap.Field   { return zap.String("dependency", value) }
func ErrorCode(value string) zap.Field    { return zap.String("error_code", value) }
func ErrorMessage(value string) zap.Field { return zap.String("error_message", value) }
func Reason(value string) zap.Field       { return zap.String("reason", value) }
func Decision(value string) zap.Field     { return zap.String("decision", value) }
func URLHost(value string) zap.Field      { return zap.String("url_host", value) }
func URLPath(value string) zap.Field      { return zap.String("url_path", value) }
func Method(value string) zap.Field       { return zap.String("method", value) }
func Route(value string) zap.Field        { return zap.String("route", value) }
func Path(value string) zap.Field         { return zap.String("path", value) }
func UserAgent(value string) zap.Field    { return zap.String("user_agent", value) }
func RemoteAddr(value string) zap.Field   { return zap.String("remote_addr", value) }

func Status(value int) zap.Field { return zap.Int("status", value) }
func DurationMS(value time.Duration) zap.Field {
	return zap.Int64("duration_ms", value.Milliseconds())
}

func Error(err error) zap.Field {
	return zap.Error(err)
}

func HTTPRequestFields(r *http.Request) []zap.Field {
	if r == nil {
		return nil
	}
	fields := []zap.Field{
		Method(r.Method),
		Path(r.URL.Path),
		RemoteAddr(r.RemoteAddr),
	}
	if ua := r.UserAgent(); ua != "" {
		fields = append(fields, UserAgent(ua))
	}
	return fields
}
