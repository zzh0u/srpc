package log

import (
	"context"
	"log/slog"
	"os"
)

// Logger 日志记录器，包装 slog.Logger
type Logger struct {
	logger *slog.Logger
}

// NewLogger 创建新的日志记录器，默认使用JSON处理器
func NewLogger() *Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	return &Logger{
		logger: slog.New(handler),
	}
}

// WithLevel 设置日志级别
func (l *Logger) WithLevel(level slog.Level) *Logger {
	handler := l.logger.Handler()
	newHandler := handler.WithAttrs([]slog.Attr{
		slog.String("level", level.String()),
	})
	return &Logger{
		logger: slog.New(newHandler),
	}
}

// Info 记录信息级别日志
func (l *Logger) Info(message string, fields ...map[string]interface{}) {
	l.log(slog.LevelInfo, message, fields...)
}

// Warn 记录警告级别日志
func (l *Logger) Warn(message string, fields ...map[string]interface{}) {
	l.log(slog.LevelWarn, message, fields...)
}

// Error 记录错误级别日志
func (l *Logger) Error(message string, fields ...map[string]interface{}) {
	l.log(slog.LevelError, message, fields...)
}

// log 内部日志记录方法
func (l *Logger) log(level slog.Level, message string, fields ...map[string]interface{}) {
	var attrs []any
	if len(fields) > 0 && fields[0] != nil {
		// 将字段转换为slog.Attr
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
