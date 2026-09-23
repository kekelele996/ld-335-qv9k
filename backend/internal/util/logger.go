package util

import (
	"log/slog"
	"os"
)

// NewLogger 构造 JSON 结构化日志。
func NewLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

// LogError 记录错误日志并返回原错误（不吞错）。
func LogError(log *slog.Logger, template string, err error, attrs ...any) error {
	if err != nil {
		log.Error(template, append(attrs, "error", err.Error())...)
	}
	return err
}
