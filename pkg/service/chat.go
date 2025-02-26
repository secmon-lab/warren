package service

import (
	"context"
	"encoding/json"

	"cloud.google.com/go/vertexai/genai"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/interfaces"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

func AskChat[T any](ctx context.Context, ssn interfaces.LLMSession, prompt string) (*T, error) {
	logger := logging.From(ctx)
	resp, err := ssn.SendMessage(ctx, genai.Text(prompt))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to send message")
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, goerr.New("no response from LLM", goerr.T(model.ErrTagInvalidLLMResponse))
	}

	text, ok := resp.Candidates[0].Content.Parts[0].(genai.Text)
	if !ok || text == "" {
		return nil, goerr.New("no text data from LLM", goerr.T(model.ErrTagInvalidLLMResponse))
	}

	var result T
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		logger.Debug("failed to unmarshal text", "text", text, "error", err)
		return nil, goerr.Wrap(err, "failed to unmarshal text", goerr.V("text", text), goerr.T(model.ErrTagInvalidLLMResponse))
	}

	return &result, nil
}
