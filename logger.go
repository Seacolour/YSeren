package main

import (
	"log/slog"
	"os"
)

// Logger 是全局日志实例
var Logger = slog.Default()

// InitLogger 初始化日志系统
// level: "debug", "info", "warn", "error"
func InitLogger(level string) {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	// 使用文本格式输出到 stderr（便于终端查看）
	handler := slog.NewTextHandler(os.Stderr, opts)
	Logger = slog.New(handler)

	// 设置为默认 logger（便于其他包使用 slog.Info 等）
	slog.SetDefault(Logger)
}

// LogInfo 记录 info 级别日志
func LogInfo(msg string, args ...any) {
	Logger.Info(msg, args...)
}

// LogDebug 记录 debug 级别日志
func LogDebug(msg string, args ...any) {
	Logger.Debug(msg, args...)
}

// LogWarn 记录 warn 级别日志
func LogWarn(msg string, args ...any) {
	Logger.Warn(msg, args...)
}

// LogError 记录 error 级别日志
func LogError(msg string, args ...any) {
	Logger.Error(msg, args...)
}
