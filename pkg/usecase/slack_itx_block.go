package usecase

import (
	"context"
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/tag"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
	"github.com/secmon-lab/warren/pkg/utils/user"
)

// HandleSlackInteractionBlockActions handles a slack interaction block action.
func (uc *UseCases) HandleSlackInteractionBlockActions(ctx context.Context, slackUser slack.User, slackThread slack.Thread, actionID slack.ActionID, value, triggerID string) error {
	ctx = user.WithUserID(ctx, slackUser.ID)

	logger := logging.From(ctx)
	logger.Info("HandleSlackInteractionBlockActions", "action_id", actionID, "value", value, "user", slackUser.ID)

	if uc.slackService == nil {
		return goerr.New("slack service not configured")
	}
	threadSvc := uc.slackService.NewThread(slackThread)
	traceFunc := func(ctx context.Context, message string) {
		threadSvc.NewStateFunc(ctx, message)
	}
	ctx = msg.With(ctx, threadSvc.Reply, traceFunc, createSlackWarnFunc(threadSvc))

	switch actionID {
	case slack.ActionIDAckAlert:
		return uc.slackActionAckAlert(ctx, slackUser, slackThread, types.AlertID(value))

	case slack.ActionIDAckList:
		return uc.slackActionAckList(ctx, slackUser, slackThread, types.AlertListID(value))

	case slack.ActionIDBindAlert:
		return uc.slackActionBindAlert(ctx, types.AlertID(value), triggerID)

	case slack.ActionIDBindList:
		return uc.slackActionBindList(ctx, slackUser, slackThread, types.AlertListID(value), triggerID)

	case slack.ActionIDResolveTicket:
		return uc.showResolveTicketModal(ctx, slackUser, slackThread, types.TicketID(value), triggerID)

	case slack.ActionIDSalvage:
		return uc.showSalvageModal(ctx, slackUser, slackThread, types.TicketID(value), triggerID)

	case slack.ActionIDEditTicket:
		return uc.showEditTicketModal(ctx, slackUser, slackThread, types.TicketID(value), triggerID)

	case slack.ActionIDDeclineAlert:
		return uc.slackActionDeclineAlert(ctx, slackUser, slackThread, types.AlertID(value))

	case slack.ActionIDReopenAlert:
		return uc.slackActionReopenAlert(ctx, slackUser, slackThread, types.AlertID(value))

	case slack.ActionIDEscalate:
		return uc.handleEscalateAction(ctx, slackUser, slackThread, value)
	}

	return nil
}

func (uc *UseCases) ackAlerts(ctx context.Context, user slack.User, slackThread slack.Thread, alerts alert.Alerts) error {
	// Extract alert IDs
	alertIDs := make([]types.AlertID, len(alerts))
	for i, alert := range alerts {
		alertIDs[i] = alert.ID
	}

	// Use the unified CreateTicketFromAlerts method for both single and multiple alerts
	_, err := uc.CreateTicketFromAlerts(ctx, alertIDs, &user, &slackThread)
	if err != nil {
		return goerr.Wrap(err, "failed to create ticket from alerts")
	}

	msg.Trace(ctx, "🎫 Ticket created")

	return nil
}

