package http

import (
	"context"

	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
)

type useCase interface {
	// Slack event handlers
	HandleSlackMessage(ctx context.Context, slackThread slack.Thread, text string, user slack.User, ts string) error
	HandleSlackAppMention(ctx context.Context, user slack.User, mention slack.Mention, slackThread slack.Thread) error

	// Slack interaction handlers
	HandleSlackInteractionViewSubmissionResolveAlert(ctx context.Context, user slack.User, metadata string, values slack.StateValue) error
	HandleSlackInteractionViewSubmissionResolveList(ctx context.Context, user slack.User, metadata string, values slack.StateValue) error
	HandleSlackInteractionViewSubmissionIgnoreList(ctx context.Context, metadata string, values slack.StateValue) error
	HandleSlackInteractionBlockActions(ctx context.Context, user slack.User, slackThread slack.Thread, actionID slack.ActionID, value, triggerID string) error

	// Alert related handlers
	HandleAlertWithAuth(ctx context.Context, schema string, alertData any) ([]*alert.Alert, error)
}
