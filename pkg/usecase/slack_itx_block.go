package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

// HandleSlackInteractionBlockActions handles a slack interaction block action.
func (uc *UseCases) HandleSlackInteractionBlockActions(ctx context.Context, user slack.User, slackThread slack.Thread, actionID slack.ActionID, value, triggerID string) error {
	threadSvc := uc.slackService.NewThread(slackThread)
	ctx = msg.With(ctx, threadSvc.Reply, threadSvc.NewStateFunc)

	switch actionID {
	case slack.ActionIDAckAlert:
		return uc.ackAlert(ctx, user, slackThread, types.AlertID(value))

	case slack.ActionIDAckList:
		return uc.ackList(ctx, user, slackThread, types.AlertListID(value))

	case slack.ActionIDBindAlert:
		return uc.bindAlert(ctx, user, slackThread, types.AlertID(value), triggerID)

	case slack.ActionIDBindList:
		return uc.bindList(ctx, user, slackThread, types.AlertListID(value), triggerID)

	case slack.ActionIDResolveTicket:
		return uc.resolveTicket(ctx, user, slackThread, types.TicketID(value), triggerID)
	}

	return nil
}

func (uc *UseCases) resolveTicket(ctx context.Context, user slack.User, slackThread slack.Thread, targetTicketID types.TicketID, triggerID string) error {
	ticket, err := uc.repository.GetTicket(ctx, targetTicketID)
	if err != nil {
		return goerr.Wrap(err, "failed to get ticket")
	} else if ticket == nil {
		return goerr.New("ticket not found", goerr.V("ticket_id", targetTicketID))
	}

	if err := uc.slackService.ShowResolveTicketModal(ctx, ticket, triggerID); err != nil {
		return goerr.Wrap(err, "failed to show resolve ticket modal")
	}

	return nil
}

func (uc *UseCases) bindAlert(ctx context.Context, user slack.User, slackThread slack.Thread, targetAlertID types.AlertID, triggerID string) error {
	targetAlert, err := uc.repository.GetAlert(ctx, targetAlertID)
	if err != nil {
		return goerr.Wrap(err, "failed to get alert")
	} else if targetAlert == nil {
		return goerr.New("alert not found")
	}

	alerts, err := uc.repository.FindSimilarAlerts(ctx, *targetAlert, 10)
	if err != nil {
		return goerr.Wrap(err, "failed to find similar alerts")
	}

	var similarAlertTicketIDs []types.TicketID
	now := clock.Now(ctx)
	for _, alert := range alerts {
		if alert.CreatedAt.Before(now.Add(-72 * time.Hour)) {
			continue
		}
		similarAlertTicketIDs = append(similarAlertTicketIDs, alert.TicketID)
	}

	tickets, err := uc.repository.BatchGetTickets(ctx, similarAlertTicketIDs)
	if err != nil {
		return goerr.Wrap(err, "failed to get tickets by alert")
	}

	target := fmt.Sprintf("alert:%s", targetAlertID)
	if err := uc.slackService.ShowBindToTicketModal(ctx, slack.CallbackSubmitBindAlert, tickets, triggerID, target); err != nil {
		return goerr.Wrap(err, "failed to show bind alert modal")
	}

	return nil
}

func (uc *UseCases) bindList(ctx context.Context, user slack.User, slackThread slack.Thread, targetListID types.AlertListID, triggerID string) error {
	if err := uc.slackService.ShowBindToTicketModal(ctx, slack.CallbackSubmitBindList, []*ticket.Ticket{}, triggerID, targetListID.String()); err != nil {
		return goerr.Wrap(err, "failed to show bind list modal")
	}

	return nil
}

func (uc *UseCases) ackAlert(ctx context.Context, user slack.User, slackThread slack.Thread, targetAlertID types.AlertID) error {
	st := uc.slackService.NewThread(slackThread)

	targetAlert, err := uc.repository.GetAlert(ctx, targetAlertID)
	if err != nil {
		return goerr.Wrap(err, "failed to get alert")
	} else if targetAlert == nil {
		return goerr.New("alert not found")
	}

	newTicket := ticket.New(ctx, []types.AlertID{targetAlert.ID}, &slackThread)
	newTicket.Assignee = &user
	newTicket.Status = types.TicketStatusAcknowledged
	newTicket.Embedding = targetAlert.Embedding

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
