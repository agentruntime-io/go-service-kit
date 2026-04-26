package logging

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	gormlogger "gorm.io/gorm/logger"
)

type GormOptions struct {
	SlowThreshold time.Duration
}

type gormAdapter struct {
	logLevel    gormlogger.LogLevel
	slowQueryAt time.Duration
}

func NewGORMLogger(opts GormOptions) gormlogger.Interface {
	slowThreshold := opts.SlowThreshold
	if slowThreshold <= 0 {
		slowThreshold = 200 * time.Millisecond
	}
	return &gormAdapter{
		logLevel:    gormlogger.Warn,
		slowQueryAt: slowThreshold,
	}
}

func (l *gormAdapter) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	next := *l
	next.logLevel = level
	return &next
}

func (l *gormAdapter) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= gormlogger.Info {
		InfoContext(ctx, formatGormMessage(msg, data...))
	}
}

func (l *gormAdapter) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= gormlogger.Warn {
		WarnContext(ctx, formatGormMessage(msg, data...))
	}
}

func (l *gormAdapter) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.logLevel >= gormlogger.Error {
		ErrorContext(ctx, formatGormMessage(msg, data...))
	}
}

func (l *gormAdapter) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.logLevel <= gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()
	attrs := []any{
		Kind("db"),
		DurationMSFloat(elapsed),
		"rows", rows,
		"sql", compactSQL(sql),
	}

	switch {
	case err != nil && l.logLevel >= gormlogger.Error && !errors.Is(err, gormlogger.ErrRecordNotFound):
		ErrorContext(ctx, "db query failed", append(attrs, "error", err.Error())...)
	case elapsed >= l.slowQueryAt && l.logLevel >= gormlogger.Warn:
		WarnContext(ctx, "db query slow", append(attrs, "slow_threshold_ms", float64(l.slowQueryAt.Microseconds())/1000)...)
	}
}

func (l *gormAdapter) ParamsFilter(_ context.Context, sql string, _ ...interface{}) (string, []interface{}) {
	return sql, nil
}

func compactSQL(sql string) string {
	sql = strings.Join(strings.Fields(strings.TrimSpace(sql)), " ")
	if len(sql) > 1000 {
		return sql[:1000] + "..."
	}
	return sql
}

func formatGormMessage(msg string, data ...interface{}) string {
	if len(data) == 0 {
		return msg
	}
	return fmt.Sprintf(msg, data...)
}
