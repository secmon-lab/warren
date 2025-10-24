package usecase

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/event"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/notice"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

type noopNotifier struct{}

func (n *noopNotifier) NotifyAlertPolicyResult(ctx context.Context, ev *event.AlertPolicyResultEvent) {
}

func (n *noopNotifier) NotifyEnrichPolicyResult(ctx context.Context, ev *event.EnrichPolicyResultEvent) {
}

func (n *noopNotifier) NotifyCommitPolicyResult(ctx context.Context, ev *event.CommitPolicyResultEvent) {
}

func (n *noopNotifier) NotifyEnrichTaskPrompt(ctx context.Context, ev *event.EnrichTaskPromptEvent) {
}

func (n *noopNotifier) NotifyEnrichTaskResponse(ctx context.Context, ev *event.EnrichTaskResponseEvent) {
}

func (n *noopNotifier) NotifyError(ctx context.Context, ev *event.ErrorEvent) {}

func (uc *UseCases) HandleAlert(ctx context.Context, schema types.AlertSchema, alertData any) ([]*alert.Alert, error) {
	logger := logging.From(ctx)

	// Use no-op notifier for now (we'll add proper notification later)
	notifier := &noopNotifier{}

	// Execute alert pipeline (policy evaluation + metadata generation)
	pipelineResults, err := uc.ProcessAlertPipeline(ctx, schema, alertData, notifier)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to process alert pipeline")
	}

	// Process each alert result
	var results []*alert.Alert
	for _, alertResult := range pipelineResults {
		processedAlert := alertResult.Alert

		// Find similar alerts for thread grouping
		now := clock.Now(ctx)
		begin := now.Add(-24 * time.Hour)
		end := now

		recentAlerts, err := uc.repository.GetAlertsBySpan(ctx, begin, end)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to get recent alerts")
		}

		// Filter alerts that are not bound to tickets
		var unboundAlerts []*alert.Alert
		for _, recentAlert := range recentAlerts {
			if recentAlert.TicketID == types.EmptyTicketID && len(recentAlert.Embedding) > 0 {
				unboundAlerts = append(unboundAlerts, recentAlert)
			}
		}

		var existingAlert *alert.Alert
		var bestSimilarity float64

		// Search for the alert with the closest embedding (similarity >= 0.99)
		if len(unboundAlerts) > 0 {
			for _, unboundAlert := range unboundAlerts {
				similarity := processedAlert.CosineSimilarity(unboundAlert.Embedding)
				if similarity >= 0.99 && similarity > bestSimilarity {
					bestSimilarity = similarity
					existingAlert = unboundAlert
				}
			}
		}

		// Post to Slack
		if existingAlert != nil && existingAlert.HasSlackThread() && uc.slackService != nil {
			// Post to existing thread
			thread := uc.slackService.NewThread(*existingAlert.SlackThread)
			if err := thread.PostAlert(ctx, processedAlert); err != nil {
				return nil, goerr.Wrap(err, "failed to post alert to existing thread")
			}
			processedAlert.SlackThread = existingAlert.SlackThread
			logger.Info("alert posted to existing thread", "alert", processedAlert, "existing_alert", existingAlert, "similarity", bestSimilarity)
		} else if uc.slackService != nil {
			// Post to new thread
			newThread, err := uc.slackService.PostAlert(ctx, processedAlert)
			if err != nil {
				return nil, goerr.Wrap(err, "failed to post alert")
			}
			if newThread != nil {
				processedAlert.SlackThread = newThread.Entity()
			}
			logger.Info("alert posted to new thread", "alert", processedAlert)
		}

		// Save to database
		if err := uc.repository.PutAlert(ctx, *processedAlert); err != nil {
			return nil, goerr.Wrap(err, "failed to put alert")
		}
		logger.Info("alert created", "alert", processedAlert)

		results = append(results, processedAlert)
	}

	return results, nil
}

