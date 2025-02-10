package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/usecase"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
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

		eventsAPIEvent, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionNoVerifyToken())
		if err != nil {
			handleError(w, r, goerr.Wrap(err, "failed to parse slack event",
				goerr.T(errBadRequest),
				goerr.V("body", string(body)),
			))
			return
		}

		switch eventsAPIEvent.Type {
		case slackevents.URLVerification:
			var response *slackevents.ChallengeResponse
			err := json.Unmarshal([]byte(body), &response)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text")
			if _, err := w.Write([]byte(response.Challenge)); err != nil {
				logging.From(r.Context()).Error("failed to write challenge response", "error", err)
			}

		case slackevents.CallbackEvent:
			innerEvent := eventsAPIEvent.InnerEvent

			switch ev := innerEvent.Data.(type) {
			case *slackevents.AppMentionEvent:
				if err := uc.HandleSlackAppMention(r.Context(), ev); err != nil {
					logging.From(r.Context()).Error("failed to handle app mention", "error", err)
				}

			case *slackevents.MessageEvent:
				if err := uc.HandleSlackMessage(r.Context(), ev); err != nil {
					logging.From(r.Context()).Error("failed to handle message", "error", err)
				}

			default:
				logging.From(r.Context()).Warn("unknown event type", "event", ev, "body", string(body))
			}
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
		println(string(payload))
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
