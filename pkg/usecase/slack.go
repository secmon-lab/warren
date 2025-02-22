package usecase

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/errs"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/thread"
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

	switch interaction.Type {
	case slack.InteractionTypeBlockActions:
		return uc.handleSlackInteractionBlockActions(ctx, interaction)
	case slack.InteractionTypeViewSubmission:
		return uc.handleSlackInteractionViewSubmission(ctx, interaction)
	}

	return nil
}

func (uc *UseCases) handleSlackInteractionViewSubmission(ctx context.Context, interaction slack.InteractionCallback) error {
	logger := logging.From(ctx)

	if interaction.View.CallbackID != "close_submit" {
		return nil
	}

	alertID := model.AlertID(interaction.View.PrivateMetadata)
	alert, err := uc.repository.GetAlert(ctx, alertID)
	if err != nil {
		return goerr.Wrap(err, "failed to get alert")
	}
	if alert == nil {
		logger.Error("alert not found", "alert_id", alertID)
		return nil
	}

	var (
		conclusion model.AlertConclusion
		comment    string
	)
	if conclusionBlock, ok := interaction.View.State.Values["conclusion"]; ok {
		if conclusionAction, ok := conclusionBlock["conclusion"]; ok {
			conclusion = model.AlertConclusion(conclusionAction.SelectedOption.Value)
		}
	}
	if commentBlock, ok := interaction.View.State.Values["comment"]; ok {
		if commentAction, ok := commentBlock["comment"]; ok {
			comment = commentAction.Value
		}
	}

	if err := conclusion.Validate(); err != nil {
		return goerr.Wrap(err, "invalid conclusion", goerr.V("conclusion", conclusion))
	}

	now := clock.Now(ctx)
	alert.Status = model.AlertStatusClosed
	alert.ClosedAt = &now
	alert.Conclusion = conclusion
	alert.Comment = comment
	if alert.Assignee == nil {
		alert.Assignee = &model.SlackUser{
			ID:   interaction.User.ID,
			Name: interaction.User.Name,
		}
	}

	if err := uc.repository.PutAlert(ctx, *alert); err != nil {
		return goerr.Wrap(err, "failed to put alert")
	}

	th := uc.slackService.NewThread(*alert)
	ctx = thread.WithReplyFunc(ctx, th.Reply)
	th.Reply(ctx, "Alert closed by <@"+interaction.User.ID+">")

	if err := th.UpdateAlert(ctx, *alert); err != nil {
		return goerr.Wrap(err, "failed to update slack thread")
	}

	newCtx := newBackgroundContext(ctx)
	genIgnorePolicy := func() {
		defer func() {
			if r := recover(); r != nil {
				errs.Handle(newCtx, goerr.New("panic", goerr.V("recover", r)))
			}
		}()

		newPolicy, err := uc.GenerateIgnorePolicy(newCtx, []model.Alert{*alert}, "")
		if err != nil {
			errs.Handle(newCtx, err)
		}

		diff := diffPolicy(uc.policyService.Sources(), newPolicy.Sources())
		if diff != "" {
			if err := th.AttachFile(newCtx, "New policy diff", "policy.diff", []byte(diff)); err != nil {
				errs.Handle(newCtx, err)
			}
		} else {
			th.Reply(newCtx, "No changes in ignore policy")
		}
	}

	if alert.Conclusion == model.AlertConclusionFalsePositive ||
		alert.Conclusion == model.AlertConclusionIntended ||
		alert.Conclusion == model.AlertConclusionUnaffected {
		go genIgnorePolicy()
	}

	return nil
}

func (uc *UseCases) handleSlackInteractionBlockActions(ctx context.Context, interaction slack.InteractionCallback) error {
	logger := logging.From(ctx)

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
		triggerID := interaction.TriggerID

		if err := uc.slackService.ShowCloseAlertModal(ctx, *alert, triggerID); err != nil {
			return goerr.Wrap(err, "failed to show close alert modal")
		}

	case "inspect":
		newCtx := newBackgroundContext(ctx)

		go func() {
			defer func() {
				if r := recover(); r != nil {
					errs.Handle(newCtx, goerr.New("panic", goerr.V("recover", r)))
				}
			}()

			if err := uc.RunWorkflow(newCtx, *alert); err != nil {
				errs.Handle(newCtx, err)
			}
		}()
	}

	return nil
}
