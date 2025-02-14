package service

import (
	"context"
	"encoding/json"

	"cloud.google.com/go/vertexai/genai"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/interfaces"
)

func AskChat[T any](ctx context.Context, ssn interfaces.GenAIChatSession, prompt string) (*T, error) {
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

	var result T
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, goerr.Wrap(err, "failed to unmarshal text", goerr.V("text", text))
	}

	return &result, nil
}
