package usecase

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

// HandleSlackInteractionBlockActions handles a slack interaction block action.
func (uc *UseCases) HandleSlackInteractionBlockActions(ctx context.Context, user slack.User, slackThread slack.Thread, actionID slack.ActionID, value, triggerID string) error {
	logger := logging.From(ctx)

	st := uc.slackService.NewThread(slackThread)
	ctx = msg.With(ctx, st.Reply, st.NewStateFunc)

	switch actionID {
	case slack.ActionIDAck:
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

		msg.Trace(ctx, "Alert acknowledged by <@%s>", user.ID)

		if err := st.UpdateAlert(ctx, *alert); err != nil {
			return goerr.Wrap(err, "failed to update slack thread")
		}

	case slack.ActionIDResolve:
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

	case slack.ActionIDInspect:
		alert, err := uc.repository.GetAlert(ctx, types.AlertID(value))
		if err != nil {
			return goerr.Wrap(err, "failed to get alert")
		} else if alert == nil {
			logger.Error("alert not found", "alert_id", value)
			return nil
		}

		// TODO: Implement

	case slack.ActionIDResolveList:
		listID := types.AlertListID(value)
		list, err := uc.repository.GetAlertList(ctx, listID)
		if err != nil {
			return goerr.Wrap(err, "failed to get alert list")
		} else if list == nil {
			msg.Trace(ctx, "💥 Alert list not found")
			return nil
		}

		if err := uc.slackService.ShowResolveListModal(ctx, *list, triggerID); err != nil {
			return goerr.Wrap(err, "failed to show resolve list modal")
		}

	case slack.ActionIDCreatePR:
		msg.Trace(ctx, "✏️ Creating pull request...")

		diffID := types.PolicyDiffID(value)
		diff, err := uc.repository.GetPolicyDiff(ctx, diffID)
		if err != nil {
			msg.Trace(ctx, "💥 Failed to get policy diff\n> %s", err.Error())
			return goerr.Wrap(err, "failed to get policy diff")
		} else if diff == nil {
			msg.Trace(ctx, "💥 Policy diff not found")
			return nil
		}

		if uc.githubApp == nil {
			msg.Trace(ctx, "💥 GitHub is not enabled")
			return nil
		}

		prURL, err := uc.githubApp.CreatePolicyDiffPullRequest(ctx, diff)
		if err != nil {
			msg.Trace(ctx, "💥 Failed to create pull request\n> %s", err.Error())
			return err
		}

		msg.Trace(ctx, "✅️ Created: <%s|%s>", prURL.String(), diff.Title)
	}

	return nil
}
