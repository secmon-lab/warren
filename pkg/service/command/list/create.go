package list

import (
	"context"

	"github.com/m-mizutani/goerr/v2"

	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/prompt"
	"github.com/secmon-lab/warren/pkg/service/llm"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

// CreateList creates a new AlertList from the given alerts and registers it to the repository.
// It also generates meta data (title and description) for the list using LLM if possible.
func CreateList(ctx context.Context, repo interfaces.Repository, llmQuery interfaces.LLMQuery, thread slack.Thread, user *slack.User, alerts alert.Alerts) (*alert.List, error) {
	list := alert.NewList(ctx, thread, user, alerts)

	// Generate meta data for the list
	p, err := prompt.BuildMetaListPrompt(ctx, list)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to build meta list prompt")
	}

	msg.Trace(ctx, "📝 Generating meta data for list: %s", list.ID)
	resp, err := llm.Ask[prompt.MetaListPromptResult](ctx, llmQuery, p)
	if err != nil {
		msg.Trace(ctx, "💥 failed meta data generation, skip")
	} else {
		list.Title = resp.Title
		list.Description = resp.Description
	}

	// Register the list to the repository
	if err := repo.PutAlertList(ctx, list); err != nil {
		return nil, goerr.Wrap(err, "failed to put alert list")
	}

	return &list, nil
}
