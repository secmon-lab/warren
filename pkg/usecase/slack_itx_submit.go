package usecase

import (
	"context"
	"encoding/json"
	"os"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

func (uc *UseCases) HandleSlackInteractionViewSubmission(ctx context.Context, user slack.User, callbackID slack.CallbackID, metadata string, values slack.StateValue) error {
	logger := logging.From(ctx)
	logger.Debug("resolving alert",
		"user", user,
		"callback_id", callbackID,
		"metadata", metadata,
		"values", values,
	)

	switch callbackID {
	case slack.CallbackSubmitResolveTicket:
		return uc.handleSlackInteractionViewSubmissionResolveTicket(ctx, user, metadata, values)
	case slack.CallbackSubmitBindAlert:
		return uc.handleSlackInteractionViewSubmissionBindAlert(ctx, user, metadata, values)
	case slack.CallbackSubmitBindList:
		return uc.handleSlackInteractionViewSubmissionBindList(ctx, user, metadata, values)
	}

	return nil
}

func getSlackValue[T ~string](values slack.StateValue, blockID slack.BlockID, actionID slack.BlockActionID) (T, bool) {
	if block, ok := values[blockID.String()]; ok {
		if action, ok := block[actionID.String()]; ok {
			return T(action.Value), true
		}
	}
	return T(""), false
}

func (uc *UseCases) handleSlackInteractionViewSubmissionBindAlert(ctx context.Context, user slack.User, metadata string, values slack.StateValue) error {
	logger := logging.From(ctx)
	logger.Debug("binding alert",
		"user", user,
		"metadata", metadata,
		"values", values,
	)

	ticketID, ok := getSlackValue[types.TicketID](values, slack.BlockIDTicketID, slack.BlockActionIDTicketID)
	if !ok {
		return goerr.New("ticket ID not found")
	}

	return uc.handleBindAlerts(ctx, user, types.TicketID(ticketID), []types.AlertID{types.AlertID(metadata)})
}

func (uc *UseCases) handleSlackInteractionViewSubmissionBindList(ctx context.Context, user slack.User, metadata string, values slack.StateValue) error {
	logger := logging.From(ctx)
	logger.Debug("binding list",
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

	ticketID, ok := getSlackValue[types.TicketID](values, slack.BlockIDTicketID, slack.BlockActionIDTicketID)
	if !ok {
		return goerr.New("ticket ID not found")
	}

	return uc.handleBindAlerts(ctx, user, ticketID, list.AlertIDs)
}

func (uc *UseCases) handleBindAlerts(ctx context.Context, user slack.User, ticketID types.TicketID, alertIDs []types.AlertID) error {
	logger := logging.From(ctx)
	logger.Debug("binding alerts",
		"user", user,
		"ticket_id", ticketID,
		"alert_ids", alertIDs,
	)

	ticket, err := uc.repository.GetTicket(ctx, ticketID)
	if err != nil {
		return goerr.Wrap(err, "failed to get ticket", goerr.V("ticket_id", ticketID))
	}
	if ticket == nil {
		return goerr.Wrap(err, "ticket not found", goerr.V("ticket_id", ticketID))
	}

	alerts, err := uc.repository.BatchGetAlerts(ctx, alertIDs)
	if err != nil {
		return goerr.Wrap(err, "failed to get alerts", goerr.V("alert_ids", alertIDs))
	}

	ticket.AlertIDs = unifyAlertIDs(ticket.AlertIDs, alertIDs)
	for _, alert := range alerts {
		ticket.AlertIDs = append(ticket.AlertIDs, alert.ID)
	}

	embeddings := make([]firestore.Vector32, len(alerts))
	for i, alert := range alerts {
		embeddings[i] = alert.Embedding
	}
	ticket.Embedding = averageEmbeddings(embeddings)

	// Update database
	if err := uc.repository.BatchBindAlertsToTicket(ctx, ticket.AlertIDs, ticketID); err != nil {
		return goerr.Wrap(err, "failed to bind alerts to ticket", goerr.V("ticket_id", ticketID), goerr.V("new_alert_ids", ticket.AlertIDs))
	}

	if err := uc.repository.PutTicket(ctx, *ticket); err != nil {
		return goerr.Wrap(err, "failed to put ticket", goerr.V("ticket_id", ticketID))
	}

	// Update slack view
	st := uc.slackService.NewThread(*ticket.SlackThread)

	if _, err := st.PostTicket(ctx, *ticket, alerts); err != nil {
		return goerr.Wrap(err, "failed to update slack thread")
	}

	for _, alert := range alerts {
		if alert.SlackThread != nil {
			st := uc.slackService.NewThread(*alert.SlackThread)
			if err := st.UpdateAlert(ctx, *alert); err != nil {
				return goerr.Wrap(err, "failed to update slack thread")
			}
		}
	}

	msg.Notify(ctx, "🎉 Alert bound to ticket")

	return nil
}

func unifyAlertIDs(oldAlertIDs, newAlertIDs []types.AlertID) []types.AlertID {
	// Create a map to eliminate duplicates
	idMap := make(map[types.AlertID]struct{})

	// Add old IDs to the map
	for _, id := range oldAlertIDs {
		idMap[id] = struct{}{}
	}

	// Add new IDs to the map
	for _, id := range newAlertIDs {
		idMap[id] = struct{}{}
	}

	// Create a slice from the map
	unified := make([]types.AlertID, 0, len(idMap))
	for id := range idMap {
		unified = append(unified, id)
	}

	return unified
}

func (uc *UseCases) handleSlackInteractionViewSubmissionResolveTicket(ctx context.Context, user slack.User, metadata string, values slack.StateValue) error {
	logger := logging.From(ctx)
	logger.Debug("resolving alert",
		"user", user,
		"metadata", metadata,
		"values", values,
	)

	ticketID := types.TicketID(metadata)
	target, err := uc.repository.GetTicket(ctx, ticketID)
	if err != nil {
		msg.Trace(ctx, "💥 Failed to get ticket\n> %s", err.Error())
		return goerr.Wrap(err, "failed to get ticket")
	}
	if target == nil {
		msg.Notify(ctx, "💥 Ticket not found")
		return nil
	}

	st := uc.slackService.NewThread(*target.SlackThread)
	ctx = msg.With(ctx, st.Reply, st.NewStateFunc)

	json.NewEncoder(os.Stdout).Encode(values)
	conclusion, ok := getSlackValue[types.AlertConclusion](values,
		slack.BlockIDTicketConclusion,
		slack.BlockActionIDTicketConclusion,
	)
	if !ok {
		return goerr.New("conclusion not found")
	}

	reason, ok := getSlackValue[string](values,
		slack.BlockIDTicketComment,
		slack.BlockActionIDTicketComment,
	)
	if !ok {
		return goerr.New("reason not found")
	}

	target.Conclusion = conclusion
	target.Reason = reason
	target.Status = types.TicketStatusResolved

	if err := uc.repository.PutTicket(ctx, *target); err != nil {
		return goerr.Wrap(err, "failed to put ticket", goerr.V("ticket_id", ticketID))
	}

	if _, err := st.PostTicket(ctx, *target, nil); err != nil {
		return goerr.Wrap(err, "failed to update slack thread")
	}

	msg.Notify(ctx, "🎉 Ticket resolved")

	return nil
}

func averageEmbeddings(embeddings []firestore.Vector32) firestore.Vector32 {
	if len(embeddings) == 0 {
		return firestore.Vector32{}
	}

	// Get dimension from first embedding
	dim := len(embeddings[0])
	sum := make([]float32, dim)

	// Sum up all embeddings
	for _, embedding := range embeddings {
		for i := 0; i < dim; i++ {
			sum[i] += embedding[i]
		}
	}

	// Calculate average
	avg := make([]float32, dim)
	n := float32(len(embeddings))
	for i := 0; i < dim; i++ {
		avg[i] = sum[i] / n
	}

	return avg
}
