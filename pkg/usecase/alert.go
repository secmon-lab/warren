package usecase

import (
	"context"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

func (uc *UseCases) HandleAlert(ctx context.Context, schema types.AlertSchema, alertData any) ([]*alert.Alert, error) {
	logger := logging.From(ctx)

	var result struct {
		Alert []alert.Metadata `json:"alert"`
	}
	if err := uc.policyClient.Query(ctx, "data.alert."+string(schema), alertData, &result); err != nil {
		return nil, goerr.Wrap(err, "failed to query policy", goerr.V("schema", schema), goerr.V("alert", alertData))
	}

	logger.Info("policy query result", "input", alertData, "output", result)

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
