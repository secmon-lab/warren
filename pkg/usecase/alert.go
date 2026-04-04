package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/notice"
	"github.com/secmon-lab/warren/pkg/domain/types"
	notifierSvc "github.com/secmon-lab/warren/pkg/service/notifier"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	slackSDK "github.com/slack-go/slack"
)

// putAlertWithLogging saves an alert and logs the full alert data on error
func putAlertWithLogging(ctx context.Context, repo interfaces.Repository, alert *alert.Alert) error {
	if err := repo.PutAlert(ctx, *alert); err != nil {
		logger := logging.From(ctx)
		if data, jsonErr := json.Marshal(alert); jsonErr == nil {
			logger.Error("failed to save alert", "error", err, "alert", string(data))
		}
		return goerr.Wrap(err, "failed to put alert", goerr.V("alert", alert))
	}
	return nil
}

func (uc *UseCases) HandleAlert(ctx context.Context, schema types.AlertSchema, alertData any) ([]*alert.Alert, error) {
	logger := logging.From(ctx)

	// Circuit breaker: check throttle before pipeline (read-only, does not consume slot)
	if uc.cbService != nil && uc.cbService.IsEnabled() {
		throttleResult, err := uc.cbService.CheckThrottle(ctx)
		if err != nil {
			// On throttle failure, fall through to normal processing (don't lose alerts)
			errutil.Handle(ctx, goerr.Wrap(err, "circuit breaker throttle check failed, processing normally"))
		} else if !throttleResult.Allowed {
			// Alert is throttled: enqueue it
			qa, enqueueErr := uc.cbService.EnqueueAlert(ctx, schema, alertData, "", "")
			if enqueueErr != nil {
				// If queueing fails, fall through to normal processing
				errutil.Handle(ctx, goerr.Wrap(enqueueErr, "failed to enqueue throttled alert, processing normally"))
			} else {
				logger.Info("alert throttled and queued", "queued_alert_id", qa.ID, "schema", schema)

				// Send Slack @channel warning if needed
				if throttleResult.ShouldNotify {
					uc.sendThrottleWarning(ctx)
				}

				return nil, nil
			}
		}
	}

	// Create notifier based on whether Slack is available
	// If Slack is available, use SlackNotifier which buffers events
	// Otherwise, use ConsoleNotifier which outputs immediately
	var pipelineNotifier interfaces.Notifier
	var slackNotifier *notifierSvc.SlackNotifier
	if uc.slackService != nil {
		slackNotifier = notifierSvc.NewSlackNotifier().(*notifierSvc.SlackNotifier)
		pipelineNotifier = slackNotifier
	} else {
		pipelineNotifier = uc.consoleNotifier
	}

	// Execute alert pipeline (policy evaluation + metadata generation)
	pipelineResults, err := uc.ProcessAlertPipeline(ctx, schema, alertData, pipelineNotifier)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to process alert pipeline")
	}

	// Circuit breaker: acquire slots for all pipeline results (ingest-passed alerts).
	// All alerts that pass ingest consume slots regardless of triage result (discard/notice/alert).
	if len(pipelineResults) > 0 && uc.cbService != nil && uc.cbService.IsEnabled() {
		for i, alertResult := range pipelineResults {
			result, err := uc.cbService.AcquireSlot(ctx)
			if err != nil {
				// On error, queue the alert to avoid losing it
				errutil.Handle(ctx, goerr.Wrap(err, "failed to acquire throttle slot, queueing alert"))
				_, enqueueErr := uc.cbService.EnqueueAlert(ctx, schema, alertData, alertResult.Alert.Title, alertResult.Alert.Channel)
				if enqueueErr != nil {
					errutil.Handle(ctx, goerr.Wrap(enqueueErr, "failed to enqueue alert after slot error"))
				}
				return nil, nil
			}
			if !result.Allowed {
				// Slot denied: queue the raw alert and stop processing all results
				logger.Info("alert throttled after pipeline, queueing",
					"acquired_slots", i,
					"total_results", len(pipelineResults),
					"alert_id", alertResult.Alert.ID)

				_, enqueueErr := uc.cbService.EnqueueAlert(ctx, schema, alertData, alertResult.Alert.Title, alertResult.Alert.Channel)
				if enqueueErr != nil {
					errutil.Handle(ctx, goerr.Wrap(enqueueErr, "failed to enqueue throttled alert"))
				}

				if result.ShouldNotify {
					uc.sendThrottleWarning(ctx)
				}

				return nil, nil
			}
		}
	}

	// Process each alert result
	var results []*alert.Alert
	for _, alertResult := range pipelineResults {
		processedAlert := alertResult.Alert
		commitResult := alertResult.TriageResult

		// Determine publish type (default to alert if not specified)
		publishType := types.PublishTypeAlert
		if commitResult != nil && commitResult.Publish != "" {
			publishType = commitResult.Publish
		}

		logger.Info("processing alert based on publish type",
			"alert_id", processedAlert.ID,
			"publish_type", publishType,
			"channel", processedAlert.Channel)

		// Handle based on publish type (slots already acquired above)
		switch publishType {
		case types.PublishTypeDiscard:
			// Discard: do nothing, just log. Slot already consumed for counting purposes.
			logger.Info("alert discarded by triage policy", "alert_id", processedAlert.ID)

		case types.PublishTypeNotice:
			if err := uc.handleNotice(ctx, processedAlert, processedAlert.Channel, nil, pipelineNotifier); err != nil {
				return nil, goerr.Wrap(err, "failed to handle notice")
			}

		case types.PublishTypeAlert:
			// Alert: full alert processing with Slack and database
			// Post to Slack and flush pipeline events
			if slackNotifier != nil {
				// Use SlackNotifier to publish alert and flush pipeline events together
				// This avoids duplicate posting of alert content
				newThread, err := slackNotifier.PublishAlert(ctx, uc.slackService, processedAlert)
				if err != nil {
					return nil, goerr.Wrap(err, "failed to publish alert with pipeline events")
				}
				if newThread != nil {
					processedAlert.SlackThread = newThread.Entity()
				}
				logger.Info("alert published with pipeline events",
					"alert_id", processedAlert.ID,
					"channel", processedAlert.SlackThread.ChannelID,
					"thread", processedAlert.SlackThread.ThreadID)
			}

			// Save to database
			if err := putAlertWithLogging(ctx, uc.repository, processedAlert); err != nil {
				return nil, err
			}
			logger.Info("alert created", "alert", processedAlert)

			// Add alert to results
			results = append(results, processedAlert)

		default:
			logger.Warn("unknown publish type, defaulting to alert", "publish_type", publishType)
			// Default to alert behavior
			if uc.slackService != nil {
				newThread, err := uc.slackService.PostAlert(ctx, processedAlert)
				if err != nil {
					return nil, goerr.Wrap(err, "failed to post alert")
				}
				if newThread != nil {
					processedAlert.SlackThread = newThread.Entity()
				}
			}
			if err := putAlertWithLogging(ctx, uc.repository, processedAlert); err != nil {
				return nil, err
			}

			// Add alert to results
			results = append(results, processedAlert)
		}
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

// BindAlertsToTicket binds multiple alerts to a ticket and updates Slack display
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

	// Update Slack display for both ticket and individual alerts (using existing metadata)
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

// handleNotice handles notice creation and simple notification
func (uc *UseCases) handleNotice(ctx context.Context, alert *alert.Alert, channel string, llmResponse *alert.GenAIResponse, notifier interfaces.Notifier) error {
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
		if data, jsonErr := json.Marshal(notice); jsonErr == nil {
			logger.Error("failed to create notice", "error", err, "notice", string(data))
		}
		return goerr.Wrap(err, "failed to create notice", goerr.V("notice", notice))
	}

	// Send simple notification to Slack and flush pipeline events if SlackNotifier
	if uc.slackService != nil {
		slackTS, err := uc.sendSimpleNotification(ctx, notice, channel, llmResponse, notifier)
		if err != nil {
			logger.Warn("failed to send simple notification", "error", err, "notice_id", notice.ID)
		} else {
			// Update notice with Slack timestamp
			notice.SlackTS = slackTS
			if err := uc.repository.UpdateNotice(ctx, notice); err != nil {
				if data, jsonErr := json.Marshal(notice); jsonErr == nil {
					logger.Warn("failed to update notice with slack timestamp", "error", err, "notice", string(data))
				} else {
					logger.Warn("failed to update notice with slack timestamp", "error", err, "notice_id", notice.ID)
				}
			}
		}
	}

	logger.Info("notice created", "notice_id", notice.ID, "alert_id", alert.ID)
	return nil
}

