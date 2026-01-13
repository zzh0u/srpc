package log

import (
	"context"
	"log/slog"
	"os"
)

// Slogger 日志记录器，包装 slog.Logger
type Slogger struct {
	logger *slog.Logger
}

// NewLogger 创建新的日志记录器，默认使用 JSON 处理器
func NewLogger() *Slogger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	return &Slogger{
		logger: slog.New(handler),
	}
}

// Info 记录信息级别日志
func (l *Slogger) Info(message string, fields ...map[string]interface{}) {
	l.log(slog.LevelInfo, message, fields...)
}

// Warn 记录警告级别日志
func (l *Slogger) Warn(message string, fields ...map[string]interface{}) {
	l.log(slog.LevelWarn, message, fields...)
}

// Error 记录错误级别日志
func (l *Slogger) Error(message string, fields ...map[string]interface{}) {
	l.log(slog.LevelError, message, fields...)
}

// log 内部日志记录方法
func (l *Slogger) log(level slog.Level, message string, fields ...map[string]interface{}) {
	var attrs []any
	if len(fields) > 0 && fields[0] != nil {
		// 将字段转换为 slog.Attr
		for k, v := range fields[0] {
			attrs = append(attrs, slog.Any(k, v))
		}
	}

	switch level {
	case slog.LevelInfo:
		l.logger.Info(message, attrs...)
	case slog.LevelWarn:
		l.logger.Warn(message, attrs...)
	case slog.LevelError:
		l.logger.Error(message, attrs...)
	default:
		l.logger.Log(context.Background(), level, message, attrs...)
	}
}
