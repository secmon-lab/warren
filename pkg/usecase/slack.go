package usecase

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/lang"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

func (uc *UseCases) HandleSlackAppMention(ctx context.Context, event *slackevents.AppMentionEvent) error {
	logger := logging.From(ctx)
	logger.Debug("slack app mention event", "event", event)

	thread := model.SlackThread{
		ChannelID: event.Channel,
		ThreadID:  event.ThreadTimeStamp,
	}
	alert, err := uc.repository.GetAlertBySlackThread(ctx, thread)
	if err != nil {
		return goerr.Wrap(err, "failed to get alert by slack thread")
	}
	if alert == nil {
		logger.Info("alert not found", "thread", thread)
		return nil
	}

	args := uc.slackService.TrimMention(event.Text)
	if args == "" {
		logger.Warn("slack app mention event is empty", "event", event)
		return nil
	}

	return nil
}

func (uc *UseCases) HandleSlackMessage(ctx context.Context, event *slackevents.MessageEvent) error {
	logger := logging.From(ctx)
	logger.Debug("slack message event", "event", event)

	if event.ThreadTimeStamp == "" {
		return nil
	}

	thread := model.SlackThread{
		ChannelID: event.Channel,
		ThreadID:  event.ThreadTimeStamp,
	}
	alert, err := uc.repository.GetAlertBySlackThread(ctx, thread)
	if err != nil {
		return goerr.Wrap(err, "failed to get alert by slack thread")
	}
	if alert == nil {
		logger.Info("alert not found", "thread", thread)
		return nil
	}

	comment := model.AlertComment{
		AlertID:   alert.ID,
		Comment:   event.Text,
		Timestamp: event.EventTimeStamp,
		UserID:    event.User,
	}
	if err := uc.repository.InsertAlertComment(ctx, comment); err != nil {
		return goerr.Wrap(err, "failed to insert alert comment", goerr.V("comment", comment))
	}

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
		alert.Status = model.AlertStatusAcknowledged
		if err := uc.repository.PutAlert(ctx, *alert); err != nil {
			return goerr.Wrap(err, "failed to put alert")
		}

		thread := uc.slackService.NewThread(*alert)
		thread.Reply(ctx, "Alert acknowledged by <@"+interaction.User.ID+">")

		if err := thread.UpdateAlert(ctx, *alert); err != nil {
			return goerr.Wrap(err, "failed to update slack thread")
		}

	case "close":
		now := clock.Now(ctx)
		alert.Status = model.AlertStatusClosed
		alert.ClosedAt = &now

		if alert.Assignee == nil {
			alert.Assignee = &model.SlackUser{
				ID:   interaction.User.ID,
				Name: interaction.User.Name,
			}
		}

		if err := uc.repository.PutAlert(ctx, *alert); err != nil {
			return goerr.Wrap(err, "failed to put alert")
		}

		thread := uc.slackService.NewThread(*alert)
		thread.Reply(ctx, "Alert closed by <@"+interaction.User.ID+">")

		if err := thread.UpdateAlert(ctx, *alert); err != nil {
			return goerr.Wrap(err, "failed to update slack thread")
		}

	case "inspect":
		go func() {
			defer func() {
				if r := recover(); r != nil {
					logger.Error("panic in workflow process", "error", r)
				}
			}()

			newCtx := context.Background()
			newCtx = lang.With(newCtx, lang.From(ctx))
			newCtx = logging.With(newCtx, logging.From(ctx))

			if err := uc.RunWorkflow(newCtx, *alert); err != nil {
				logger.Error("failed to run workflow", "error", err)
			}
		}()
	}

	return nil
}