// GetUnboundAlertsFiltered returns unbound alerts filtered by similarity threshold and keyword
func (uc *UseCases) GetUnboundAlertsFiltered(ctx context.Context, threshold *float64, keyword *string, ticketID *types.TicketID, offset, limit int) ([]*alert.Alert, int, error) {
	var candidateAlerts []*alert.Alert
	var err error

	// Step 1: Get candidate alerts - always start with unbound alerts for salvage search
	if threshold != nil && ticketID != nil && *ticketID != types.EmptyTicketID {
		// Get ticket for similarity comparison
		ticketObj, err := uc.repository.GetTicket(ctx, *ticketID)
		if err != nil {
			return nil, 0, goerr.Wrap(err, "failed to get ticket for similarity filtering")
		}

		// Get ALL unbound alerts first, then filter by similarity
		allUnboundAlerts, err := uc.repository.GetAlertWithoutTicket(ctx, 0, 0) // Get all unbound alerts
		if err != nil {
			return nil, 0, goerr.Wrap(err, "failed to get unbound alerts")
		}

		if len(ticketObj.Embedding) > 0 {
			// Filter unbound alerts by similarity threshold
			for _, a := range allUnboundAlerts {
				// Only check alerts that have embeddings
				if len(a.Embedding) > 0 {
					similarity := a.CosineSimilarity(ticketObj.Embedding)
					if float64(similarity) >= *threshold {
						candidateAlerts = append(candidateAlerts, a)
					}
				}
			}
		} else {
			candidateAlerts = allUnboundAlerts
		}
	} else {
		// Get all unbound alerts as candidates
		candidateAlerts, err = uc.repository.GetAlertWithoutTicket(ctx, 0, 0) // Get all for filtering
		if err != nil {
			return nil, 0, goerr.Wrap(err, "failed to get unbound alerts")
		}
	}

	// Step 2: Apply keyword filter to candidate alerts
	var filteredAlerts []*alert.Alert
	if keyword != nil && *keyword != "" {
		for _, a := range candidateAlerts {
			// Convert data to JSON string for keyword search
			dataBytes, err := json.Marshal(a.Data)
			if err != nil {
				continue
			}
			dataStr := string(dataBytes)

			// Check if keyword exists in title, description, or data
			if containsIgnoreCase(a.Title, *keyword) ||
				containsIgnoreCase(a.Description, *keyword) ||
				containsIgnoreCase(dataStr, *keyword) {
				filteredAlerts = append(filteredAlerts, a)
			}
		}
	} else {
		// No keyword filter, use all candidates
		filteredAlerts = candidateAlerts
	}

	// Step 3: Calculate total count from fully filtered results
	totalCount := len(filteredAlerts)

	// Step 4: Apply pagination to the filtered results
	start := offset
	if start > len(filteredAlerts) {
		start = len(filteredAlerts)
	}

	end := start + limit
	if limit > 0 && end > len(filteredAlerts) {
		end = len(filteredAlerts)
	}
	if limit == 0 {
		end = len(filteredAlerts)
	}

	result := filteredAlerts[start:end]

	return result, totalCount, nil
}