// sendSimpleNotification sends a simple notification to Slack
func (uc *UseCases) sendSimpleNotification(ctx context.Context, notice *notice.Notice, channel string, llmResponse *alert.GenAIResponse, notifier interfaces.Notifier) (string, error) {
	if uc.slackService == nil {
		return "", goerr.New("slack service not available")
	}

	// Resolve target channel (use default if empty)
	targetChannel := channel
	if targetChannel == "" {
		targetChannel = uc.slackService.DefaultChannelID()
	}

	// Use SlackNotifier.PublishNotice if available to flush pipeline events
	if slackNotifier, ok := notifier.(*notifierSvc.SlackNotifier); ok {
		timestamp, err := slackNotifier.PublishNotice(ctx, uc.slackService, notice, targetChannel, llmResponse)
		if err != nil {
			return "", goerr.Wrap(err, "failed to publish notice to Slack", goerr.V("channel", targetChannel))
		}
		return timestamp, nil
	}

	// Fallback: post notice without pipeline events (for non-Slack notifiers)
	alertData := &notice.Alert
	var mainMessage string
	if alertData.Title != "" {
		mainMessage = "🔔 " + alertData.Title
	} else {
		mainMessage = "🔔 Security Notice"
	}

	timestamp, err := uc.slackService.PostNotice(ctx, targetChannel, mainMessage, notice.ID)
	if err != nil {
		return "", goerr.Wrap(err, "failed to post notice to Slack", goerr.V("channel", targetChannel))
	}

	if err := uc.slackService.PostNoticeThreadDetails(ctx, targetChannel, timestamp, alertData, llmResponse); err != nil {
		logging.From(ctx).Warn("failed to post notice thread details", "error", err, "channel", targetChannel)
	}

	return timestamp, nil
}

