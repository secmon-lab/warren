package server

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/interfaces"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/utils/lang"
)

func alertPubSubHandler(uc interfaces.UseCase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		schema := chi.URLParam(r, "schema")

		var msg struct {
			Message struct {
				Data []byte `json:"data"`
			} `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			handleError(w, r, goerr.Wrap(err, "failed to decode message",
				goerr.T(model.ErrTagInvalidRequest),
				goerr.V("body", r.Body),
			))
			return
		}
		r.Body = http.NoBody

		var alertData any
		if err := json.Unmarshal(msg.Message.Data, &alertData); err != nil {
			handleError(w, r, goerr.Wrap(err, "failed to decode message",
				goerr.T(model.ErrTagInvalidRequest),
				goerr.V("body", r.Body),
			))
			return
		}

		if _, err := uc.HandleAlertWithAuth(r.Context(), schema, alertData); err != nil {
			handleError(w, r, err)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func alertRawHandler(uc interfaces.UseCase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		schema := chi.URLParam(r, "schema")
		println("lang", lang.From(r.Context()).Name())

		if r.Header.Get("Content-Type") != "application/json" {
			handleError(w, r, goerr.New("invalid content type",
				goerr.T(model.ErrTagInvalidRequest),
			))
			return
		}

		var alertData any
		if err := json.NewDecoder(r.Body).Decode(&alertData); err != nil {
			handleError(w, r, goerr.Wrap(err, "failed to decode message",
				goerr.T(model.ErrTagInvalidRequest),
				goerr.V("body", r.Body),
			))
			return
		}

		if _, err := uc.HandleAlertWithAuth(r.Context(), schema, alertData); err != nil {
			handleError(w, r, err)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
