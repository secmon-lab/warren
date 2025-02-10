package usecase

import (
	"context"
	"net/http"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

func (uc *UseCases) VerifySlackRequest(ctx context.Context, header http.Header, body []byte) error {
	return uc.slackService.VerifyRequest(header, body)
}

func (uc *UseCases) HandleSlackAppMention(ctx context.Context, event *slackevents.AppMentionEvent) error {
	logger := logging.From(ctx)
	logger.Info("slack app mention event", "event", event)
	return nil
}

func (uc *UseCases) HandleSlackMessage(ctx context.Context, event *slackevents.MessageEvent) error {
	logger := logging.From(ctx)
	logger.Info("slack message event", "event", event)
	return nil
}

func (uc *UseCases) HandleSlackInteraction(ctx context.Context, interaction slack.InteractionCallback) error {
	logger := logging.From(ctx)
	logger.Info("slack interaction event", "event", interaction)

	if interaction.Type != slack.InteractionTypeBlockActions {
		logger.Warn("slack interaction event is not block actions", "event", interaction)
		return nil
	}

	action := interaction.ActionCallback.BlockActions[0]

	alertID := model.AlertID(action.Value)
	alert, err := uc.repository.GetAlert(ctx, alertID)
	if err != nil {
		return goerr.Wrap(err, "failed to get alert")
	}
	if alert == nil {
		logger.Error("alert not found", "alert_id", alertID)
		return nil
	}

	switch action.ActionID {
	case "ack":
		alert.Assignee = &model.SlackUser{
			ID:   interaction.User.ID,
			Name: interaction.User.Name,
		}

		if err := uc.repository.PutAlert(ctx, *alert); err != nil {
			return goerr.Wrap(err, "failed to put alert")
		}

		thread := uc.slackService.NewThread(*alert)
		if err := thread.Reply(ctx, "Alert acknowledged by "+interaction.User.Name); err != nil {
			return goerr.Wrap(err, "failed to reply to slack")
		}

	case "investigate":
		go func() {
			if err := uc.RunWorkflow(ctx, *alert); err != nil {
				logger.Error("failed to run workflow", "error", err)
			}
		}()
	}
	return nil
}
