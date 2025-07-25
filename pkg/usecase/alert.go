package usecase

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/opaq"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

func (uc *UseCases) HandleAlert(ctx context.Context, schema types.AlertSchema, alertData any) ([]*alert.Alert, error) {
	logger := logging.From(ctx)

	var result alert.QueryOutput
	query := "data.alert." + string(schema)
	hook := func(ctx context.Context, loc opaq.PrintLocation, msg string) error {
		logging.From(ctx).Debug("[rego.print] "+msg, "location", loc)
		return nil
	}
	if err := uc.policyClient.Query(ctx, query, alertData, &result, opaq.WithPrintHook(hook)); err != nil {
		return nil, goerr.Wrap(err, "failed to query policy", goerr.V("schema", schema), goerr.V("alert", alertData))
	}

	logger.Info("policy query result", "input", alertData, "output", result, "query", query)

	var results []*alert.Alert
	for _, a := range result.Alert {
		alert := alert.New(ctx, schema, alertData, a)
		if alert.Data == nil {
			alert.Data = alertData
		}

		newAlert, err := uc.handleAlert(ctx, alert)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to handle alert", goerr.V("alert", a))
		}
		results = append(results, newAlert)
	}

	return results, nil
}

func (uc *UseCases) handleAlert(ctx context.Context, newAlert alert.Alert) (*alert.Alert, error) {
	logger := logging.From(ctx)

	if err := newAlert.FillMetadata(ctx, uc.llmClient); err != nil {
		return nil, goerr.Wrap(err, "failed to fill alert metadata")
	}

	// Get alerts from the last 24 hours and search for those not bound to tickets
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
			similarity := newAlert.CosineSimilarity(unboundAlert.Embedding)
			if similarity >= 0.99 && similarity > bestSimilarity {
				bestSimilarity = similarity
				existingAlert = unboundAlert
			}
		}
	}

	if existingAlert != nil && existingAlert.HasSlackThread() {
		// Post to existing thread
		thread := uc.slackNotifier.NewThread(*existingAlert.SlackThread)
		if err := thread.PostAlert(ctx, &newAlert); err != nil {
			return nil, goerr.Wrap(err, "failed to post alert to existing thread", goerr.V("alert", newAlert), goerr.V("existing_alert", existingAlert))
		}
		newAlert.SlackThread = existingAlert.SlackThread
		logger.Info("alert posted to existing thread", "alert", newAlert, "existing_alert", existingAlert, "similarity", bestSimilarity)
	} else {
		// Post to new thread (normal posting)
		newThread, err := uc.slackNotifier.PostAlert(ctx, &newAlert)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to post alert", goerr.V("alert", newAlert))
		}
		if newThread != nil {
			newAlert.SlackThread = newThread.Entity()
		}
		logger.Info("alert posted to new thread", "alert", newAlert)
	}

	if err := uc.repository.PutAlert(ctx, newAlert); err != nil {
		return nil, goerr.Wrap(err, "failed to put alert", goerr.V("alert", newAlert))
	}
	logger.Info("alert created", "alert", newAlert)

	return &newAlert, nil
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
		} else {
			thread := uc.slackNotifier.NewThread(*ticket.SlackThread)
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
			if alert.HasSlackThread() {
				alertThread := uc.slackNotifier.NewThread(*alert.SlackThread)
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
