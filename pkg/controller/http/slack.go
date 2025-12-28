package http

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/m-mizutani/goerr/v2"
	slack_ctrl "github.com/secmon-lab/warren/pkg/controller/slack"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

func slackEventHandler(ctrl *slack_ctrl.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			handleError(w, r, goerr.Wrap(err, "failed to read request body"))
			return
		}

		eventsAPIEvent, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionNoVerifyToken())
		if err != nil {
			handleError(w, r, goerr.Wrap(err, "failed to parse slack event",
				goerr.T(errutil.TagInvalidRequest),
				goerr.V("body", string(body)),
			))
			return
		}

		switch eventsAPIEvent.Type {
		case slackevents.URLVerification:
			var response *slackevents.ChallengeResponse
			err := json.Unmarshal([]byte(body), &response)
			if err != nil {
				handleError(w, r, goerr.Wrap(err, "failed to unmarshal slack challenge response",
					goerr.T(errutil.TagInvalidRequest),
					goerr.V("body", string(body)),
				))
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
				if err := ctrl.HandleSlackAppMention(r.Context(), &eventsAPIEvent, ev); err != nil {
					logging.From(r.Context()).Error("failed to handle app mention", "error", err)
				}

			case *slackevents.MessageEvent:
				if err := ctrl.HandleSlackMessage(r.Context(), &eventsAPIEvent, ev); err != nil {
					logging.From(r.Context()).Error("failed to handle message", "error", err)
				}

			default:
				logging.From(r.Context()).Warn("unknown event type", "event", ev, "body", string(body))
			}
		}
		w.WriteHeader(http.StatusOK)
	}
}

func slackInteractionHandler(slackCtrl *slack_ctrl.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		payload := r.FormValue("payload")
		if payload == "" {
			handleError(w, r, goerr.New("payload is required",
				goerr.T(errutil.TagInvalidRequest)),
			)
			return
		}

		var interaction slack.InteractionCallback
		if err := json.Unmarshal([]byte(payload), &interaction); err != nil {
			handleError(w, r, goerr.Wrap(err, "failed to unmarshal slack interaction",
				goerr.T(errutil.TagInvalidRequest),
				goerr.V("payload", payload),
			))
			return
		}

		if err := slackCtrl.HandleSlackInteraction(r.Context(), interaction); err != nil {
			handleError(w, r, err)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
