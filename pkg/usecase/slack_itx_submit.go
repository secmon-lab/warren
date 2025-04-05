package usecase

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/source"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/policy"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

// HandleSlackInteractionViewSubmissionResolveAlert handles a slack interaction view submission for "resolving an alert".
func (uc *UseCases) HandleSlackInteractionViewSubmissionResolveAlert(ctx context.Context, user slack.User, metadata string, values slack.StateValue) error {
	logger := logging.From(ctx)
	logger.Debug("resolving alert",
		"user", user,
		"metadata", metadata,
		"values", values,
	)

	alertID := types.AlertID(metadata)
	target, err := uc.repository.GetAlert(ctx, alertID)
	if err != nil {
		msg.Trace(ctx, "💥 Failed to get alert\n> %s", err.Error())
		return goerr.Wrap(err, "failed to get alert")
	}
	if target == nil {
		msg.Notify(ctx, "💥 Alert not found")
		return nil
	}

	if target.SlackThread != nil {
		st := uc.slackService.NewThread(*target.SlackThread)
		ctx = msg.With(ctx, st.Reply, st.NewStateFunc)
	}

	if err := uc.handleSlackInteractionViewSubmissionResolve(ctx, user, values, alert.Alerts{target}); err != nil {
		msg.Trace(ctx, "💥 Failed to resolve alert\n> %s", err.Error())
		logger.Error("failed to resolve alert", "error", err)
		return err
	}

	return nil
}

// HandleSlackInteractionViewSubmissionResolveList handles a slack interaction view submission for "resolving an alert list".
func (uc *UseCases) HandleSlackInteractionViewSubmissionResolveList(ctx context.Context, user slack.User, metadata string, values slack.StateValue) error {
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
		msg.Trace(ctx, "💥 Failed to get alerts\n> %s", err.Error())
		return goerr.Wrap(err, "failed to get alerts")
	}

	if err := uc.handleSlackInteractionViewSubmissionResolve(ctx, user, values, alerts); err != nil {
		msg.Trace(ctx, "💥 Failed to resolve alerts of list\n> %s", err.Error())
		return goerr.Wrap(err, "failed to resolve alerts")
	}

	return nil
}

func (uc *UseCases) handleSlackInteractionViewSubmissionResolve(ctx context.Context, user slack.User, values slack.StateValue, alerts alert.Alerts) error {
	logger := logging.From(ctx)
	ctx = msg.NewTrace(ctx, "⏳ Resolving %d alerts...", len(alerts))
	logger.Info("resolving alerts", "alerts", alerts)

	var (
		conclusion types.AlertConclusion
		reason     string
	)
	if conclusionBlock, ok := values[slack.SlackBlockIDConclusion.String()]; ok {
		if conclusionAction, ok := conclusionBlock[slack.ActionIDConclusion.String()]; ok {
			conclusion = types.AlertConclusion(conclusionAction.SelectedOption.Value)
		}
	}
	if commentBlock, ok := values[slack.SlackBlockIDComment.String()]; ok {
		if commentAction, ok := commentBlock[slack.ActionIDComment.String()]; ok {
			reason = commentAction.Value
		}
	}

	if err := conclusion.Validate(); err != nil {
		return goerr.Wrap(err, "invalid conclusion", goerr.V("conclusion", conclusion))
	}

	now := clock.Now(ctx)
	for i, alert := range alerts {
		if i > 0 && i%25 == 0 {
			msg.Trace(ctx, "🏃 Resolving %d/%d alerts...", i+1, len(alerts))
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
		msg.Notify(newCtx, "Alert resolved by <@%s>", user.ID)

		logger.Info("alert resolved", "alert", alert)

		if err := st.UpdateAlert(newCtx, *alert); err != nil {
			return goerr.Wrap(err, "failed to update slack thread")
		}
	}

	msg.Trace(ctx, "✅️ Resolved %d alerts", len(alerts))

	return nil
}

// HandleSlackInteractionViewSubmissionIgnoreList handles a slack interaction view submission for "ignoring an alert list".
func (x *UseCases) HandleSlackInteractionViewSubmissionIgnoreList(ctx context.Context, metadata string, values slack.StateValue) error {
	listID := types.AlertListID(metadata)

	var prompt string
	if promptBlock, ok := values[slack.SlackBlockIDIgnorePrompt.String()]; ok {
		if promptAction, ok := promptBlock[slack.ActionIDIgnorePrompt.String()]; ok {
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

	src := source.AlertListID(listID)

	ssn := x.llmClient.StartChat()
	input := policy.GenerateIgnorePolicyInput{
		Repo:         x.repository,
		Source:       src,
		LLM:          ssn,
		PolicyClient: x.policyClient,
		Prompt:       prompt,
		TestDataSet:  x.testDataSet,
	}
	newPolicyDiff, err := policy.GenerateIgnorePolicy(ctx, input)
	if err != nil {
		return err
	}

	if err := x.repository.PutPolicyDiff(ctx, newPolicyDiff); err != nil {
		return err
	}

	if list.SlackThread != nil {
		slackSvc := x.slackService.NewThread(*list.SlackThread)
		if err := slackSvc.PostPolicyDiff(ctx, newPolicyDiff); err != nil {
			return err
		}
	}

	return nil
}
