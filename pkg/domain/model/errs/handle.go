package errs

import (
	"context"
	"log/slog"

	"github.com/getsentry/sentry-go"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

func Handle(ctx context.Context, err error) {
	logAttrs := []any{slog.Any("error", err)}
	logger := logging.From(ctx)

	hub := sentry.CurrentHub().Clone()
	hub.ConfigureScope(func(scope *sentry.Scope) {
		for k, v := range goerr.Values(err) {
			scope.SetExtra(k, v)
		}
	})
	evID := hub.CaptureException(err)
	logAttrs = append(logAttrs, slog.Any("sentry.id", evID))

	logger.Error("Error: "+err.Error(), logAttrs...)
}
