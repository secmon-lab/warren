package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/opaq"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/auth"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/prompt"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/llm"
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
		alert := alert.New(ctx, schema, a)
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

	newAlert, err := uc.generateAlertMetadata(ctx, alert)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate alert metadata")
	}
	alert = *newAlert

	if uc.embeddingClient != nil {
		rawData, err := json.Marshal(alert.Data)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to marshal alert data")
		}
		embedding, err := uc.embeddingClient.Embeddings(ctx, []string{string(rawData)}, 256)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to embed alert data")
		}
		if len(embedding) == 0 {
			return nil, goerr.New("failed to embed alert data")
		}
		logger.Info("alert embedding", "embedding", embedding[0], "alert", alert.ID)
		alert.Embedding = embedding[0]
	}

	// Check if the alert is similar to any existing alerts
	// NOTE: Disable similarity merger for now

	// Post new alert to Slack and save the alert with Slack channel and message ID
	thread, err := uc.slackService.PostAlert(ctx, alert)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to post alert", goerr.V("alert", alert))
	}
	alert.SlackThread = &slack.Thread{
		ChannelID: thread.ChannelID(),
		ThreadID:  thread.ThreadID(),
	}

	if err := uc.repository.PutAlert(ctx, alert); err != nil {
		return nil, goerr.Wrap(err, "failed to put alert", goerr.V("alert", alert))
	}
	logger.Info("alert created", "alert", alert)

	return &alert, nil
}

func (uc *UseCases) generateAlertMetadata(ctx context.Context, alert alert.Alert) (*alert.Alert, error) {
	logger := logging.From(ctx)
	p, err := prompt.BuildMetaPrompt(ctx, alert)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to build meta prompt")
	}

	ssn, err := uc.llmClient.NewSession(ctx, nil)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create LLM session")
	}

	var result *prompt.MetaPromptResult
	for i := 0; i < 3 && result == nil; i++ {
		result, err = llm.Ask[prompt.MetaPromptResult](ctx, ssn, p)
		if err != nil {
			if goerr.HasTag(err, errs.TagInvalidLLMResponse) {
				logger.Warn("invalid LLM response, retry to generate alert metadata", "error", err)
				p = fmt.Sprintf("invalid format, please try again: %s", err.Error())
				continue
			}
			return nil, goerr.Wrap(err, "failed to ask chat")
		}
	}
	if result == nil {
		return nil, goerr.New("failed to generate alert metadata")
	}

	if alert.Title == "" {
		alert.Title = result.Title
	}

	if alert.Description == "" {
		alert.Description = result.Description
	}

	for _, resAttr := range result.Attrs {
		found := false
		for _, aAttr := range alert.Attributes {
			if aAttr.Value == resAttr.Value {
				found = true
				break
			}
		}
		if !found {
			resAttr.Auto = true
			alert.Attributes = append(alert.Attributes, resAttr)
		}
	}

	// Calcluating embedding for the alert
	if uc.embeddingClient == nil {
		panic("embedding client is not set")
	}
	rawData, err := json.Marshal(alert.Data)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to marshal alert data")
	}
	embedding, err := uc.embeddingClient.Embeddings(ctx, []string{string(rawData)}, 256)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to embed alert data")
	}
	if len(embedding) == 0 {
		return nil, goerr.New("failed to embed alert data")
	}
	alert.Embedding = embedding[0]

	return &alert, nil
}
