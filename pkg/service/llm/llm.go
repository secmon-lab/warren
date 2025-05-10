package llm

import (
	"context"
	"encoding/json"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

type askConfig struct {
	maxRetry    int
	retryPrompt func(ctx context.Context, err error) string
}

type AskOption func(*askConfig)

func WithMaxRetry(maxRetry int) AskOption {
	return func(c *askConfig) {
		c.maxRetry = maxRetry
	}
}

func WithRetryPrompt(f func(ctx context.Context, err error) string) AskOption {
	return func(c *askConfig) {
		c.retryPrompt = f
	}
}

func Ask[T any](ctx context.Context, llm gollem.Session, prompt string, opts ...AskOption) (*T, error) {
	logger := logging.From(ctx)

	config := &askConfig{
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
		resp, err := llm.GenerateContent(ctx, gollem.Text(prompt))
		if err != nil {
			return nil, goerr.Wrap(err, "failed to send message")
		}

		if len(resp.Texts) == 0 {
			ctx = msg.Trace(ctx, "💥 failed to get valid response from LLM (no content parts), retry (%d/%d)", i+1, config.maxRetry)
			prompt = config.retryPrompt(ctx, err)
			continue
		}

		text := resp.Texts[0]
		if text == "" {
			ctx = msg.Trace(ctx, "💥 failed to get valid response from LLM (no text data), retry (%d/%d)", i+1, config.maxRetry)
			prompt = config.retryPrompt(ctx, err)
			continue
		}

		var result T
		if err := json.Unmarshal([]byte(text), &result); err != nil {
			logger.Debug("failed to unmarshal text", "text", text, "error", err)
			ctx = msg.Trace(ctx, "💥 failed to unmarshal text. retry (%d/%d)\n> %s", i+1, config.maxRetry, err.Error())
			prompt = config.retryPrompt(ctx, err)
			continue
		}

		response = &result
	}

	if response == nil {
		return nil, goerr.New("failed to get valid response from LLM", goerr.T(errs.TagInvalidLLMResponse))
	}

	return response, nil
}
