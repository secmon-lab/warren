package usecase

import (
	"context"
	_ "embed"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/lang"
	"github.com/secmon-lab/warren/pkg/domain/model/prompt"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/embedding"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

//go:embed prompt/resolve_message.md
var resolveMessagePromptTemplate string

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

func getSlackSelectValue[T ~string](values slack.StateValue, blockID slack.BlockID, actionID slack.BlockActionID) (T, bool) {
	if block, ok := values[blockID.String()]; ok {
		if action, ok := block[actionID.String()]; ok {
			return T(action.SelectedOption.Value), true
		}
	}
	return T(""), false
}

func getTicketID(values slack.StateValue) (types.TicketID, error) {
	inputTicketID, ok := getSlackValue[types.TicketID](values, slack.BlockIDTicketID, slack.BlockActionIDTicketID)
	if !ok {
		return "", goerr.New("ticket ID not found (invalid schema)", goerr.V("values", values))
	}

	selectedTicketID, ok := getSlackSelectValue[types.TicketID](values, slack.BlockIDTicketSelect, slack.BlockActionIDTicketSelect)
	if !ok {
		return "", goerr.New("ticket ID not found (invalid schema)", goerr.V("values", values))
	}

	var ticketID types.TicketID
	if inputTicketID != "" {
		ticketID = types.TicketID(inputTicketID)
	} else if selectedTicketID != "" {
		ticketID = types.TicketID(selectedTicketID)
	} else {
		return "", goerr.New("ticket ID not found (invalid schema)", goerr.V("values", values))
	}

	return ticketID, nil
}

func (uc *UseCases) handleSlackInteractionViewSubmissionBindAlert(ctx context.Context, user slack.User, metadata string, values slack.StateValue) error {
	logger := logging.From(ctx)
	logger.Debug("binding alert",
		"user", user,
		"metadata", metadata,
		"values", values,
	)
	ctx = msg.Trace(ctx, "💥 binding alert\n> %s", metadata)

	alertID := types.AlertID(metadata)
	alert, err := uc.repository.GetAlert(ctx, alertID)
	if err != nil {
		return goerr.Wrap(err, "failed to get alert", goerr.V("alert_id", alertID))
	}
	if alert == nil {
		return goerr.Wrap(err, "alert not found", goerr.V("alert_id", alertID))
	}

	st := uc.slackService.NewThread(*alert.SlackThread)
	ctx = msg.With(ctx, st.Reply, st.NewStateFunc)

	ticketID, err := getTicketID(values)
	if err != nil {
		_ = msg.Trace(ctx, "💥 Failed to get ticket ID\n> %s", err.Error())
		return err
	}

	return uc.handleBindAlerts(ctx, user, ticketID, []types.AlertID{alertID})
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

	st := uc.slackService.NewThread(*list.SlackThread)
	ctx = msg.With(ctx, st.Reply, st.NewStateFunc)

	ticketID, err := getTicketID(values)
	if err != nil {
		_ = msg.Trace(ctx, "💥 Failed to get ticket ID\n> %s", err.Error())
		return err
	}

	err = uc.handleBindAlerts(ctx, user, ticketID, list.AlertIDs)
	if err != nil {
		return err
	}

	// Update the alert list status to bound
	list.Status = alert.ListStatusBound
	if err := uc.repository.PutAlertList(ctx, list); err != nil {
		logger.Warn("failed to update alert list status", "error", err)
	}

	return nil
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

	embeddings := make([]firestore.Vector32, len(alerts))
	for i, alert := range alerts {
		embeddings[i] = alert.Embedding
	}
	ticket.Embedding = embedding.Average(embeddings)

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

	msg.Notify(ctx, "🎉 Alert bound to ticket to %s (%s)", ticketID, ticket.Metadata.Title)

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

// generateResolveMessage generates a humorous message for when a ticket is resolved
func (uc *UseCases) generateResolveMessage(ctx context.Context, ticket *ticket.Ticket) string {
	conclusionText := ""
	if ticket.Conclusion != "" {
		conclusionText = string(ticket.Conclusion)
	}

	reasonText := ticket.Reason
	if reasonText == "" {
		reasonText = "No reason provided"
	}

	// Generate prompt with ticket information
	resolvePrompt, err := prompt.Generate(ctx, resolveMessagePromptTemplate, map[string]any{
		"title":      ticket.Metadata.Title,
		"conclusion": conclusionText,
		"reason":     reasonText,
		"lang":       lang.From(ctx),
	})
	if err != nil {
		// Fallback to default message if prompt generation fails
		return "🎉 Great work! Ticket resolved successfully 🎯"
	}

	// Create LLM session
	session, err := uc.llmClient.NewSession(ctx)
	if err != nil {
		// Fallback to default message if session creation fails
		return "🎉 Great work! Ticket resolved successfully 🎯"
	}

	// Generate content
	response, err := session.GenerateContent(ctx, gollem.Text(resolvePrompt))
	if err != nil || len(response.Texts) == 0 || response.Texts[0] == "" {
		// Fallback to default message if generation fails
		return "🎉 Great work! Ticket resolved successfully 🎯"
	}

	return response.Texts[0]
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
		_ = msg.Trace(ctx, "💥 Failed to get ticket\n> %s", err.Error())
		return goerr.Wrap(err, "failed to get ticket")
	}
	if target == nil {
		msg.Notify(ctx, "💥 Ticket not found")
		return nil
	}

	st := uc.slackService.NewThread(*target.SlackThread)
	ctx = msg.With(ctx, st.Reply, st.NewStateFunc)

	conclusion, ok := getSlackSelectValue[types.AlertConclusion](values,
		slack.BlockIDTicketConclusion,
		slack.BlockActionIDTicketConclusion,
	)
	if !ok || conclusion == "" {
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

	// Generate and send humorous resolve message
	resolveMessage := uc.generateResolveMessage(ctx, target)
	msg.Notify(ctx, "%s", resolveMessage)

	return nil
}