// BindAlertsToTicket binds multiple alerts to a ticket, recalculates embedding, and updates Slack display
func (uc *UseCases) BindAlertsToTicket(ctx context.Context, ticketID types.TicketID, alertIDs []types.AlertID) error {
	// Bind alerts to ticket (repository handles bidirectional binding)
	err := uc.repository.BindAlertsToTicket(ctx, alertIDs, ticketID)
	if err != nil {
		return goerr.Wrap(err, "failed to bind alerts to ticket")
	}

	// Get the updated ticket with new AlertIDs
	ticket, err := uc.repository.GetTicket(ctx, ticketID)
	if err != nil {
		return goerr.Wrap(err, "failed to get updated ticket")
	}

	// Recalculate ticket embedding with all bound alerts
	if err := ticket.CalculateEmbedding(ctx, uc.llmClient, uc.repository); err != nil {
		return goerr.Wrap(err, "failed to recalculate ticket embedding")
	}

	// Update ticket metadata based on existing title/description and new alert information
	if err := ticket.FillMetadata(ctx, uc.llmClient, uc.repository); err != nil {
		return goerr.Wrap(err, "failed to update ticket metadata with new alert information")
	}

	// Save the updated ticket with new embedding and metadata
	if err := uc.repository.PutTicket(ctx, *ticket); err != nil {
		return goerr.Wrap(err, "failed to save ticket with updated embedding and metadata")
	}

	// Update Slack display for both ticket and individual alerts (using updated metadata)
	// Update ticket display if it has a Slack thread
	if ticket.HasSlackThread() {
		// Get all alerts bound to the ticket for display
		alerts, err := uc.repository.BatchGetAlerts(ctx, ticket.AlertIDs)
		if err != nil {
			logging.From(ctx).Warn("failed to get alerts for Slack update", "error", err, "ticket_id", ticketID)
		} else if uc.slackService != nil {
			thread := uc.slackService.NewThread(*ticket.SlackThread)
			if _, err := thread.PostTicket(ctx, ticket, alerts); err != nil {
				// Log error but don't fail the operation
				logging.From(ctx).Warn("failed to update Slack thread after binding alerts", "error", err, "ticket_id", ticketID)
			}
		}
	}

	// Update individual alert displays in their respective threads
	boundAlerts, err := uc.repository.BatchGetAlerts(ctx, alertIDs)
	if err != nil {
		logging.From(ctx).Warn("failed to get bound alerts for individual Slack updates", "error", err, "alert_ids", alertIDs)
	} else {
		for _, alert := range boundAlerts {
			if alert.HasSlackThread() && uc.slackService != nil {
				alertThread := uc.slackService.NewThread(*alert.SlackThread)
				if err := alertThread.UpdateAlert(ctx, *alert); err != nil {
					// Log error but don't fail the operation
					logging.From(ctx).Warn("failed to update alert Slack display", "error", err, "alert_id", alert.ID)
				}
			}
		}
	}

	return nil
}

// containsIgnoreCase checks if substr exists in s (case insensitive)
func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// processGenAI processes GenAI configuration for the given alert
func (uc *UseCases) processGenAI(ctx context.Context, alert *alert.Alert) (any, error) {
	if alert.Metadata.GenAI == nil {
		return "", nil
	}

	if uc.promptService == nil {
		return "", goerr.New("prompt service not configured")
	}

	genaiConfig := alert.Metadata.GenAI

	// Generate prompt using PromptService
	prompt, err := uc.promptService.GeneratePrompt(ctx, genaiConfig.Prompt, alert)
	if err != nil {
		return "", goerr.Wrap(err, "failed to generate prompt", goerr.V("prompt", genaiConfig.Prompt))
	}

	var options []gollem.SessionOption
	if genaiConfig.Format == types.GenAIContentFormatJSON {
		options = append(options, gollem.WithSessionContentType(gollem.ContentTypeJSON))
	}

	// Query LLM with JSON response format
	session, err := uc.llmClient.NewSession(ctx, options...)
	if err != nil {
		return "", goerr.Wrap(err, "failed to create LLM session")
	}

	response, err := session.GenerateContent(ctx, gollem.Text(prompt))
	if err != nil {
		return "", goerr.Wrap(err, "failed to query LLM", goerr.V("prompt", prompt))
	}

	var responseText string
	if len(response.Texts) > 0 {
		responseText = response.Texts[0]
	}

	var responseData any = responseText
	logger := logging.From(ctx)

	// Parse JSON response if format is JSON
	if genaiConfig.Format == types.GenAIContentFormatJSON {
		var parsedResponse any
		if err := json.Unmarshal([]byte(responseText), &parsedResponse); err != nil {
			// If JSON parsing fails, return raw string
			logger.Warn("failed to parse LLM response as JSON", "text", responseText)
		} else {
			responseData = parsedResponse
		}
	}

	logger.Info("Got GenAI response", "response", responseData)
	return responseData, nil
}

