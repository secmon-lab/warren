package errs

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/getsentry/sentry-go"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/request_id"
)

func Handle(ctx context.Context, err error) {
	defer func() {
		if r := recover(); r != nil {
			// Ultimate fallback to stderr if slog crashes
			fmt.Fprintf(os.Stderr, "[CRITICAL] slog crashed during error handling: original_error=%s, slog_panic=%v\n",
				err.Error(), r)
		}
	}()

	logAttrs := []any{slog.Any("error", err)}
	logger := logging.From(ctx)

	hub := sentry.CurrentHub().Clone()
	hub.ConfigureScope(func(scope *sentry.Scope) {
		// Add request ID from context
		if reqID := request_id.FromContext(ctx); reqID != "" && reqID != "(unknown)" {
			scope.SetTag("request_id", reqID)
		}

		for k, v := range goerr.Values(err) {
			scope.SetExtra(k, v)
		}
	})
	evID := hub.CaptureException(err)
	logAttrs = append(logAttrs, slog.Any("sentry.id", evID))

	logger.Error("Error: "+err.Error(), logAttrs...)
}
