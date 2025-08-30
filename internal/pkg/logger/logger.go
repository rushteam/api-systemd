package logger

import (
	"context"
	"log/slog"
	"os"
)

var defaultLogger *slog.Logger

func init() {
	// 创建结构化日志记录器
	opts := &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: true,
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	defaultLogger = slog.New(handler)
	slog.SetDefault(defaultLogger)
}

// Info 记录信息级别日志
func Info(ctx context.Context, msg string, args ...any) {
	defaultLogger.InfoContext(ctx, msg, args...)
}

// Error 记录错误级别日志
func Error(ctx context.Context, msg string, args ...any) {
	defaultLogger.ErrorContext(ctx, msg, args...)
}

// Warn 记录警告级别日志
func Warn(ctx context.Context, msg string, args ...any) {
	defaultLogger.WarnContext(ctx, msg, args...)
}

// Debug 记录调试级别日志
func Debug(ctx context.Context, msg string, args ...any) {
	defaultLogger.DebugContext(ctx, msg, args...)
}

// WithFields 添加字段到日志上下文
func WithFields(ctx context.Context, fields map[string]any) context.Context {
	for k, v := range fields {
		ctx = context.WithValue(ctx, k, v)
	}
	return ctx
}
