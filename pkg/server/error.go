package server

import (
	"log/slog"
	"net/http"

	"github.com/getsentry/sentry-go"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

func handleError(w http.ResponseWriter, r *http.Request, err error) {
	logAttrs := []any{slog.Any("error", err)}
	logger := logging.From(r.Context())

	switch {
	case goerr.HasTag(err, model.ErrTagInvalidRequest):
		logger.Warn("Bad Request", logAttrs...)
		http.Error(w, err.Error(), http.StatusBadRequest)

	default:
		hub := sentry.CurrentHub().Clone()
		hub.ConfigureScope(func(scope *sentry.Scope) {
			for k, v := range goerr.Values(err) {
				scope.SetExtra(k, v)
			}
		})
		evID := hub.CaptureException(err)
		logAttrs = append(logAttrs, slog.Any("sentry.id", evID))

		logger.Error("Request error", logAttrs...)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
