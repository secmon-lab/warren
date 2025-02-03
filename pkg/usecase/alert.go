package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/vertexai/genai"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/interfaces"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/prompt"
	"github.com/secmon-lab/warren/pkg/service"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

func (uc *UseCases) HandleAlert(ctx context.Context, schema string, alertData any) error {
	logger := logging.From(ctx)

	var result model.PolicyResult
	if err := uc.policyClient.Query(ctx, "alert."+schema, alertData, &result); err != nil {
		return goerr.Wrap(err, "failed to query policy", goerr.V("schema", schema), goerr.V("alert", alertData))
	}

	logger.Info("policy query result", "input", alertData, "output", result)

	for _, a := range result.Alert {
		alert := model.NewAlert(ctx, schema, alertData, a)

		if err := uc.runWorkflow(ctx, alert); err != nil {
			return goerr.Wrap(err, "failed to handle alert", goerr.V("alert", a))
		}
	}

	return nil
}

func (uc *UseCases) runWorkflow(ctx context.Context, alert model.Alert) error {
	logger := logging.From(ctx)

	// Check if the alert is similar to any existing alerts
	similarAlert, err := uc.findSimilarAlert(ctx, alert)
	if err != nil {
		return goerr.Wrap(err, "failed to find similar alert")
	}
	if similarAlert != nil {
		logger.Info("alert merged", "parent", similarAlert, "merged", alert)

		alert.ParentID = similarAlert.ID
		alert.Status = model.AlertStatusMerged
		if err := uc.repository.PutAlert(ctx, alert); err != nil {
			return goerr.Wrap(err, "failed to put alert", goerr.V("alert", alert))
		}

		return nil
	}

	// Post new alert to Slack and save the alert with Slack channel and message ID
	threadID, msgID, err := uc.slackService.PostAlert(ctx, alert)
	if err != nil {
		return goerr.Wrap(err, "failed to post alert", goerr.V("alert", alert))
	}
	alert.SlackChannel = threadID
	alert.SlackMessageID = msgID

	if err := uc.repository.PutAlert(ctx, alert); err != nil {
		return goerr.Wrap(err, "failed to put alert", goerr.V("alert", alert))
	}
	logger.Info("alert created", "alert", alert)

	ssn := uc.geminiStartChat()

	prePrompt, err := prompt.BuildInitPrompt(alert)
	if err != nil {
		return goerr.Wrap(err, "failed to build init prompt")
	}

	for i := 0; i < uc.loopLimit; i++ {
		actionPrompt, err := planAction(ctx, ssn, prePrompt, uc.actionService)
		if err != nil {
			return goerr.Wrap(err, "failed to plan action")
		}
		logger.Info("action planned", "action", actionPrompt)

		actionResult, err := uc.actionService.Execute(ctx, uc.slackService, actionPrompt.Action, ssn, actionPrompt.Args)
		if err != nil {
			return goerr.Wrap(err, "failed to execute action")
		}

		logger.Info("action executed", "action", actionResult)

		prePrompt = fmt.Sprintf("Here is the result of the action:\n\n```\n%s\n```", actionResult)
	}

	return nil
}

func planAction(ctx context.Context, ssn interfaces.GenAIChatSession, prePrompt string, actionSvc *service.ActionService) (*prompt.ActionPromptResult, error) {
	mainPrompt, err := prompt.BuildActionPrompt(actionSvc.Spec())
	if err != nil {
		return nil, goerr.Wrap(err, "failed to build action prompt")
	}

	resp, err := ssn.SendMessage(ctx, genai.Text(prePrompt), genai.Text(mainPrompt))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to send message")
	}
	eb := goerr.NewBuilder(goerr.V("prompt", mainPrompt), goerr.V("response", resp))

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, eb.New("no action prompt result")
	}

	text, ok := resp.Candidates[0].Content.Parts[0].(genai.Text)
	if !ok || text == "" {
		return nil, eb.New("no action prompt result")
	}

	var result prompt.ActionPromptResult
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, eb.Wrap(err, "failed to unmarshal action prompt result", goerr.V("text", text))
	}

	return &result, nil
}

func (uc *UseCases) findSimilarAlert(ctx context.Context, alert model.Alert) (*model.Alert, error) {
	oldest := alert.CreatedAt.Add(-24 * time.Hour)
	alerts, err := uc.repository.FetchLatestAlerts(ctx, oldest, 100)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to fetch latest alerts")
	}

	prompt, err := prompt.BuildAggregatePrompt(alert, alerts)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to build aggregate prompt")
	}

	ssn := uc.geminiStartChat()
	resp, err := ssn.SendMessage(ctx, genai.Text(prompt))
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

	alertID := model.AlertID(strings.TrimSpace(string(text)))

	for _, candidate := range alerts {
		if candidate.ID == alertID {
			return &candidate, nil
		}
	}

	return nil, nil
}
