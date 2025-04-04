package http

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/interfaces"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/utils/logging"
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

func alertSNSHandler(uc interfaces.UseCase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := logging.From(ctx)

		body, err := io.ReadAll(r.Body)
		if err != nil {
			handleError(w, r, goerr.Wrap(err, "failed to read body"))
			return
		}

		var msg model.SNSMessage
		if err := json.Unmarshal(body, &msg); err != nil {
			handleError(w, r, goerr.Wrap(err, "failed to decode message",
				goerr.T(model.ErrTagInvalidRequest),
				goerr.V("body", string(body)),
			))
			return
		}

		// Convert SNS message to Alert
		var alertData any
		if err := json.Unmarshal([]byte(msg.Message), &alertData); err != nil {
			handleError(w, r, goerr.Wrap(err, "failed to marshal message",
				goerr.T(model.ErrTagInvalidRequest),
				goerr.V("msg", msg),
			))
			return
		}

		schema := chi.URLParam(r, "schema")

		// Handle alert
		alerts, err := uc.HandleAlertWithAuth(ctx, schema, alertData)
		if err != nil {
			handleError(w, r, goerr.Wrap(err, "failed to handle alert"))
			return
		}

		logger.Info("alert handled", "alerts", alerts)
		w.WriteHeader(http.StatusOK)
	}
}
