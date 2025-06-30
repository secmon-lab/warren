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

	if existingAlert != nil && existingAlert.SlackThread != nil {
		// Post to existing thread
		thread := uc.slackService.NewThread(*existingAlert.SlackThread)
		if err := thread.PostAlert(ctx, newAlert); err != nil {
			return nil, goerr.Wrap(err, "failed to post alert to existing thread", goerr.V("alert", newAlert), goerr.V("existing_alert", existingAlert))
		}
		newAlert.SlackThread = existingAlert.SlackThread
		logger.Info("alert posted to existing thread", "alert", newAlert, "existing_alert", existingAlert, "similarity", bestSimilarity)
	} else {
		// Post to new thread (normal posting)
		newThread, err := uc.slackService.PostAlert(ctx, newAlert)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to post alert", goerr.V("alert", newAlert))
		}
		newAlert.SlackThread = newThread.Entity()
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
	// Get unbound alerts
	alerts, err := uc.repository.GetAlertWithoutTicket(ctx, offset, limit)
	if err != nil {
		return nil, 0, goerr.Wrap(err, "failed to get unbound alerts")
	}

	// Filter by keyword if provided
	if keyword != nil && *keyword != "" {
		filteredAlerts := make([]*alert.Alert, 0)
		for _, a := range alerts {
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
		alerts = filteredAlerts
	}

	// Filter by similarity threshold if provided and ticketID is provided
	if threshold != nil && ticketID != nil && *ticketID != types.EmptyTicketID {
		ticketObj, err := uc.repository.GetTicket(ctx, *ticketID)
		if err != nil {
			return nil, 0, goerr.Wrap(err, "failed to get ticket for similarity filtering")
		}

		// Get ticket's alerts to use for similarity comparison
		if len(ticketObj.AlertIDs) > 0 {
			ticketAlerts, err := uc.repository.BatchGetAlerts(ctx, ticketObj.AlertIDs)
			if err != nil {
				return nil, 0, goerr.Wrap(err, "failed to get ticket alerts for similarity")
			}

			// Use the first alert's embedding for similarity
			if len(ticketAlerts) > 0 && len(ticketAlerts[0].Embedding) > 0 {
				similarAlerts, err := uc.repository.FindNearestAlerts(ctx, ticketAlerts[0].Embedding, 100)
				if err != nil {
					return nil, 0, goerr.Wrap(err, "failed to find similar alerts")
				}

				// Filter alerts by threshold and ensure they are unbound
				filteredAlerts := make([]*alert.Alert, 0)
				for _, a := range similarAlerts {
					if a.TicketID == types.EmptyTicketID {
						// Calculate similarity and filter by threshold
						similarity := ticketAlerts[0].CosineSimilarity(a.Embedding)
						if float64(similarity) >= *threshold {
							filteredAlerts = append(filteredAlerts, a)
						}
					}
				}
				alerts = filteredAlerts
			}
		}
	}

	// Get total count (for pagination, we'll just return the filtered count for now)
	totalCount := len(alerts)

	return alerts, totalCount, nil
}

// BindAlertsToTicket binds multiple alerts to a ticket
func (uc *UseCases) BindAlertsToTicket(ctx context.Context, ticketID types.TicketID, alertIDs []types.AlertID) error {
	// Bind alerts to ticket
	err := uc.repository.BatchBindAlertsToTicket(ctx, alertIDs, ticketID)
	if err != nil {
		return goerr.Wrap(err, "failed to bind alerts to ticket")
	}

	return nil
}

// containsIgnoreCase checks if substr exists in s (case insensitive)
func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
