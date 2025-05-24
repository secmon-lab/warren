package usecase

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/embedding"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

// HandleSlackInteractionBlockActions handles a slack interaction block action.
func (uc *UseCases) HandleSlackInteractionBlockActions(ctx context.Context, user slack.User, slackThread slack.Thread, actionID slack.ActionID, value, triggerID string) error {
	threadSvc := uc.slackService.NewThread(slackThread)
	ctx = msg.With(ctx, threadSvc.Reply, threadSvc.NewStateFunc)

	switch actionID {
	case slack.ActionIDAckAlert:
		return uc.slackActionAckAlert(ctx, user, slackThread, types.AlertID(value))

	case slack.ActionIDAckList:
		return uc.slackActionAckList(ctx, user, slackThread, types.AlertListID(value))

	case slack.ActionIDBindAlert:
		return uc.slackActionBindAlert(ctx, types.AlertID(value), triggerID)

	case slack.ActionIDBindList:
		return uc.slackActionBindList(ctx, user, slackThread, types.AlertListID(value), triggerID)

	case slack.ActionIDResolveTicket:
		return uc.showResolveTicketModal(ctx, user, slackThread, types.TicketID(value), triggerID)
	}

	return nil
}

func (uc *UseCases) ackAlerts(ctx context.Context, user slack.User, slackThread slack.Thread, alerts alert.Alerts) error {
	st := uc.slackService.NewThread(slackThread)
	alertIDs := make([]types.AlertID, len(alerts))
	for i, alert := range alerts {
		alertIDs[i] = alert.ID
	}

	embeddings := make([]firestore.Vector32, len(alerts))
	for i, alert := range alerts {
		embeddings[i] = alert.Embedding
	}

	newTicket := ticket.New(ctx, alertIDs, &slackThread)
	newTicket.Assignee = &user
	newTicket.Embedding = embedding.Averate(embeddings)

	if err := newTicket.FillMetadata(ctx, uc.llmClient, uc.repository); err != nil {
		return goerr.Wrap(err, "failed to fill ticket metadata")
	}

	ts, err := st.PostTicket(ctx, newTicket, alerts)
	if err != nil {
		return goerr.Wrap(err, "failed to post ticket")
	}
	newTicket.SlackMessageID = ts
	for _, alert := range alerts {
		alert.TicketID = newTicket.ID
	}

	if err := uc.repository.PutTicket(ctx, newTicket); err != nil {
		return goerr.Wrap(err, "failed to put ticket")
	}
	if err := uc.repository.BatchPutAlerts(ctx, alerts); err != nil {
		return goerr.Wrap(err, "failed to put alert")
	}

	for _, alert := range alerts {
		if err := st.UpdateAlert(ctx, *alert); err != nil {
			return goerr.Wrap(err, "failed to update slack thread")
		}
	}

	msg.Trace(ctx, "🎫 Ticket created. Why don't you ask <@%s> about it?", uc.slackService.BotID())
	return nil
}

func (uc *UseCases) slackActionAckAlert(ctx context.Context, user slack.User, slackThread slack.Thread, targetAlertID types.AlertID) error {
	targetAlert, err := uc.repository.GetAlert(ctx, targetAlertID)
	if err != nil {
		return goerr.Wrap(err, "failed to get alert")
	} else if targetAlert == nil {
		return goerr.New("alert not found")
	}

	return uc.ackAlerts(ctx, user, slackThread, alert.Alerts{targetAlert})
}

func (uc *UseCases) slackActionAckList(ctx context.Context, user slack.User, slackThread slack.Thread, targetListID types.AlertListID) error {
	logger := logging.From(ctx)

	list, err := uc.repository.GetAlertList(ctx, targetListID)
	if err != nil {
		return goerr.Wrap(err, "failed to get alert list")
	}
	if list == nil {
		logger.Error("alert list not found", "list_id", targetListID)
		return nil
	}

	alerts, err := list.GetAlerts(ctx, uc.repository)
	if err != nil {
		return goerr.Wrap(err, "failed to get alerts")
	}

	return uc.ackAlerts(ctx, user, slackThread, alerts)
}

func (uc *UseCases) slackActionBindAlert(ctx context.Context, targetAlertID types.AlertID, triggerID string) error {
	targetAlert, err := uc.repository.GetAlert(ctx, targetAlertID)
	if err != nil {
		return goerr.Wrap(err, "failed to get alert")
	} else if targetAlert == nil {
		return goerr.New("alert not found")
	}

	nearestTickets, err := uc.repository.FindNearestTickets(ctx, targetAlert.Embedding, 10)
	if err != nil {
		return goerr.Wrap(err, "failed to find similar tickets")
	}

	var tickets []*ticket.Ticket
	now := clock.Now(ctx)
	for _, ticket := range nearestTickets {
		if ticket.CreatedAt.Before(now.Add(-72 * time.Hour)) {
			continue
		}
		tickets = append(tickets, ticket)
	}

	if err := uc.slackService.ShowBindToTicketModal(ctx, slack.CallbackSubmitBindAlert, tickets, triggerID, targetAlertID.String()); err != nil {
		return goerr.Wrap(err, "failed to show bind alert modal")
	}

	return nil
}

func (uc *UseCases) slackActionBindList(ctx context.Context, user slack.User, slackThread slack.Thread, targetListID types.AlertListID, triggerID string) error {
	if err := uc.slackService.ShowBindToTicketModal(ctx, slack.CallbackSubmitBindList, []*ticket.Ticket{}, triggerID, targetListID.String()); err != nil {
		return goerr.Wrap(err, "failed to show bind list modal")
	}

	return nil
}

func (uc *UseCases) showResolveTicketModal(ctx context.Context, user slack.User, slackThread slack.Thread, targetTicketID types.TicketID, triggerID string) error {
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