// handleNotice handles notice creation and simple notification
func (uc *UseCases) handleNotice(ctx context.Context, alert *alert.Alert, channel string, llmResponse *alert.GenAIResponse) error {
	logger := logging.From(ctx)

	// Create notice
	notice := &notice.Notice{
		ID:        types.NewNoticeID(),
		Alert:     *alert,
		CreatedAt: clock.Now(ctx),
		Escalated: false,
	}

	// Store notice in repository
	if err := uc.repository.CreateNotice(ctx, notice); err != nil {
		return goerr.Wrap(err, "failed to create notice")
	}

	// Send simple notification to Slack
	if uc.slackService != nil {
		slackTS, err := uc.sendSimpleNotification(ctx, notice, channel, llmResponse)
		if err != nil {
			logger.Warn("failed to send simple notification", "error", err, "notice_id", notice.ID)
		} else {
			// Update notice with Slack timestamp
			notice.SlackTS = slackTS
			if err := uc.repository.UpdateNotice(ctx, notice); err != nil {
				logger.Warn("failed to update notice with slack timestamp", "error", err, "notice_id", notice.ID)
			}
		}
	}

	logger.Info("notice created", "notice_id", notice.ID, "alert_id", alert.ID)
	return nil
}

// sendSimpleNotification sends a simple notification to Slack
func (uc *UseCases) sendSimpleNotification(ctx context.Context, notice *notice.Notice, channel string, llmResponse *alert.GenAIResponse) (string, error) {
	if uc.slackService == nil {
		return "", goerr.New("slack service not available")
	}

	alert := &notice.Alert

	// Create simple message with only title for main channel
	var mainMessage string
	if alert.Metadata.Title != "" {
		mainMessage = "🔔 " + alert.Metadata.Title
	} else {
		mainMessage = "🔔 Security Notice"
	}

	// Resolve target channel (use default if empty)
	targetChannel := channel
	if targetChannel == "" {
		targetChannel = uc.slackService.DefaultChannelID()
	}

	// Post main notice message
	timestamp, err := uc.slackService.PostNotice(ctx, targetChannel, mainMessage, notice.ID)
	if err != nil {
		return "", goerr.Wrap(err, "failed to post notice to Slack", goerr.V("channel", targetChannel))
	}

	// Post detailed information in thread
	if err := uc.slackService.PostNoticeThreadDetails(ctx, targetChannel, timestamp, alert, llmResponse); err != nil {
		// Log error but don't fail the main operation
		logging.From(ctx).Warn("failed to post notice thread details", "error", err, "channel", targetChannel)
	}

	return timestamp, nil
}

// EscalateNotice escalates a notice to a full alert
func (uc *UseCases) EscalateNotice(ctx context.Context, noticeID types.NoticeID) error {
	logger := logging.From(ctx)

	// Get notice from repository
	notice, err := uc.repository.GetNotice(ctx, noticeID)
	if err != nil {
		return goerr.Wrap(err, "failed to get notice", goerr.V("notice_id", noticeID))
	}

	// Mark notice as escalated
	notice.Escalated = true
	if err := uc.repository.UpdateNotice(ctx, notice); err != nil {
		return goerr.Wrap(err, "failed to update notice", goerr.V("notice_id", noticeID))
	}

	// Post escalated alert to Slack
	escalatedAlert := notice.Alert
	if uc.slackService != nil {
		newThread, err := uc.slackService.PostAlert(ctx, &escalatedAlert)
		if err != nil {
			return goerr.Wrap(err, "failed to post escalated alert", goerr.V("notice_id", noticeID))
		}
		if newThread != nil {
			escalatedAlert.SlackThread = newThread.Entity()
		}
		logger.Info("escalated alert posted to new thread", "notice_id", noticeID, "alert_id", escalatedAlert.ID)
	}

	// Store escalated alert
	if err := uc.repository.PutAlert(ctx, escalatedAlert); err != nil {
		return goerr.Wrap(err, "failed to put escalated alert", goerr.V("notice_id", noticeID))
	}

	logger.Info("notice escalated to alert", "notice_id", noticeID, "alert_id", escalatedAlert.ID)
	return nil
}
