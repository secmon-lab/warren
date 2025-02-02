package server

import (
	"net/http"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

var (
	errBadRequest = goerr.NewTag("bad request")
)

func handleError(w http.ResponseWriter, r *http.Request, err error) {
	logger := logging.From(r.Context())

	logger.Error("Request error", "error", err)

	switch {
	case goerr.HasTag(err, errBadRequest):
		http.Error(w, err.Error(), http.StatusBadRequest)

	default:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
