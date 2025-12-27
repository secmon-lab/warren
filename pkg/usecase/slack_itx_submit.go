package usecase

import (
	"context"
	_ "embed"
	"encoding/json"
	"strconv"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/lang"
	"github.com/secmon-lab/warren/pkg/domain/model/prompt"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
	"github.com/secmon-lab/warren/pkg/utils/user"
)

//go:embed prompt/resolve_message.md
var resolveMessagePromptTemplate string

func (uc *UseCases) HandleSlackInteractionViewSubmission(ctx context.Context, slackUser slack.User, callbackID slack.CallbackID, metadata string, values slack.StateValue) error {
	// Set user ID in context for activity tracking
	ctx = user.WithUserID(ctx, slackUser.ID)

	logger := logging.From(ctx)
	logger.Debug("resolving alert",
		"user", slackUser,
		"callback_id", callbackID,
		"metadata", metadata,
		"values", values,
	)
	switch callbackID {
	case slack.CallbackSubmitResolveTicket:
		return uc.handleSlackInteractionViewSubmissionResolveTicket(ctx, slackUser, metadata, values)
	case slack.CallbackSubmitBindAlert:
		return uc.handleSlackInteractionViewSubmissionBindAlert(ctx, slackUser, metadata, values)
	case slack.CallbackSubmitBindList:
		return uc.handleSlackInteractionViewSubmissionBindList(ctx, slackUser, metadata, values)
	case slack.CallbackSubmitSalvage:
		return uc.handleSlackInteractionViewSubmissionSalvage(ctx, slackUser, metadata, values)
	case slack.CallbackEditTicket:
		return uc.handleSlackInteractionViewSubmissionEditTicket(ctx, slackUser, metadata, values)
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

func (uc *UseCases) handleSlackInteractionViewSubmissionBindAlert(ctx context.Context, slackUser slack.User, metadata string, values slack.StateValue) error {
	logger := logging.From(ctx)
	logger.Debug("binding alert",
		"user", slackUser,
		"metadata", metadata,
		"values", values,
	)
	msg.Trace(ctx, "💥 binding alert\n> %s", metadata)

	alertID := types.AlertID(metadata)
	alert, err := uc.repository.GetAlert(ctx, alertID)
	if err != nil {
		return goerr.Wrap(err, "failed to get alert", goerr.V("alert_id", alertID))
	}
	if alert == nil {
		return goerr.Wrap(err, "alert not found", goerr.V("alert_id", alertID))
	}

	if uc.slackService == nil {
		return goerr.New("slack service not configured")
	}
	st := uc.slackService.NewThread(*alert.SlackThread)
	traceFunc := func(ctx context.Context, message string) {
		st.NewStateFunc(ctx, message)
	}
	ctx = msg.With(ctx, st.Reply, traceFunc, createSlackWarnFunc(st))

	ticketID, err := getTicketID(values)
	if err != nil {
		msg.Trace(ctx, "💥 Failed to get ticket ID\n> %s", err.Error())
		return err
	}

	return uc.handleBindAlerts(ctx, slackUser, ticketID, []types.AlertID{alertID})
}

func (uc *UseCases) handleSlackInteractionViewSubmissionBindList(ctx context.Context, slackUser slack.User, metadata string, values slack.StateValue) error {
	logger := logging.From(ctx)
	logger.Debug("binding list",
		"user", slackUser,
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
	traceFunc := func(ctx context.Context, message string) {
		st.NewStateFunc(ctx, message)
	}
	ctx = msg.With(ctx, st.Reply, traceFunc, createSlackWarnFunc(st))

	ticketID, err := getTicketID(values)
	if err != nil {
		msg.Trace(ctx, "💥 Failed to get ticket ID\n> %s", err.Error())
		return err
	}

	err = uc.handleBindAlerts(ctx, slackUser, ticketID, list.AlertIDs)
	if err != nil {
		return err
	}

	// Update the alert list status to bound
	list.Status = alert.ListStatusBound
	if err := uc.repository.PutAlertList(ctx, list); err != nil {
		if data, jsonErr := json.Marshal(list); jsonErr == nil {
			logger.Error("failed to save alert list", "error", err, "list", string(data))
		}
		logger.Warn("failed to update alert list status", "error", err)
	}

	// Update the alert list message to show bound status
	if list.SlackMessageID != "" {
		if err := st.UpdateAlertList(ctx, list, "bound"); err != nil {
			logger.Warn("failed to update alert list", "error", err)
		}
	}

	return nil
}

func (uc *UseCases) handleBindAlerts(ctx context.Context, slackUser slack.User, ticketID types.TicketID, alertIDs []types.AlertID) error {
	logger := logging.From(ctx)
	logger.Debug("binding alerts",
		"user", slackUser,
		"ticket_id", ticketID,
		"alert_ids", alertIDs,
	)

	// Use the unified BindAlertsToTicket usecase which handles:
	// - Repository binding (bidirectional)
	// - Embedding recalculation
	// - Slack updates for ticket and individual alerts
	if err := uc.BindAlertsToTicket(ctx, ticketID, alertIDs); err != nil {
		return goerr.Wrap(err, "failed to bind alerts to ticket", goerr.V("ticket_id", ticketID), goerr.V("alert_ids", alertIDs))
	}

	// Get updated ticket for notification
	ticket, err := uc.repository.GetTicket(ctx, ticketID)
	if err != nil {
		return goerr.Wrap(err, "failed to get updated ticket", goerr.V("ticket_id", ticketID))
	}

	msg.Notify(ctx, "🎉 Alert bound to ticket to %s (%s)", ticketID, ticket.Title)

	return nil
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

	// Get ticket comments
	comments, err := uc.repository.GetTicketComments(ctx, ticket.ID)
	if err != nil {
		// Continue without comments if there's an error
		comments = nil
	}

	// Generate prompt with ticket information including comments
	resolvePrompt, err := prompt.GenerateWithStruct(ctx, resolveMessagePromptTemplate, map[string]any{
		"title":      ticket.Title,
		"conclusion": conclusionText,
		"reason":     reasonText,
		"comments":   comments,
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
	logger.Debug("getting ticket", "ticket_id", ticketID)
	target, err := uc.repository.GetTicket(ctx, ticketID)
	if err != nil {
		msg.Trace(ctx, "💥 Failed to get ticket\n> %s", err.Error())
		return goerr.Wrap(err, "failed to get ticket")
	}
	if target == nil {
		msg.Notify(ctx, "💥 Ticket not found")
		return nil
	}

	if uc.slackService == nil {
		return goerr.New("slack service not configured")
	}
	st := uc.slackService.NewThread(*target.SlackThread)
	traceFunc := func(ctx context.Context, message string) {
		st.NewStateFunc(ctx, message)
	}
	ctx = msg.With(ctx, st.Reply, traceFunc, createSlackWarnFunc(st))

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
		// Comment is optional, set empty string if not provided
		reason = ""
		logger.Debug("comment field not provided, using empty string")
	}

	target.Conclusion = conclusion
	target.Reason = reason
	target.Status = types.TicketStatusResolved

	// Handle tag selection if available (supports both checkbox and multi-select)
	if block, ok := values[slack.BlockIDTicketTags.String()]; ok {
		if action, ok := block[slack.BlockActionIDTicketTags.String()]; ok {
			logger.Debug("processing tag selection",
				"action", action,
				"selected_options", action.SelectedOptions,
				"selected_option", action.SelectedOption,
			)

			var selectedTagIDs []string

			// Handle checkbox and multi-select formats
			if len(action.SelectedOptions) > 0 {
				// Multi-select or checkbox with multiple selections
				selectedTagIDs = make([]string, 0, len(action.SelectedOptions))
				for _, option := range action.SelectedOptions {
					selectedTagIDs = append(selectedTagIDs, option.Value)
				}
			}

			logger.Debug("extracted values from modal", "values", selectedTagIDs)

			// Process values - could be tag IDs (existing tags) or tag names (new tags to create)
			if len(selectedTagIDs) > 0 && uc.tagService != nil {
				logger.Debug("processing tag values", "values", selectedTagIDs)

				// Initialize TagIDs if needed
				if target.TagIDs == nil {
					target.TagIDs = make(map[string]bool)
				}

				// For each value, check if it's a valid tag ID first, then try as tag name
				for _, value := range selectedTagIDs {
					// First try to find by ID (for existing tags sent as IDs)
					if tag, err := uc.repository.GetTagByID(ctx, value); err == nil && tag != nil {
						target.TagIDs[tag.ID] = true
						logger.Debug("found existing tag by ID", "tag_id", tag.ID, "tag_name", tag.Name)
					} else {
						// If not found by ID, try to convert as tag name (for new tags or legacy names)
						logger.Debug("value not found as tag ID, trying as tag name", "value", value)
						tags, err := uc.tagService.ConvertNamesToTags(ctx, []string{value})
						if err != nil {
							logger.Warn("failed to convert tag name to ID", "error", err, "tag_name", value)
							// Continue with other tags even if one fails
						} else if len(tags) > 0 {
							target.TagIDs[tags[0]] = true
							logger.Debug("successfully converted tag name to ID", "tag_name", value, "tag_id", tags[0])
						}
					}
				}
				logger.Debug("target ticket updated with tag IDs", "TagIDs", target.TagIDs)
			} else {
				logger.Debug("no tag values selected or tag service unavailable")
			}
		}
	}

	logger.Debug("saving ticket to repository", "ticketID", target.ID, "TagIDs", target.TagIDs, "target.Status", target.Status)
	if err := uc.repository.PutTicket(ctx, *target); err != nil {
		if data, jsonErr := json.Marshal(target); jsonErr == nil {
			logger.Error("failed to save ticket", "error", err, "ticket", string(data))
		}
		return goerr.Wrap(err, "failed to put ticket", goerr.V("ticket_id", ticketID), goerr.V("ticket", target))
	}

	if _, err := st.PostTicket(ctx, target, nil); err != nil {
		return goerr.Wrap(err, "failed to update slack thread")
	}

	// Generate and send humorous resolve message
	resolveMessage := uc.generateResolveMessage(ctx, target)
	msg.Notify(ctx, "%s", resolveMessage)

	return nil
}

func (uc *UseCases) handleSlackInteractionViewSubmissionSalvage(ctx context.Context, user slack.User, metadata string, values slack.StateValue) error {
	logger := logging.From(ctx)
	logger.Debug("salvaging alerts",
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
	traceFunc := func(ctx context.Context, message string) {
		st.NewStateFunc(ctx, message)
	}
	ctx = msg.With(ctx, st.Reply, traceFunc, createSlackWarnFunc(st))

	// Get threshold and keyword from form values
	thresholdStr, _ := getSlackValue[string](values,
		slack.BlockIDSalvageThreshold,
		slack.BlockActionIDSalvageThreshold,
	)

	keyword, _ := getSlackValue[string](values,
		slack.BlockIDSalvageKeyword,
		slack.BlockActionIDSalvageKeyword,
	)

	// Parse threshold
	var threshold float64
	if thresholdStr != "" {
		if parsed, err := strconv.ParseFloat(thresholdStr, 64); err == nil {
			threshold = parsed
		}
	}

	// Get salvageable alerts based on current form values
	unboundAlerts, err := uc.getSalvageableAlerts(ctx, target, threshold, keyword)
	if err != nil {
		return goerr.Wrap(err, "failed to get salvageable alerts")
	}

	if len(unboundAlerts) == 0 {
		msg.Notify(ctx, "📭 No alerts found matching the criteria")
		return nil
	}

	// Convert alerts to alert IDs
	alertIDs := make([]types.AlertID, len(unboundAlerts))
	for i, alert := range unboundAlerts {
		alertIDs[i] = alert.ID
	}

	// Use the unified BindAlertsToTicket usecase which handles:
	// - Repository binding (bidirectional)
	// - Embedding recalculation
	// - Slack updates for ticket and individual alerts
	if err := uc.BindAlertsToTicket(ctx, ticketID, alertIDs); err != nil {
		return goerr.Wrap(err, "failed to bind alerts to ticket", goerr.V("ticket_id", ticketID), goerr.V("alert_ids", alertIDs))
	}

	// Get updated ticket for notification
	target, err = uc.repository.GetTicket(ctx, ticketID)
	if err != nil {
		return goerr.Wrap(err, "failed to get updated ticket", goerr.V("ticket_id", ticketID))
	}

	msg.Notify(ctx, "🎉 Salvaged %d alerts to ticket %s", len(unboundAlerts), target.Title)

	return nil
}

func (uc *UseCases) handleSlackInteractionViewSubmissionEditTicket(ctx context.Context, user slack.User, metadata string, values slack.StateValue) error {
	logger := logging.From(ctx)
	logger.Debug("editing ticket",
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

	if uc.slackService == nil {
		return goerr.New("slack service not configured")
	}
	st := uc.slackService.NewThread(*target.SlackThread)
	traceFunc := func(ctx context.Context, message string) {
		st.NewStateFunc(ctx, message)
	}
	ctx = msg.With(ctx, st.Reply, traceFunc, createSlackWarnFunc(st))

	// Get new title (required)
	newTitle, ok := getSlackValue[string](values,
		slack.BlockIDTicketID,
		slack.BlockActionIDTicketTitle,
	)
	if !ok || newTitle == "" {
		return goerr.New("title is required")
	}

	// Get new description (optional)
	newDescription, _ := getSlackValue[string](values,
		slack.BlockIDTicketComment,
		slack.BlockActionIDTicketDesc,
	)

	// Update ticket metadata
	target.Title = newTitle
	target.Description = newDescription
	target.TitleSource = types.SourceHuman
	target.DescriptionSource = types.SourceHuman

	// Save updated ticket
	if err := uc.repository.PutTicket(ctx, *target); err != nil {
		if data, jsonErr := json.Marshal(target); jsonErr == nil {
			logger.Error("failed to save ticket", "error", err, "ticket", string(data))
		}
		return goerr.Wrap(err, "failed to update ticket", goerr.V("ticket_id", ticketID), goerr.V("ticket", target))
	}

	// Update Slack message
	if _, err := st.PostTicket(ctx, target, nil); err != nil {
		return goerr.Wrap(err, "failed to update slack thread")
	}

	msg.Notify(ctx, "✅ Ticket updated successfully")

	return nil
}
