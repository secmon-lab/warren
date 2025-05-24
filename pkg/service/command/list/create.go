package list

import (
	"context"

	"github.com/m-mizutani/goerr/v2"

	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
)

// CreateList creates a new AlertList from the given alerts and registers it to the repository.
// It also generates meta data (title and description) for the list using LLM if possible.
func CreateList(ctx context.Context, repo interfaces.Repository, llmClient interfaces.LLMClient, thread slack.Thread, user *slack.User, alerts alert.Alerts) (*alert.List, error) {
	list := alert.NewList(ctx, thread, user, alerts)

	if err := list.FillMetadata(ctx, llmClient); err != nil {
		return nil, goerr.Wrap(err, "failed to fill metadata")
	}

	// Register the list to the repository
	if err := repo.PutAlertList(ctx, list); err != nil {
		return nil, goerr.Wrap(err, "failed to put alert list")
	}

	return list, nil
}
