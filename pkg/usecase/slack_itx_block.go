package usecase

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

// HandleSlackInteractionBlockActions handles a slack interaction block action.
func (uc *UseCases) HandleSlackInteractionBlockActions(ctx context.Context, user slack.User, slackThread slack.Thread, actionID slack.ActionID, value, triggerID string) error {
	logger := logging.From(ctx)

	threadSvc := uc.slackService.NewThread(slackThread)
	ctx = msg.With(ctx, threadSvc.Reply, threadSvc.NewStateFunc)

	switch actionID {
	case slack.ActionIDAckAlert:
		return uc.ackAlert(ctx, user, slackThread, types.AlertID(value))

	case slack.ActionIDAckList:
		return uc.ackList(ctx, user, slackThread, types.AlertListID(value))

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
	}

	return nil
}

func (uc *UseCases) ackAlert(ctx context.Context, user slack.User, slackThread slack.Thread, targetAlertID types.AlertID) error {
	logger := logging.From(ctx)
	st := uc.slackService.NewThread(slackThread)

	targetAlert, err := uc.repository.GetAlert(ctx, targetAlertID)
	if err != nil {
		return goerr.Wrap(err, "failed to get alert")
	} else if targetAlert == nil {
		logger.Error("alert not found", "alert_id", targetAlertID)
		return nil
	}

	newTicket := ticket.New(ctx, []types.AlertID{targetAlert.ID}, &slackThread)
	newTicket.Assignee = &user
	newTicket.Status = types.TicketStatusAcknowledged

	if err := newTicket.FillMetadata(ctx, uc.llmClient, uc.repository); err != nil {
		return goerr.Wrap(err, "failed to fill ticket metadata")
	}

	ts, err := st.PostTicket(ctx, newTicket, alert.Alerts{targetAlert})
	if err != nil {
		return goerr.Wrap(err, "failed to post ticket")
	}
	newTicket.SlackMessageID = ts
	targetAlert.TicketID = newTicket.ID

	if err := uc.repository.PutTicket(ctx, newTicket); err != nil {
		return goerr.Wrap(err, "failed to put ticket")
	}
	if err := uc.repository.PutAlert(ctx, *targetAlert); err != nil {
		return goerr.Wrap(err, "failed to put alert")
	}

	if err := st.UpdateAlert(ctx, *targetAlert); err != nil {
		return goerr.Wrap(err, "failed to update slack thread")
	}

	msg.Trace(ctx, "🎫 Alert acknowledged by <@%s>", user.ID)
	return nil
}

func (uc *UseCases) ackList(ctx context.Context, user slack.User, slackThread slack.Thread, targetListID types.AlertListID) error {
	logger := logging.From(ctx)
	st := uc.slackService.NewThread(slackThread)

	list, err := uc.repository.GetAlertList(ctx, targetListID)
	if err != nil {
		return goerr.Wrap(err, "failed to get alert list")
	}
	if list == nil {
		logger.Error("alert list not found", "list_id", targetListID)
		return nil
	}

	alertIDs := make([]types.AlertID, len(list.Alerts))
	for i, alert := range list.Alerts {
		alertIDs[i] = alert.ID
	}

	newTicket := ticket.New(ctx, alertIDs, &slackThread)
	newTicket.Assignee = &user
	newTicket.Status = types.TicketStatusAcknowledged

	if err := newTicket.FillMetadata(ctx, uc.llmClient, uc.repository); err != nil {
		return goerr.Wrap(err, "failed to fill ticket metadata")
	}

	ts, err := st.PostTicket(ctx, newTicket, list.Alerts)
	if err != nil {
		return goerr.Wrap(err, "failed to post ticket")
	}
	newTicket.SlackMessageID = ts

	if err := uc.repository.BatchBindAlertsToTicket(ctx, alertIDs, newTicket.ID); err != nil {
		return goerr.Wrap(err, "failed to bind alerts to ticket")
	}

	for _, alert := range list.Alerts {
		if err := st.UpdateAlert(ctx, *alert); err != nil {
			return goerr.Wrap(err, "failed to update alert")
		}
	}

	msg.Trace(ctx, "🎫 Alert list acknowledged by <@%s>", user.ID)
	return nil
}
