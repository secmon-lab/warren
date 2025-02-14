package logging

import (
	"context"
	"log/slog"
)

type loggerKeyType string

const loggerKey loggerKeyType = "logger"

func With(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

func From(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return logger
	}
	return Default()
}
