package usecase

import (
	"context"
	"errors"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/opaq"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/auth"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

func (uc *UseCases) HandleAlertWithAuth(ctx context.Context, schema types.AlertSchema, alertData any) ([]*alert.Alert, error) {
	authCtx := auth.BuildContext(ctx)

	var result struct {
		Allow bool `json:"allow"`
	}
	if err := uc.policyClient.Query(ctx, "data.auth", authCtx, &result); err != nil {
		if !errors.Is(err, opaq.ErrNoEvalResult) {
			return nil, goerr.Wrap(err, "failed to query policy", goerr.V("auth", authCtx))
		}
	}

	if !result.Allow {
		return nil, goerr.New("unauthorized", goerr.V("auth", authCtx), goerr.V("result", result))
	}

	return uc.HandleAlert(ctx, schema, alertData)
}

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

func (uc *UseCases) handleAlert(ctx context.Context, alert alert.Alert) (*alert.Alert, error) {
	logger := logging.From(ctx)

	// Post new alert to Slack and save the alert with Slack channel and message ID. It should be done before querying LLM.
	thread, err := uc.slackService.PostAlert(ctx, alert)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to post alert", goerr.V("alert", alert))
	}
	alert.SlackThread = thread.Entity()

	if err := uc.repository.PutAlert(ctx, alert); err != nil {
		return nil, goerr.Wrap(err, "failed to put alert", goerr.V("alert", alert))
	}
	logger.Info("alert created", "alert", alert)

	if err := alert.FillMetadata(ctx, uc.llmClient); err != nil {
		return nil, goerr.Wrap(err, "failed to fill alert metadata")
	}

	// Update posted alert in Slack.
	if _, err := uc.slackService.PostAlert(ctx, alert); err != nil {
		return nil, goerr.Wrap(err, "failed to post alert", goerr.V("alert", alert))
	}

	if err := uc.repository.PutAlert(ctx, alert); err != nil {
		return nil, goerr.Wrap(err, "failed to put alert", goerr.V("alert", alert))
	}
	logger.Info("alert created", "alert", alert)

	return &alert, nil
}