func (uc *UseCases) slackActionAckAlert(ctx context.Context, user slack.User, slackThread slack.Thread, targetAlertID types.AlertID) error {
	targetAlert, err := uc.repository.GetAlert(ctx, targetAlertID)
	if err != nil {
		return goerr.Wrap(err, "failed to get alert")
	} else if targetAlert == nil {
		return goerr.New("alert not found")
	}

	targetAlert.Normalize()
	if targetAlert.Status == alert.AlertStatusDeclined {
		msg.Notify(ctx, "🚫 This alert has been declined. Please Re-open it first before acknowledging.")
		return nil
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

	err = uc.ackAlerts(ctx, user, slackThread, alerts)
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

	// Update the alert list message to show acknowledged status
	st := uc.slackService.NewThread(slackThread)
	if list.SlackMessageID != "" {
		if err := st.UpdateAlertList(ctx, list, "acknowledged"); err != nil {
			logger.Warn("failed to update alert list", "error", err)
		}
	}

	return nil
}

func (uc *UseCases) slackActionBindAlert(ctx context.Context, targetAlertID types.AlertID, triggerID string) error {
	targetAlert, err := uc.repository.GetAlert(ctx, targetAlertID)
	if err != nil {
		return goerr.Wrap(err, "failed to get alert")
	} else if targetAlert == nil {
		return goerr.New("alert not found")
	}

	targetAlert.Normalize()
	if targetAlert.Status == alert.AlertStatusDeclined {
		msg.Notify(ctx, "🚫 This alert has been declined. Please Re-open it first before binding to a ticket.")
		return nil
	}

	nearestTickets, err := uc.repository.FindNearestTicketsWithSpan(ctx, targetAlert.Embedding, clock.Now(ctx).Add(-72*time.Hour), clock.Now(ctx), 10)
	if err != nil {
		return goerr.Wrap(err, "failed to find similar tickets")
	}

	if err := uc.slackService.ShowBindToTicketModal(ctx, slack.CallbackSubmitBindAlert, nearestTickets, triggerID, targetAlertID.String()); err != nil {
		return goerr.Wrap(err, "failed to show bind alert modal")
	}

	return nil
}

func (uc *UseCases) slackActionBindList(ctx context.Context, _ slack.User, _ slack.Thread, targetListID types.AlertListID, triggerID string) error {
	list, err := uc.repository.GetAlertList(ctx, targetListID)
	if err != nil {
		return goerr.Wrap(err, "failed to get alert list")
	}
	if list == nil {
		return goerr.New("alert list not found")
	}

	nearestTickets, err := uc.repository.FindNearestTicketsWithSpan(ctx, list.Embedding, clock.Now(ctx).Add(-72*time.Hour), clock.Now(ctx), 10)
	if err != nil {
		return goerr.Wrap(err, "failed to find similar tickets")
	}

	if err := uc.slackService.ShowBindToTicketModal(ctx, slack.CallbackSubmitBindList, nearestTickets, triggerID, targetListID.String()); err != nil {
		return goerr.Wrap(err, "failed to show bind list modal")
	}

	return nil
}

func (uc *UseCases) showResolveTicketModal(ctx context.Context, _ slack.User, _ slack.Thread, targetTicketID types.TicketID, triggerID string) error {
	ticket, err := uc.repository.GetTicket(ctx, targetTicketID)
	if err != nil {
		return goerr.Wrap(err, "failed to get ticket")
	} else if ticket == nil {
		return goerr.New("ticket not found", goerr.V("ticket_id", targetTicketID))
	}

	// Fetch available tags
	var availableTags []*tag.Tag
	if uc.tagService != nil {
		tags, err := uc.tagService.ListAllTags(ctx)
		if err != nil {
			// Log error but continue without tags
			logging.From(ctx).Warn("failed to list tags for resolve modal", "error", err)
		} else {
			availableTags = tags
		}
	}

	if err := uc.slackService.ShowResolveTicketModal(ctx, ticket, triggerID, availableTags); err != nil {
		return goerr.Wrap(err, "failed to show resolve ticket modal")
	}

	return nil
}

func (uc *UseCases) showSalvageModal(ctx context.Context, _ slack.User, _ slack.Thread, targetTicketID types.TicketID, triggerID string) error {
	ticket, err := uc.repository.GetTicket(ctx, targetTicketID)
	if err != nil {
		return goerr.Wrap(err, "failed to get ticket")
	} else if ticket == nil {
		return goerr.New("ticket not found", goerr.V("ticket_id", targetTicketID))
	}

	// Get all unbound alerts with default threshold 0.9
	unboundAlerts, err := uc.getSalvageableAlerts(ctx, ticket, 0.9, "")
	if err != nil {
		return goerr.Wrap(err, "failed to get salvageable alerts")
	}

	if err := uc.slackService.ShowSalvageModal(ctx, ticket, unboundAlerts, triggerID); err != nil {
		return goerr.Wrap(err, "failed to show salvage modal")
	}

	return nil
}

func (uc *UseCases) showEditTicketModal(ctx context.Context, _ slack.User, _ slack.Thread, targetTicketID types.TicketID, triggerID string) error {
	ticket, err := uc.repository.GetTicket(ctx, targetTicketID)
	if err != nil {
		return goerr.Wrap(err, "failed to get ticket")
	} else if ticket == nil {
		return goerr.New("ticket not found", goerr.V("ticket_id", targetTicketID))
	}

	if err := uc.slackService.ShowEditTicketModal(ctx, ticket, triggerID); err != nil {
		return goerr.Wrap(err, "failed to show edit ticket modal")
	}

	return nil
}

func (uc *UseCases) getSalvageableAlerts(ctx context.Context, ticket *ticket.Ticket, threshold float64, keyword string) (alert.Alerts, error) {
	// Get all unbound alerts
	unboundAlerts, err := uc.repository.GetAlertWithoutTicket(ctx, 0, 0)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get unbound alerts")
	}

	var filteredAlerts alert.Alerts

	for _, alert := range unboundAlerts {
		include := true

		// Apply threshold filter if threshold > 0
		if threshold > 0 && ticket.Embedding != nil && alert.Embedding != nil {
			similarity := calculateCosineSimilarity(ticket.Embedding, alert.Embedding)
			if similarity < threshold {
				include = false
			}
		}

		// Apply keyword filter if keyword is not empty
		if include && keyword != "" {
			dataBytes, err := json.Marshal(alert.Data)
			if err != nil {
				continue // Skip this alert if data can't be marshaled
			}
			if !containsKeyword(string(dataBytes), keyword) {
				include = false
			}
		}

		if include {
			filteredAlerts = append(filteredAlerts, alert)
		}
	}

	return filteredAlerts, nil
}

