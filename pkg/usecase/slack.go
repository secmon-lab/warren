package usecase

import (
	"context"
	"fmt"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/service"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/errs"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/thread"
	"github.com/slack-go/slack"
)

func (uc *UseCases) HandleSlackAppMention(ctx context.Context, user model.SlackUser, mention model.SlackMention, slackThread model.SlackThread) error {
	logger := logging.From(ctx)
	logger.Debug("slack app mention event", "mention", mention, "slack_thread", slackThread)

	// Nothing to do
	if !uc.slackService.IsBotUser(mention.UserID) {
		return nil
	}

	alerts, err := uc.repository.GetAlertsBySlackThread(ctx, slackThread)
	if err != nil {
		return goerr.Wrap(err, "failed to get alert by slack thread")
	}

	th := uc.slackService.NewThread(slackThread)
	ctx = thread.WithReplyFunc(ctx, th.Reply)

	if len(mention.Args) == 0 {
		th.Reply(ctx, "⏸️ No action specified")
		return nil
	}

	arguments := append([]string{"warren"}, mention.Args...)
	uc.dispatchSlackAction(ctx, func(ctx context.Context) error {
		return uc.RunCommand(ctx, arguments, alerts, th, &user)
	})

	return nil
}

func (uc *UseCases) dispatchSlackAction(ctx context.Context, action func(ctx context.Context) error) {
	newCtx := newBackgroundContext(ctx)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				errs.Handle(newCtx, goerr.New("panic", goerr.V("recover", r)))
			}
		}()

		if err := action(newCtx); err != nil {
			errs.Handle(newCtx, err)
		}
	}()
}

func (uc *UseCases) HandleSlackMessage(ctx context.Context, slackThread model.SlackThread, text string, user model.SlackUser, ts string) error {
	logger := logging.From(ctx)

	// Skip if the message is from the bot
	if uc.slackService.IsBotUser(user.ID) {
		return nil
	}

	alerts, err := uc.repository.GetAlertsBySlackThread(ctx, slackThread)
	if err != nil {
		return goerr.Wrap(err, "failed to get alert by slack thread")
	}
	if len(alerts) == 0 {
		logger.Info("alert not found", "thread", slackThread)
		return nil
	}

	var baseAlert *model.Alert
	for _, a := range alerts {
		if a.ParentID == "" {
			baseAlert = &a
		}
	}
	if baseAlert == nil {
		logger.Warn("base alert not found", "thread", slackThread)
		return nil
	}

	comment := model.AlertComment{
		AlertID:   baseAlert.ID,
		Comment:   text,
		Timestamp: ts,
		User:      user,
	}
	if err := uc.repository.InsertAlertComment(ctx, comment); err != nil {
		return goerr.Wrap(err, "failed to insert alert comment", goerr.V("comment", comment))
	}

	return nil
}

func (uc *UseCases) HandleSlackInteractionViewSubmissionResolveAlert(ctx context.Context, user model.SlackUser, metadata string, values map[string]map[string]slack.BlockAction) error {
	logger := logging.From(ctx)

	alertID := model.AlertID(metadata)
	alert, err := uc.repository.GetAlert(ctx, alertID)
	if err != nil {
		thread.Reply(ctx, "💥 Failed to get alert\n> "+err.Error())
		return goerr.Wrap(err, "failed to get alert")
	}
	if alert == nil {
		thread.Reply(ctx, "💥 Alert not found")
		return nil
	}

	if err := uc.handleSlackInteractionViewSubmissionResolve(ctx, user, values, []model.Alert{*alert}); err != nil {
		logger.Error("failed to resolve alert", "error", err)
		return err
	}

	return nil
}

func (uc *UseCases) HandleSlackInteractionViewSubmissionResolveList(ctx context.Context, user model.SlackUser, metadata string, values map[string]map[string]slack.BlockAction) error {
	listID := model.AlertListID(metadata)
	list, err := uc.repository.GetAlertList(ctx, listID)
	if err != nil {
		thread.Reply(ctx, "💥 Failed to get alert list\n> "+err.Error())
		return goerr.Wrap(err, "failed to get alert list")
	}

	alerts, err := uc.repository.BatchGetAlerts(ctx, list.AlertIDs)
	if err != nil {
		thread.Reply(ctx, "💥 Failed to get alerts\n> "+err.Error())
		return goerr.Wrap(err, "failed to get alerts")
	}

	if err := uc.handleSlackInteractionViewSubmissionResolve(ctx, user, values, alerts); err != nil {
		thread.Reply(ctx, "💥 Failed to resolve alerts of list\n> "+err.Error())
		return goerr.Wrap(err, "failed to resolve alerts")
	}

	return nil
}

