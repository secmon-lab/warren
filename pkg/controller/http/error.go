package http

import (
	"net/http"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/utils/errs"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

func handleError(w http.ResponseWriter, r *http.Request, err error) {
	logger := logging.From(r.Context())

	switch {
	case goerr.HasTag(err, model.ErrTagInvalidRequest):
		logger.Warn("Bad Request", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)

	default:
		errs.Handle(r.Context(), err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
