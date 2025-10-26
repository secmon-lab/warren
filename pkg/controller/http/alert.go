package http

import (
	"context"
	"encoding/json"
	"io"
	"mime"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/domain/model/message"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/async"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/safe"
)

// hookEndpointType represents the type of alert hook endpoint
type hookEndpointType string

const (
	hookEndpointRaw    hookEndpointType = "raw"
	hookEndpointPubSub hookEndpointType = "pubsub"
	hookEndpointSNS    hookEndpointType = "sns"
)

// handleAlertWithAsync processes an alert either synchronously or asynchronously based on the configuration
func handleAlertWithAsync(w http.ResponseWriter, r *http.Request, uc useCase, schema string, alertData any, endpointType hookEndpointType) {
	cfg := async.GetAsyncMode(r.Context())

	// Check if async mode is enabled for this endpoint type
	isAsync := false
	if cfg != nil {
		switch endpointType {
		case hookEndpointRaw:
			isAsync = cfg.Raw
		case hookEndpointPubSub:
			isAsync = cfg.PubSub
		case hookEndpointSNS:
			isAsync = cfg.SNS
		}
	}

	if isAsync {
		// Return 200 immediately
		w.WriteHeader(http.StatusOK)

		// Process in background
		async.Dispatch(r.Context(), func(ctx context.Context) error {
			alerts, err := uc.HandleAlert(ctx, types.AlertSchema(schema), alertData)
			if err != nil {
				return err
			}
			if endpointType == hookEndpointSNS {
				logging.From(ctx).Info("alert handled", "alerts", alerts)
			}
			return nil
		})
		return
	}

	// Synchronous processing
	alerts, err := uc.HandleAlert(r.Context(), types.AlertSchema(schema), alertData)
	if err != nil {
		handleError(w, r, err)
		return
	}

	if endpointType == hookEndpointSNS {
		logging.From(r.Context()).Info("alert handled", "alerts", alerts)
	}
	w.WriteHeader(http.StatusOK)
}

func alertPubSubHandler(uc useCase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		schema := chi.URLParam(r, "schema")

		rawBody, err := io.ReadAll(r.Body)
		if err != nil {
			handleError(w, r, goerr.Wrap(err, "failed to read body"))
			return
		}
		defer safe.Close(r.Context(), r.Body)

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

		handleAlertWithAsync(w, r, uc, schema, alertData, hookEndpointPubSub)
	}
}

func alertRawHandler(uc useCase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		schema := chi.URLParam(r, "schema")

		contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || contentType != "application/json" {
			handleError(w, r, goerr.New("invalid content type",
				goerr.T(errs.TagInvalidRequest),
			))
			return
		}

		// If Content-Length is 0, treat as empty data
		var alertData any
		if r.ContentLength == 0 {
			alertData = nil
		} else {
			if err := json.NewDecoder(r.Body).Decode(&alertData); err != nil {
				handleError(w, r, goerr.Wrap(err, "failed to decode message",
					goerr.T(errs.TagInvalidRequest),
					goerr.V("body", r.Body),
				))
				return
			}
		}

		handleAlertWithAsync(w, r, uc, schema, alertData, hookEndpointRaw)
	}
}

func alertSNSHandler(uc useCase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		handleAlertWithAsync(w, r, uc, schema, alertData, hookEndpointSNS)
	}
}
