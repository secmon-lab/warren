package llm

import (
	"context"
	"encoding/json"

	"cloud.google.com/go/vertexai/genai"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

type SendPrompt func(ctx context.Context, msg ...genai.Part) (*genai.GenerateContentResponse, error)

type config struct {
	maxRetry    int
	retryPrompt func(ctx context.Context, err error) string
}

type Option func(*config)

func WithMaxRetry(maxRetry int) Option {
	return func(c *config) {
		c.maxRetry = maxRetry
	}
}

func WithRetryPrompt(f func(ctx context.Context, err error) string) Option {
	return func(c *config) {
		c.retryPrompt = f
	}
}

func Ask[T any](ctx context.Context, f SendPrompt, prompt string, opts ...Option) (*T, error) {
	logger := logging.From(ctx)

	config := &config{
		maxRetry: 3,
		retryPrompt: func(ctx context.Context, err error) string {
			return "Invalid response. Please try again: " + err.Error()
		},
	}
	for _, opt := range opts {
		opt(config)
	}

	var response *T
	for i := 0; i < config.maxRetry && response == nil; i++ {
		resp, err := f(ctx, genai.Text(prompt))
		if err != nil {
			return nil, goerr.Wrap(err, "failed to send message")
		}

		if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
			ctx = msg.State(ctx, "💥 failed to get valid response from LLM, retry (%d/%d)\n> %s", i+1, config.maxRetry, err.Error())
			prompt = config.retryPrompt(ctx, err)
			continue
		}

		text, ok := resp.Candidates[0].Content.Parts[0].(genai.Text)
		if !ok || text == "" {
			ctx = msg.State(ctx, "💥 failed to get valid response from LLM, retry (%d/%d)\n> %s", i+1, config.maxRetry, err.Error())
			prompt = config.retryPrompt(ctx, err)
			continue
		}

		var result T
		if err := json.Unmarshal([]byte(text), &result); err != nil {
			logger.Debug("failed to unmarshal text", "text", text, "error", err)
			return nil, goerr.Wrap(err, "failed to unmarshal text", goerr.V("text", text), goerr.T(errs.TagInvalidLLMResponse))
		}

		response = &result
	}

	if response == nil {
		return nil, goerr.New("failed to get valid response from LLM", goerr.T(errs.TagInvalidLLMResponse))
	}

	return response, nil
}
