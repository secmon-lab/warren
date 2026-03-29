package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

type askConfig[T any] struct {
	maxRetry int
	validate func(v T) error
}

type AskOption[T any] func(*askConfig[T])

func WithMaxRetry[T any](maxRetry int) AskOption[T] {
	return func(c *askConfig[T]) {
		c.maxRetry = maxRetry
	}
}

func WithValidate[T any](f func(v T) error) AskOption[T] {
	return func(c *askConfig[T]) {
		c.validate = f
	}
}

func Ask[T any](ctx context.Context, llm gollem.LLMClient, prompt string, opts ...AskOption[T]) (*T, error) {
	config := &askConfig[T]{
		maxRetry: 3,
	}
	for _, opt := range opts {
		opt(config)
	}

	// Without validation, delegate directly to gollem.Query
	if config.validate == nil {
		resp, err := gollem.Query[T](ctx, llm, prompt,
			gollem.WithQueryMaxRetry(config.maxRetry),
		)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to get valid response from LLM", goerr.T(errutil.TagInvalidLLMResponse))
		}
		return resp.Data, nil
	}

	// With validation, use a single session to preserve conversation context
	// across retries. This ensures the LLM sees the original prompt and all
	// correction feedback in the same conversation history.
	return askWithValidation(ctx, llm, prompt, config)
}

// askWithValidation runs a session-based retry loop that preserves conversation
// context. On each iteration the LLM can see the original prompt plus every
// prior correction, so it has full context for fixing validation errors.
func askWithValidation[T any](ctx context.Context, llm gollem.LLMClient, prompt string, config *askConfig[T]) (*T, error) {
	logger := logging.From(ctx)

	var zero T
	sessionOpts := []gollem.SessionOption{
		gollem.WithSessionContentType(gollem.ContentTypeJSON),
	}
	if schema, err := gollem.ToSchema(zero); err == nil {
		sessionOpts = append(sessionOpts, gollem.WithSessionResponseSchema(schema))
	} else {
		errutil.Handle(ctx, goerr.Wrap(err, "failed to generate schema from type",
			goerr.V("type", fmt.Sprintf("%T", zero))))
	}

	ssn, err := llm.NewSession(ctx, sessionOpts...)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create session")
	}

	currentPrompt := prompt
	for i := 0; i < config.maxRetry; i++ {
		resp, err := ssn.Generate(ctx, []gollem.Input{gollem.Text(currentPrompt)})
		if err != nil {
			return nil, goerr.Wrap(err, "failed to send message")
		}

		if len(resp.Texts) == 0 || resp.Texts[0] == "" {
			msg.Trace(ctx, "💥 empty response from LLM, retry (%d/%d)", i+1, config.maxRetry)
			currentPrompt = "No response received. Please try again with the original request."
			continue
		}

		var result T
		if err := json.Unmarshal([]byte(resp.Texts[0]), &result); err != nil {
			logger.Debug("failed to unmarshal text", "text", resp.Texts[0], "error", err)
			msg.Trace(ctx, "💥 failed to unmarshal response, retry (%d/%d)", i+1, config.maxRetry)
			currentPrompt = fmt.Sprintf("Invalid JSON response. Please fix and try again: %s", err.Error())
			continue
		}

		if err := config.validate(result); err != nil {
			msg.Trace(ctx, "💥 invalid response from LLM, retry (%d/%d)", i+1, config.maxRetry)
			currentPrompt = fmt.Sprintf("Response did not pass validation. Please fix and try again: %s", err.Error())
			continue
		}

		return &result, nil
	}

	return nil, goerr.New("failed to get valid response from LLM", goerr.T(errutil.TagInvalidLLMResponse))
}
