package http

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/domain/model/message"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/async"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

func alertPubSubHandler(uc useCase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		schema := chi.URLParam(r, "schema")

		rawBody, err := io.ReadAll(r.Body)
		if err != nil {
			handleError(w, r, goerr.Wrap(err, "failed to read body"))
			return
		}
		defer r.Body.Close()

		var msg message.PubSub
		if err := json.Unmarshal(rawBody, &msg); err != nil {
			handleError(w, r, goerr.Wrap(err, "failed to decode message",
				goerr.T(errs.TagInvalidRequest),
				goerr.V("body", rawBody),
			))
			return
		}

		var alertData any
		if err := json.Unmarshal([]byte(msg.Message.Data), &alertData); err != nil {
			handleError(w, r, goerr.Wrap(err, "failed to decode message",
				goerr.T(errs.TagInvalidRequest),
				goerr.V("body", msg.Message.Data),
			))
			return
		}

		// Check async mode
		if cfg := async.GetAsyncMode(r.Context()); cfg != nil && cfg.PubSub {
			// Return 200 immediately
			w.WriteHeader(http.StatusOK)

			// Process in background
			async.Dispatch(r.Context(), func(ctx context.Context) error {
				_, err := uc.HandleAlert(ctx, types.AlertSchema(schema), alertData)
				return err
			})
			return
		}

		// Synchronous processing
		if _, err := uc.HandleAlert(r.Context(), types.AlertSchema(schema), alertData); err != nil {
			handleError(w, r, err)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func alertRawHandler(uc useCase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		schema := chi.URLParam(r, "schema")

		if r.Header.Get("Content-Type") != "application/json" {
			handleError(w, r, goerr.New("invalid content type",
				goerr.T(errs.TagInvalidRequest),
			))
			return
		}

		var alertData any
		if err := json.NewDecoder(r.Body).Decode(&alertData); err != nil {
			handleError(w, r, goerr.Wrap(err, "failed to decode message",
				goerr.T(errs.TagInvalidRequest),
				goerr.V("body", r.Body),
			))
			return
		}

		// Check async mode
		if cfg := async.GetAsyncMode(r.Context()); cfg != nil && cfg.Raw {
			// Return 200 immediately
			w.WriteHeader(http.StatusOK)

			// Process in background
			async.Dispatch(r.Context(), func(ctx context.Context) error {
				_, err := uc.HandleAlert(ctx, types.AlertSchema(schema), alertData)
				return err
			})
			return
		}

		// Synchronous processing
		if _, err := uc.HandleAlert(r.Context(), types.AlertSchema(schema), alertData); err != nil {
			handleError(w, r, err)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func alertSNSHandler(uc useCase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := logging.From(ctx)

		body, err := io.ReadAll(r.Body)
		if err != nil {
			handleError(w, r, goerr.Wrap(err, "failed to read body"))
			return
		}

		var msg message.SNS
		if err := json.Unmarshal(body, &msg); err != nil {
			handleError(w, r, goerr.Wrap(err, "failed to decode message",
				goerr.T(errs.TagInvalidRequest),
				goerr.V("body", string(body)),
			))
			return
		}

		// Convert SNS message to Alert
		var alertData any
		if err := json.Unmarshal([]byte(msg.Message), &alertData); err != nil {
			handleError(w, r, goerr.Wrap(err, "failed to marshal message",
				goerr.T(errs.TagInvalidRequest),
				goerr.V("msg", msg),
			))
			return
		}

		schema := chi.URLParam(r, "schema")

		// Check async mode
		if cfg := async.GetAsyncMode(r.Context()); cfg != nil && cfg.SNS {
			// Return 200 immediately
			w.WriteHeader(http.StatusOK)

			// Process in background
			async.Dispatch(r.Context(), func(ctx context.Context) error {
				alerts, err := uc.HandleAlert(ctx, types.AlertSchema(schema), alertData)
				if err != nil {
					return err
				}
				logging.From(ctx).Info("alert handled", "alerts", alerts)
				return nil
			})
			return
		}

		// Synchronous processing
		alerts, err := uc.HandleAlert(ctx, types.AlertSchema(schema), alertData)
		if err != nil {
			handleError(w, r, goerr.Wrap(err, "failed to handle alert"))
			return
		}

		logger.Info("alert handled", "alerts", alerts)
		w.WriteHeader(http.StatusOK)
	}
}
