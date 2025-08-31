package http

import (
	"net/http"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

func handleError(w http.ResponseWriter, r *http.Request, err error) {
	logger := logging.From(r.Context())

	switch {
	case goerr.HasTag(err, errs.TagNotFound):
		logger.Warn("Not Found", "error", err)
		http.Error(w, err.Error(), http.StatusNotFound)

	case goerr.HasTag(err, errs.TagValidation), goerr.HasTag(err, errs.TagInvalidRequest):
		logger.Warn("Bad Request", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)

	case goerr.HasTag(err, errs.TagUnauthorized):
		logger.Warn("Unauthorized", "error", err)
		http.Error(w, err.Error(), http.StatusUnauthorized)

	case goerr.HasTag(err, errs.TagForbidden):
		logger.Warn("Forbidden", "error", err)
		http.Error(w, err.Error(), http.StatusForbidden)

	case goerr.HasTag(err, errs.TagConflict):
		logger.Warn("Conflict", "error", err)
		http.Error(w, err.Error(), http.StatusConflict)

	case goerr.HasTag(err, errs.TagRateLimit):
		logger.Warn("Rate Limit Exceeded", "error", err)
		http.Error(w, err.Error(), http.StatusTooManyRequests)

	case goerr.HasTag(err, errs.TagExternal):
		logger.Error("External Service Error", "error", err)
		http.Error(w, err.Error(), http.StatusBadGateway)

	case goerr.HasTag(err, errs.TagTimeout):
		logger.Error("Gateway Timeout", "error", err)
		http.Error(w, err.Error(), http.StatusGatewayTimeout)

	case goerr.HasTag(err, errs.TagDatabase), goerr.HasTag(err, errs.TagInternal):
		errs.Handle(r.Context(), err)
		http.Error(w, err.Error(), http.StatusInternalServerError)

	default:
		errs.Handle(r.Context(), err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
