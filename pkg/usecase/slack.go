package usecase

import (
	"context"
	"fmt"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/source"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/logic"
	slack_svc "github.com/secmon-lab/warren/pkg/service/slack"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

// HandleSlackAppMention handles a slack app mention event. It will dispatch a slack action to the alert.
func (uc *UseCases) HandleSlackAppMention(ctx context.Context, user slack.User, mention slack.Mention, thread slack.Thread) error {
	logger := logging.From(ctx)
	logger.Debug("slack app mention event", "mention", mention, "slack_thread", thread)

	notifyThread := uc.slackService.NewThread(thread)
	ctx = msg.With(ctx, notifyThread.Reply, notifyThread.NewStateFunc)

	// Nothing to do
	if !uc.slackService.IsBotUser(mention.UserID) {
		return nil
	}

	alert, err := uc.repository.GetAlertByThread(ctx, thread)
	if err != nil {
		return goerr.Wrap(err, "failed to get alert by slack thread")
	}
	session, err := uc.repository.GetSessionByThread(ctx, thread)
	if err != nil {
		return goerr.Wrap(err, "failed to get session by slack thread")
	}

	if len(mention.Args) == 0 {
		msg.Reply(ctx, "⏸️ No action specified")
		return nil
	}

	arguments := append([]string{"warren"}, mention.Args...)
	uc.dispatchSlackAction(ctx, func(ctx context.Context) error {
		// TODO: Implement

		// If alert is not fo
		if alert != nil {
			logger.Info("alert found", "alert", alert)
		}
		if session != nil {
			logger.Info("session found", "session", session)
		}

		logger.Info("dispatch slack action", "arguments", arguments)
		return nil
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

// HandleSlackMessage handles a message from a slack user. It saves the message as an alert comment if the message is in the Alert thread.
func (uc *UseCases) HandleSlackMessage(ctx context.Context, thread slack.Thread, text string, user slack.User, ts string) error {
	logger := logging.From(ctx)
	th := uc.slackService.NewThread(thread)
	ctx = msg.With(ctx, th.Reply, th.NewStateFunc)

	// Skip if the message is from the bot
	if uc.slackService.IsBotUser(user.ID) {
		return nil
	}

	baseAlert, err := uc.repository.GetAlertByThread(ctx, thread)
	if err != nil {
		return goerr.Wrap(err, "failed to get alert by slack thread")
	}
	if baseAlert == nil {
		logger.Info("alert not found", "thread", thread)
		return nil
	}

	comment := alert.AlertComment{
		AlertID:   baseAlert.ID,
		Comment:   text,
		Timestamp: ts,
		User:      user,
	}
	if err := uc.repository.PutAlertComment(ctx, comment); err != nil {
		msg.Reply(ctx, "💥 Failed to insert alert comment\n> "+err.Error())
		return goerr.Wrap(err, "failed to insert alert comment", goerr.V("comment", comment))
	}

	return nil
}

// HandleSlackInteractionViewSubmissionResolveAlert handles a slack interaction view submission for "resolving an alert".
func (uc *UseCases) HandleSlackInteractionViewSubmissionResolveAlert(ctx context.Context, user slack.User, metadata string, values map[string]map[string]slack.BlockAction) error {
	logger := logging.From(ctx)
	logger.Debug("resolving alert",
		"user", user,
		"metadata", metadata,
		"values", values,
	)

	alertID := types.AlertID(metadata)
	target, err := uc.repository.GetAlert(ctx, alertID)
	if err != nil {
		msg.Reply(ctx, "💥 Failed to get alert\n> "+err.Error())
		return goerr.Wrap(err, "failed to get alert")
	}
	if target == nil {
		msg.Reply(ctx, "💥 Alert not found")
		return nil
	}

	if target.SlackThread != nil {
		st := uc.slackService.NewThread(*target.SlackThread)
		ctx = msg.With(ctx, st.Reply, st.NewStateFunc)
	}

	if err := uc.handleSlackInteractionViewSubmissionResolve(ctx, user, values, alert.Alerts{target}); err != nil {
		msg.Reply(ctx, "💥 Failed to resolve alert\n> "+err.Error())
		logger.Error("failed to resolve alert", "error", err)
		return err
	}

	return nil
}

// HandleSlackInteractionViewSubmissionResolveList handles a slack interaction view submission for "resolving an alert list".
func (uc *UseCases) HandleSlackInteractionViewSubmissionResolveList(ctx context.Context, user slack.User, metadata string, values map[string]map[string]slack.BlockAction) error {
	logger := logging.From(ctx)
	logger.Debug("resolving alert list",
		"user", user,
		"metadata", metadata,
		"values", values,
	)

	listID := types.AlertListID(metadata)
	list, err := uc.repository.GetAlertList(ctx, listID)
	if err != nil {
		return goerr.Wrap(err, "failed to get alert list", goerr.V("list_id", listID))
	}
	if list == nil {
		return goerr.Wrap(err, "alert list not found", goerr.V("list_id", listID))
	}

	if list.SlackThread != nil {
		st := uc.slackService.NewThread(*list.SlackThread)
		ctx = msg.With(ctx, st.Reply, st.NewStateFunc)
	}

	alerts, err := uc.repository.BatchGetAlerts(ctx, list.AlertIDs)
	if err != nil {
		msg.Reply(ctx, "💥 Failed to get alerts\n> "+err.Error())
		return goerr.Wrap(err, "failed to get alerts")
	}

	if err := uc.handleSlackInteractionViewSubmissionResolve(ctx, user, values, alerts); err != nil {
		msg.Reply(ctx, "💥 Failed to resolve alerts of list\n> "+err.Error())
		return goerr.Wrap(err, "failed to resolve alerts")
	}

	return nil
}

func (uc *UseCases) handleSlackInteractionViewSubmissionResolve(ctx context.Context, user slack.User, values map[string]map[string]slack.BlockAction, alerts alert.Alerts) error {
	logger := logging.From(ctx)
	ctx = msg.NewState(ctx, fmt.Sprintf("⏳ Resolving %d alerts...", len(alerts)))
	logger.Info("resolving alerts", "alerts", alerts)

	var (
		conclusion types.AlertConclusion
		reason     string
	)
	if conclusionBlock, ok := values[slack.SlackBlockIDConclusion.String()]; ok {
		if conclusionAction, ok := conclusionBlock[slack.SlackActionIDConclusion.String()]; ok {
			conclusion = types.AlertConclusion(conclusionAction.SelectedOption.Value)
		}
	}
	if commentBlock, ok := values[slack.SlackBlockIDComment.String()]; ok {
		if commentAction, ok := commentBlock[slack.SlackActionIDComment.String()]; ok {
			reason = commentAction.Value
		}
	}

	if err := conclusion.Validate(); err != nil {
		return goerr.Wrap(err, "invalid conclusion", goerr.V("conclusion", conclusion))
	}

	now := clock.Now(ctx)
	for i, alert := range alerts {
		if i > 0 && i%10 == 0 {
			ctx = msg.State(ctx, fmt.Sprintf("🏃 Resolving %d/%d alerts...", i+1, len(alerts)))
		}

		alert.Status = types.AlertStatusResolved
		alert.ResolvedAt = &now
		alert.Conclusion = conclusion
		alert.Reason = reason
		if alert.Assignee == nil {
			alert.Assignee = &user
		}

		if err := uc.repository.PutAlert(ctx, *alert); err != nil {
			return goerr.Wrap(err, "failed to put alert")
		}

		st := uc.slackService.NewThread(*alert.SlackThread)
		newCtx := msg.With(ctx, st.Reply, st.NewStateFunc)
		msg.Reply(newCtx, "Alert resolved by <@"+user.ID+">")

		logger.Info("alert resolved", "alert", alert)

		if err := st.UpdateAlert(newCtx, *alert); err != nil {
			return goerr.Wrap(err, "failed to update slack thread")
		}
	}

	ctx = msg.State(ctx, fmt.Sprintf("✅️ Resolved %d alerts", len(alerts)))

	return nil
}

// HandleSlackInteractionViewSubmissionIgnoreList handles a slack interaction view submission for "ignoring an alert list".
func (x *UseCases) HandleSlackInteractionViewSubmissionIgnoreList(ctx context.Context, metadata string, values map[string]map[string]slack.BlockAction) error {
	listID := types.AlertListID(metadata)

	var prompt string
	if promptBlock, ok := values[slack.SlackBlockIDIgnorePrompt.String()]; ok {
		if promptAction, ok := promptBlock[slack.SlackActionIDIgnorePrompt.String()]; ok {
			prompt = promptAction.Value
		}
	}

	list, err := x.repository.GetAlertList(ctx, listID)
	if err != nil {
		return goerr.Wrap(err, "failed to get alert list", goerr.V("list_id", listID))
	}
	if list == nil {
		return goerr.Wrap(err, "alert list not found", goerr.V("list_id", listID))
	}

	var slackSvc *slack_svc.ThreadService
	if list.SlackThread != nil {
		slackSvc = x.slackService.NewThread(*list.SlackThread)
	}

	src := source.AlertListID(listID)

	ssn := x.llmClient.StartChat()
	input := logic.GenerateIgnorePolicyInput{
		Repo:         x.repository,
		Source:       src,
		LLMFunc:      ssn.SendMessage,
		PolicyClient: x.policyClient,
		Prompt:       prompt,
		TestDataSet:  x.testDataSet,
	}
	newPolicyDiff, err := logic.GenerateIgnorePolicy(ctx, input)
	if err != nil {
		return err
	}

	if err := x.repository.PutPolicyDiff(ctx, newPolicyDiff); err != nil {
		return err
	}

	if slackSvc != nil {
		if err := slackSvc.PostPolicyDiff(ctx, newPolicyDiff); err != nil {
			return err
		}
	}

	return nil
}

// HandleSlackInteractionBlockActions handles a slack interaction block action.
func (uc *UseCases) HandleSlackInteractionBlockActions(ctx context.Context, user slack.User, slackThread slack.Thread, actionID slack.SlackActionID, value, triggerID string) error {
	logger := logging.From(ctx)

	st := uc.slackService.NewThread(slackThread)
	ctx = msg.With(ctx, st.Reply, st.NewStateFunc)

	switch actionID {
	case slack.SlackActionIDAck:
		alert, err := uc.repository.GetAlert(ctx, types.AlertID(value))
		if err != nil {
			return goerr.Wrap(err, "failed to get alert")
		} else if alert == nil {
			logger.Error("alert not found", "alert_id", value)
			return nil
		}

		alert.Assignee = &user
		alert.Status = types.AlertStatusAcknowledged
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

	case slack.SlackActionIDResolve:
		alert, err := uc.repository.GetAlert(ctx, types.AlertID(value))
		if err != nil {
			return goerr.Wrap(err, "failed to get alert")
		} else if alert == nil {
			logger.Error("alert not found", "alert_id", value)
			return nil
		}

		if err := uc.slackService.ShowResolveAlertModal(ctx, *alert, triggerID); err != nil {
			return goerr.Wrap(err, "failed to show resolve alert modal")
		}

	case slack.SlackActionIDInspect:
		alert, err := uc.repository.GetAlert(ctx, types.AlertID(value))
		if err != nil {
			return goerr.Wrap(err, "failed to get alert")
		} else if alert == nil {
			logger.Error("alert not found", "alert_id", value)
			return nil
		}

		if err := uc.RunWorkflow(ctx, *alert); err != nil {
			return err
		}

	case slack.SlackActionIDIgnoreList:
		return uc.RunCommand(ctx, []string{"warren", "ignore", value}, nil, th, &user)

	case slack.SlackActionIDResolveList:
		listID := alert.ListID(value)
		list, err := uc.repository.GetAlertList(ctx, listID)
		if err != nil {
			return goerr.Wrap(err, "failed to get alert list")
		} else if list == nil {
			thread.Reply(ctx, "💥 Alert list not found")
			return nil
		}

		if svc, ok := uc.slackService.(*service.Slack); ok {
			if err := svc.ShowResolveListModal(ctx, *list, triggerID); err != nil {
				return goerr.Wrap(err, "failed to show resolve list modal")
			}
		} else {
			logger.Warn("slack service is not available")
		}

	case slack.SlackActionIDCreatePR:
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