func (uc *UseCases) handleSlackInteractionViewSubmissionResolve(ctx context.Context, user model.SlackUser, values map[string]map[string]slack.BlockAction, alerts []model.Alert) error {
	thread.Reply(ctx, fmt.Sprintf("⏳ Resolving %d alerts...", len(alerts)))

	var (
		conclusion model.AlertConclusion
		reason     string
	)
	if conclusionBlock, ok := values[model.SlackBlockIDConclusion.String()]; ok {
		if conclusionAction, ok := conclusionBlock[model.SlackActionIDConclusion.String()]; ok {
			conclusion = model.AlertConclusion(conclusionAction.SelectedOption.Value)
		}
	}
	if commentBlock, ok := values[model.SlackBlockIDComment.String()]; ok {
		if commentAction, ok := commentBlock[model.SlackActionIDComment.String()]; ok {
			reason = commentAction.Value
		}
	}

	if err := conclusion.Validate(); err != nil {
		return goerr.Wrap(err, "invalid conclusion", goerr.V("conclusion", conclusion))
	}

	now := clock.Now(ctx)
	for _, alert := range alerts {
		alert.Status = model.AlertStatusResolved
		alert.ResolvedAt = &now
		alert.Conclusion = conclusion
		alert.Reason = reason
		if alert.Assignee == nil {
			alert.Assignee = &user
		}

		if err := uc.repository.PutAlert(ctx, alert); err != nil {
			return goerr.Wrap(err, "failed to put alert")
		}

		th := uc.slackService.NewThread(*alert.SlackThread)
		ctx = thread.WithReplyFunc(ctx, th.Reply)
		th.Reply(ctx, "Alert resolved by <@"+user.ID+">")

		if alert.ParentID == "" {
			if err := th.UpdateAlert(ctx, alert); err != nil {
				return goerr.Wrap(err, "failed to update slack thread")
			}
		}
	}

	thread.Reply(ctx, fmt.Sprintf("✅️ Resolved %d alerts", len(alerts)))

	return nil
}

func (uc *UseCases) HandleSlackInteractionBlockActions(ctx context.Context, user model.SlackUser, slackThread model.SlackThread, actionID model.SlackActionID, value, triggerID string) error {
	logger := logging.From(ctx)

	th := uc.slackService.NewThread(slackThread)
	ctx = thread.WithReplyFunc(ctx, th.Reply)

	switch actionID {
	case model.SlackActionIDAck:
		alert, err := uc.repository.GetAlert(ctx, model.AlertID(value))
		if err != nil {
			return goerr.Wrap(err, "failed to get alert")
		} else if alert == nil {
			logger.Error("alert not found", "alert_id", value)
			return nil
		}

		alert.Assignee = &user
		alert.Status = model.AlertStatusAcknowledged
		if err := uc.repository.PutAlert(ctx, *alert); err != nil {
			return goerr.Wrap(err, "failed to put alert")
		}

		if alert.SlackThread != nil {
			thread := uc.slackService.NewThread(*alert.SlackThread)
			thread.Reply(ctx, "Alert acknowledged by <@"+user.ID+">")

			if err := thread.UpdateAlert(ctx, *alert); err != nil {
				return goerr.Wrap(err, "failed to update slack thread")
			}
		} else {
			logger.Warn("slack thread not found", "alert_id", alert.ID)
		}

	case model.SlackActionIDResolve:
		alert, err := uc.repository.GetAlert(ctx, model.AlertID(value))
		if err != nil {
			return goerr.Wrap(err, "failed to get alert")
		} else if alert == nil {
			logger.Error("alert not found", "alert_id", value)
			return nil
		}

		if svc, ok := uc.slackService.(*service.Slack); ok {
			if err := svc.ShowResolveAlertModal(ctx, *alert, triggerID); err != nil {
				return goerr.Wrap(err, "failed to show resolve alert modal")
			}
		} else {
			logger.Warn("slack service is not available")
		}

	case model.SlackActionIDInspect:
		alert, err := uc.repository.GetAlert(ctx, model.AlertID(value))
		if err != nil {
			return goerr.Wrap(err, "failed to get alert")
		} else if alert == nil {
			logger.Error("alert not found", "alert_id", value)
			return nil
		}

		if err := uc.RunWorkflow(ctx, *alert); err != nil {
			return err
		}

	case model.SlackActionIDIgnoreList:
		return uc.RunCommand(ctx, []string{"warren", "ignore", value}, nil, th, &user)

	case model.SlackActionIDResolveList:
		return uc.RunCommand(ctx, []string{"warren", "resolve", value}, nil, th, &user)

	case model.SlackActionIDCreatePR:
		th.Reply(ctx, "✏️ Creating pull request...")

		diffID := model.PolicyDiffID(value)
		diff, err := uc.repository.GetPolicyDiff(ctx, diffID)
		if err != nil {
			thread.Reply(ctx, "💥 Failed to get policy diff\n> "+err.Error())
			return goerr.Wrap(err, "failed to get policy diff")
		} else if diff == nil {
			thread.Reply(ctx, "💥 Policy diff not found")
			return nil
		}

		if uc.gitHubApp == nil {
			thread.Reply(ctx, "💥 GitHub is not enabled")
			return nil
		}

		prURL, err := uc.gitHubApp.CreatePullRequest(ctx, diff)
		if err != nil {
			thread.Reply(ctx, "💥 Failed to create pull request\n> "+err.Error())
			return err
		}

		thread.Reply(ctx, fmt.Sprintf("✅️ Created: <%s|%s>", prURL.String(), diff.Title))
	}

	return nil
}
