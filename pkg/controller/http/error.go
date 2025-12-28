package http

import (
	"net/http"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

func handleError(w http.ResponseWriter, r *http.Request, err error) {
	logger := logging.From(r.Context())

	switch {
	case goerr.HasTag(err, errutil.TagNotFound):
		logger.Warn("Not Found", "error", err)
		http.Error(w, err.Error(), http.StatusNotFound)

	case goerr.HasTag(err, errutil.TagValidation), goerr.HasTag(err, errutil.TagInvalidRequest):
		logger.Warn("Bad Request", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)

	case goerr.HasTag(err, errutil.TagUnauthorized):
		logger.Warn("Unauthorized", "error", err)
		http.Error(w, err.Error(), http.StatusUnauthorized)

	case goerr.HasTag(err, errutil.TagForbidden):
		logger.Warn("Forbidden", "error", err)
		http.Error(w, err.Error(), http.StatusForbidden)

	case goerr.HasTag(err, errutil.TagConflict):
		logger.Warn("Conflict", "error", err)
		http.Error(w, err.Error(), http.StatusConflict)

	case goerr.HasTag(err, errutil.TagRateLimit):
		logger.Warn("Rate Limit Exceeded", "error", err)
		http.Error(w, err.Error(), http.StatusTooManyRequests)

	case goerr.HasTag(err, errutil.TagExternal):
		logger.Error("External Service Error", "error", err)
		http.Error(w, err.Error(), http.StatusBadGateway)

	case goerr.HasTag(err, errutil.TagTimeout):
		logger.Error("Gateway Timeout", "error", err)
		http.Error(w, err.Error(), http.StatusGatewayTimeout)

	case goerr.HasTag(err, errutil.TagDatabase), goerr.HasTag(err, errutil.TagInternal):
		errutil.Handle(r.Context(), err)
		http.Error(w, err.Error(), http.StatusInternalServerError)

	default:
		errutil.Handle(r.Context(), err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
