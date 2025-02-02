package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/usecase"
	"github.com/slack-go/slack"
)

func slackEventHandler(uc *usecase.UseCases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			handleError(w, r, goerr.Wrap(err, "failed to read request body"))
			return
		}

		if err := uc.VerifySlackRequest(r.Context(), r.Header, body); err != nil {
			handleError(w, r, goerr.Wrap(err, "failed to verify slack request",
				goerr.T(errBadRequest),
				goerr.V("body", string(body)),
			))
			return
		}

		var event slack.Event
		if err := json.Unmarshal(body, &event); err != nil {
			handleError(w, r, goerr.Wrap(err, "failed to unmarshal slack event",
				goerr.T(errBadRequest),
				goerr.V("body", string(body)),
			))
			return
		}

		if err := uc.HandleSlackEvent(r.Context(), event); err != nil {
			handleError(w, r, goerr.Wrap(err, "failed to handle slack event",
				goerr.T(errBadRequest),
				goerr.V("body", string(body)),
			))
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func slackInteractionHandler(uc *usecase.UseCases) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			handleError(w, r, goerr.Wrap(err, "failed to read request body"))
			return
		}

		if err := uc.VerifySlackRequest(r.Context(), r.Header, body); err != nil {
			handleError(w, r, goerr.Wrap(err, "failed to verify slack request",
				goerr.T(errBadRequest),
				goerr.V("body", string(body)),
			))
			return
		}

		r.Body = io.NopCloser(bytes.NewReader(body))
		payload := r.FormValue("payload")
		if payload == "" {
			handleError(w, r, goerr.New("payload is required",
				goerr.V("body", string(body)),
				goerr.T(errBadRequest)),
			)
			return
		}

		var interaction slack.InteractionCallback
		if err := json.Unmarshal([]byte(payload), &interaction); err != nil {
			handleError(w, r, goerr.Wrap(err, "failed to unmarshal slack interaction",
				goerr.T(errBadRequest),
				goerr.V("payload", payload),
			))
			return
		}

		if err := uc.HandleSlackInteraction(r.Context(), interaction); err != nil {
			handleError(w, r, goerr.Wrap(err, "failed to handle slack interaction"))
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
