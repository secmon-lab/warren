package server

import (
	"net/http"

	"github.com/getsentry/sentry-go"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

var (
	errBadRequest = goerr.NewTag("bad request")
)

func handleError(w http.ResponseWriter, r *http.Request, err error) {
	logger := logging.From(r.Context())

	hub := sentry.CurrentHub().Clone()
	hub.ConfigureScope(func(scope *sentry.Scope) {
		for k, v := range goerr.Values(err) {
			scope.SetExtra(k, v)
		}
	})
	evID := hub.CaptureException(err)

	logger.Error("Request error", "error", err, "sentry.id", evID)

	switch {
	case goerr.HasTag(err, errBadRequest):
		http.Error(w, err.Error(), http.StatusBadRequest)

	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