// EscalateNotice escalates a notice to a full alert
func (uc *UseCases) EscalateNotice(ctx context.Context, noticeID types.NoticeID) (*alert.Alert, error) {
	logger := logging.From(ctx)

	// Get notice from repository
	notice, err := uc.repository.GetNotice(ctx, noticeID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get notice", goerr.V("notice_id", noticeID))
	}

	// Mark notice as escalated
	notice.Escalated = true
	if err := uc.repository.UpdateNotice(ctx, notice); err != nil {
		if data, jsonErr := json.Marshal(notice); jsonErr == nil {
			logger.Error("failed to update notice", "error", err, "notice", string(data))
		}
		return nil, goerr.Wrap(err, "failed to update notice", goerr.V("notice_id", noticeID), goerr.V("notice", notice))
	}

	// Post escalated alert to Slack
	escalatedAlert := notice.Alert
	if uc.slackService != nil {
		newThread, err := uc.slackService.PostAlert(ctx, &escalatedAlert)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to post escalated alert", goerr.V("notice_id", noticeID))
		}
		if newThread != nil {
			escalatedAlert.SlackThread = newThread.Entity()
		}
		logger.Info("escalated alert posted to new thread", "notice_id", noticeID, "alert_id", escalatedAlert.ID)
	}

	// Store escalated alert
	if err := putAlertWithLogging(ctx, uc.repository, &escalatedAlert); err != nil {
		return nil, goerr.Wrap(err, "failed to put escalated alert", goerr.V("notice_id", noticeID))
	}

	logger.Info("notice escalated to alert", "notice_id", noticeID, "alert_id", escalatedAlert.ID)
	return &escalatedAlert, nil
}

