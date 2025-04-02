package usecase

import (
	"context"

	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

type SlackEvent interface {
	// Slack event handlers
	HandleSlackMessage(ctx context.Context, slackMsg *slack.Message, text string, user slack.User, ts string) error
	HandleSlackAppMention(ctx context.Context, user slack.User, mention slack.Mention, slackMsg *slack.Message) error
}

type SlackInteraction interface {
	// Slack interaction handlers
	HandleSlackInteractionViewSubmissionResolveAlert(ctx context.Context, user slack.User, metadata string, values slack.StateValue) error
	HandleSlackInteractionViewSubmissionResolveList(ctx context.Context, user slack.User, metadata string, values slack.StateValue) error
	HandleSlackInteractionViewSubmissionIgnoreList(ctx context.Context, metadata string, values slack.StateValue) error
	HandleSlackInteractionBlockActions(ctx context.Context, user slack.User, slackThread slack.Thread, actionID slack.ActionID, value, triggerID string) error
}

type Alert interface {
	// Alert related handlers
	HandleAlertWithAuth(ctx context.Context, schema types.AlertSchema, alertData any) ([]*alert.Alert, error)
}
