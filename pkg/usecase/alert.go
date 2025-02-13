package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	"cloud.google.com/go/vertexai/genai"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/prompt"
	"github.com/secmon-lab/warren/pkg/utils/authctx"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

func (uc *UseCases) HandleAlertWithAuth(ctx context.Context, schema string, alertData any) ([]*model.Alert, error) {
	authCtx := authctx.Build(ctx)
	if authCtx == nil {
		return nil, goerr.New("failed to build auth context")
	}

	var result model.PolicyAuth
	if err := uc.policyClient.Query(ctx, "data.auth", authCtx, &result); err != nil {
		return nil, goerr.Wrap(err, "failed to query policy", goerr.V("auth", authCtx))
	}

	if !result.Allow {
		return nil, goerr.New("unauthorized", goerr.V("auth", authCtx))
	}

	return uc.HandleAlert(ctx, schema, alertData)
}

func (uc *UseCases) HandleAlert(ctx context.Context, schema string, alertData any) ([]*model.Alert, error) {
	logger := logging.From(ctx)

	var result model.PolicyResult
	if err := uc.policyClient.Query(ctx, "data.alert."+schema, alertData, &result); err != nil {
		return nil, goerr.Wrap(err, "failed to query policy", goerr.V("schema", schema), goerr.V("alert", alertData))
	}

	logger.Info("policy query result", "input", alertData, "output", result)

	var results []*model.Alert
	for _, a := range result.Alert {
		alert := model.NewAlert(ctx, schema, a)
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

func (uc *UseCases) handleAlert(ctx context.Context, alert model.Alert) (*model.Alert, error) {
	logger := logging.From(ctx)

	// Check if the alert is similar to any existing alerts
	similarAlert, err := uc.findSimilarAlert(ctx, alert)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to find similar alert")
	}
	if similarAlert != nil {
		logger.Info("alert merged", "parent", similarAlert, "merged", alert)

		alert.ParentID = similarAlert.ID
		alert.Status = model.AlertStatusMerged
		if err := uc.repository.PutAlert(ctx, alert); err != nil {
			return nil, goerr.Wrap(err, "failed to put alert", goerr.V("alert", alert))
		}

		thread := uc.slackService.NewThread(alert)

		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetIndent("", "  ")
		if err := enc.Encode(alert.Data); err != nil {
			return nil, goerr.Wrap(err, "failed to encode alert data")
		}

		thread.AttachFile(ctx, "New merged alert", "alert.json", buf.Bytes())
		return nil, nil
	}

	// Post new alert to Slack and save the alert with Slack channel and message ID
	thread, err := uc.slackService.PostAlert(ctx, alert)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to post alert", goerr.V("alert", alert))
	}
	alert.SlackThread = &model.SlackThread{
		ChannelID: thread.ChannelID(),
		ThreadID:  thread.ThreadID(),
	}

	if err := uc.repository.PutAlert(ctx, alert); err != nil {
		return nil, goerr.Wrap(err, "failed to put alert", goerr.V("alert", alert))
	}
	logger.Info("alert created", "alert", alert)

	return &alert, nil
}

func (uc *UseCases) findSimilarAlert(ctx context.Context, alert model.Alert) (*model.Alert, error) {
	oldest := alert.CreatedAt.Add(-24 * time.Hour)
	alerts, err := uc.repository.FetchLatestAlerts(ctx, oldest, 100)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to fetch latest alerts")
	}

	p, err := prompt.BuildAggregatePrompt(alert, alerts)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to build aggregate prompt")
	}

	ssn := uc.geminiStartChat()
	resp, err := ssn.SendMessage(ctx, genai.Text(p))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to send message")
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, nil
	}

	text, ok := resp.Candidates[0].Content.Parts[0].(genai.Text)
	if !ok || text == "" {
		return nil, nil
	}

	var result prompt.AggregatePromptResult
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, goerr.Wrap(err, "failed to unmarshal aggregate prompt result", goerr.V("text", text))
	}

	if result.AlertID == "" {
		return nil, nil
	}

	alertID := model.AlertID(result.AlertID)

	for _, candidate := range alerts {
		if candidate.ID == alertID {
			return &candidate, nil
		}
	}

	return nil, nil
}