// DeclineAlerts declines multiple alerts by updating their status to declined.
func (uc *UseCases) DeclineAlerts(ctx context.Context, alertIDs []types.AlertID) ([]*alert.Alert, error) {
	alerts, err := uc.repository.BatchGetAlerts(ctx, alertIDs)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to batch get alerts")
	}

	if len(alerts) != len(alertIDs) {
		foundIDs := make(map[types.AlertID]bool, len(alerts))
		for _, a := range alerts {
			foundIDs[a.ID] = true
		}
		for _, id := range alertIDs {
			if !foundIDs[id] {
				return nil, goerr.New("alert not found", goerr.V("alert_id", id))
			}
		}
	}

	var results []*alert.Alert
	for _, a := range alerts {
		if err := uc.repository.UpdateAlertStatus(ctx, a.ID, alert.AlertStatusDeclined); err != nil {
			return nil, goerr.Wrap(err, "failed to decline alert", goerr.V("alert_id", a.ID))
		}
		a.Status = alert.AlertStatusDeclined
		results = append(results, a)
	}

	return results, nil
}

// sendThrottleWarning sends a @channel warning to Slack when alert throttle is activated.
func (uc *UseCases) sendThrottleWarning(ctx context.Context) {
	if uc.slackService == nil {
		return
	}

	channelID := uc.slackService.DefaultChannelID()
	queueURL := uc.frontendURL + "/queue"
	message := fmt.Sprintf("<!channel> :warning: Alert circuit breaker activated. Incoming alerts are being queued due to rate limiting. <%s|Manage queued alerts>", queueURL)

	_, _, err := uc.slackService.GetClient().PostMessageContext(
		ctx,
		channelID,
		slackSDK.MsgOptionText(message, false),
	)
	if err != nil {
		errutil.Handle(ctx, goerr.Wrap(err, "failed to send circuit breaker warning to Slack", goerr.V("channel", channelID)))
	}
}

// ReprocessQueuedAlert creates a background job to reprocess a queued alert.
func (uc *UseCases) ReprocessQueuedAlert(ctx context.Context, queuedAlertID types.QueuedAlertID) (*alert.ReprocessJob, error) {
	if uc.cbService == nil {
		return nil, goerr.New("circuit breaker service not configured")
	}

	return uc.cbService.ReprocessAlert(ctx, queuedAlertID, func(bgCtx context.Context, schema types.AlertSchema, data any) error {
		_, err := uc.HandleAlert(bgCtx, schema, data)
		return err
	})
}

// DiscardQueuedAlerts removes queued alerts without processing.
func (uc *UseCases) DiscardQueuedAlerts(ctx context.Context, ids []types.QueuedAlertID) error {
	if uc.cbService == nil {
		return goerr.New("circuit breaker service not configured")
	}

	return uc.cbService.DiscardAlerts(ctx, ids)
}

// DiscardQueuedAlertsByFilter removes all queued alerts matching the keyword filter.
func (uc *UseCases) DiscardQueuedAlertsByFilter(ctx context.Context, keyword *string) (int, error) {
	if uc.cbService == nil {
		return 0, goerr.New("circuit breaker service not configured")
	}

	return uc.cbService.DiscardAlertsByFilter(ctx, keyword)
}

// ReprocessQueuedAlertsByFilter creates a batch job to reprocess all queued alerts matching the keyword filter.
func (uc *UseCases) ReprocessQueuedAlertsByFilter(ctx context.Context, keyword *string) (*alert.ReprocessBatchJob, error) {
	if uc.cbService == nil {
		return nil, goerr.New("circuit breaker service not configured")
	}

	return uc.cbService.ReprocessAlertsByFilter(ctx, keyword, func(bgCtx context.Context, schema types.AlertSchema, data any) error {
		_, err := uc.HandleAlert(bgCtx, schema, data)
		return err
	})
}
