package logger

import (
	"context"
	"log/slog"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"
)

type TeeHandler struct {
	Terminal slog.Handler
	File     slog.Handler
}

func (t *TeeHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return t.Terminal.Enabled(ctx, level) || t.File.Enabled(ctx, level)
}

func (t *TeeHandler) Handle(ctx context.Context, r slog.Record) error {
	_ = t.Terminal.Handle(ctx, r)
	return t.File.Handle(ctx, r)
}

func (t *TeeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &TeeHandler{
		Terminal: t.Terminal.WithAttrs(attrs),
		File:     t.File.WithAttrs(attrs),
	}
}

func (t *TeeHandler) WithGroup(name string) slog.Handler {
	return &TeeHandler{
		Terminal: t.Terminal.WithGroup(name),
		File:     t.File.WithGroup(name),
	}
}

func SetupLogger(env string) {
	os.MkdirAll("logs", os.ModePerm)

	fileWriter := &lumberjack.Logger{
		Filename:   "logs/api.log",
		MaxSize:    10,
		MaxBackups: 30,
		MaxAge:     30,
		Compress:   true,
	}

	var level slog.Level
	if env == "production" {
		level = slog.LevelInfo
	} else {
		level = slog.LevelDebug
	}
	opts := &slog.HandlerOptions{Level: level}

	terminalHandler := slog.NewTextHandler(os.Stdout, opts)
	fileHandler := slog.NewJSONHandler(fileWriter, opts)

	slog.SetDefault(slog.New(&TeeHandler{
		Terminal: terminalHandler,
		File:     fileHandler,
	}))
}