func calculateCosineSimilarity(a, b []float32) float64 {
	if a == nil || b == nil {
		return 0
	}

	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64

	for i := range len(a) {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

func containsKeyword(text, keyword string) bool {
	return strings.Contains(strings.ToLower(text), strings.ToLower(keyword))
}

func (uc *UseCases) HandleSalvageRefresh(ctx context.Context, user slack.User, metadata string, values slack.StateValue, viewID string) error {
	ticketID := types.TicketID(metadata)
	target, err := uc.repository.GetTicket(ctx, ticketID)
	if err != nil {
		return goerr.Wrap(err, "failed to get ticket")
	} else if target == nil {
		return goerr.New("ticket not found", goerr.V("ticket_id", ticketID))
	}

	// Extract threshold and keyword from current form values
	thresholdStr, _ := getSlackValue[string](values,
		slack.BlockIDSalvageThreshold,
		slack.BlockActionIDSalvageThreshold,
	)

	keyword, _ := getSlackValue[string](values,
		slack.BlockIDSalvageKeyword,
		slack.BlockActionIDSalvageKeyword,
	)

	// Parse threshold - default to 0.9
	threshold := 0.9 // Default threshold
	if thresholdStr != "" {
		if parsed, err := strconv.ParseFloat(thresholdStr, 64); err == nil {
			threshold = parsed
		} else {
			logging.From(ctx).Warn("failed to parse threshold", "threshold_str", thresholdStr, "error", err)
		}
	}

	// Get updated salvageable alerts based on current form values
	unboundAlerts, err := uc.getSalvageableAlerts(ctx, target, threshold, keyword)
	if err != nil {
		return goerr.Wrap(err, "failed to get salvageable alerts")
	}

	// Update the modal view with refreshed alert list
	if err := uc.slackService.UpdateSalvageModal(ctx, target, unboundAlerts, viewID, threshold, keyword); err != nil {
		return goerr.Wrap(err, "failed to update salvage modal")
	}

	return nil
}

// handleEscalateAction handles the escalation of a notice to a full alert
func (uc *UseCases) handleEscalateAction(ctx context.Context, slackUser slack.User, slackThread slack.Thread, value string) error {
	logger := logging.From(ctx)

	// Extract notice ID from the value
	// The value should contain the notice ID
	noticeID := types.NoticeID(value)

	logger.Info("escalating notice", "notice_id", noticeID, "user", slackUser.ID)

	// Call the EscalateNotice method from alert.go
	escalatedAlert, err := uc.EscalateNotice(ctx, noticeID)
	if err != nil {
		return goerr.Wrap(err, "failed to escalate notice", goerr.V("notice_id", noticeID))
	}

	// Send confirmation message with link to new alert thread
	if uc.slackService != nil {
		threadSvc := uc.slackService.NewThread(slackThread)

		// Generate URL to the new alert thread
		var message string
		if escalatedAlert.SlackThread != nil {
			alertURL := uc.slackService.ToExternalMsgURL(
				escalatedAlert.SlackThread.ChannelID,
				escalatedAlert.SlackThread.ThreadID,
			)
			message = "✅ Notice escalated to full alert: <" + alertURL + "|View Alert>"
		} else {
			message = "✅ Notice escalated to full alert"
		}

		threadSvc.Reply(ctx, message)
	}

	logger.Info("notice escalated successfully", "notice_id", noticeID)
	return nil
}

func (uc *UseCases) slackActionDeclineAlert(ctx context.Context, slackUser slack.User, slackThread slack.Thread, targetAlertID types.AlertID) error {
	targetAlert, err := uc.repository.GetAlert(ctx, targetAlertID)
	if err != nil {
		return goerr.Wrap(err, "failed to get alert", goerr.V("alert_id", targetAlertID))
	} else if targetAlert == nil {
		return goerr.New("alert not found", goerr.V("alert_id", targetAlertID))
	}

	// Update status to declined
	if err := uc.repository.UpdateAlertStatus(ctx, targetAlertID, alert.AlertStatusDeclined); err != nil {
		return goerr.Wrap(err, "failed to decline alert", goerr.V("alert_id", targetAlertID))
	}

	// Update the Slack message to show declined state
	targetAlert.Status = alert.AlertStatusDeclined
	if targetAlert.HasSlackThread() && uc.slackService != nil {
		alertThread := uc.slackService.NewThread(*targetAlert.SlackThread)
		if err := alertThread.UpdateAlert(ctx, *targetAlert); err != nil {
			logging.From(ctx).Warn("failed to update alert Slack display", "error", err, "alert_id", targetAlertID)
		}
	}

	// Post to thread that user declined
	msg.Notify(ctx, "🚫 Declined by %s", slackUser.Name)

	return nil
}

func (uc *UseCases) slackActionReopenAlert(ctx context.Context, slackUser slack.User, slackThread slack.Thread, targetAlertID types.AlertID) error {
	targetAlert, err := uc.repository.GetAlert(ctx, targetAlertID)
	if err != nil {
		return goerr.Wrap(err, "failed to get alert", goerr.V("alert_id", targetAlertID))
	} else if targetAlert == nil {
		return goerr.New("alert not found", goerr.V("alert_id", targetAlertID))
	}

	// Update status back to unbound
	if err := uc.repository.UpdateAlertStatus(ctx, targetAlertID, alert.AlertStatusUnbound); err != nil {
		return goerr.Wrap(err, "failed to reopen alert", goerr.V("alert_id", targetAlertID))
	}

	// Update the Slack message to show unbound state
	targetAlert.Status = alert.AlertStatusUnbound
	if targetAlert.HasSlackThread() && uc.slackService != nil {
		alertThread := uc.slackService.NewThread(*targetAlert.SlackThread)
		if err := alertThread.UpdateAlert(ctx, *targetAlert); err != nil {
			logging.From(ctx).Warn("failed to update alert Slack display", "error", err, "alert_id", targetAlertID)
		}
	}

	// Post to thread that user re-opened
	msg.Notify(ctx, "🔄 Re-opened by %s", slackUser.Name)

	return nil
}
