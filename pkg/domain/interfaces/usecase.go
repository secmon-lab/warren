package interfaces

import (
	"context"

	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	model "github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/slack-go/slack"
)

type UseCase interface {
	// Slack event handlers
	HandleSlackMessage(ctx context.Context, slackThread model.Thread, text string, user model.User, ts string) error
	HandleSlackAppMention(ctx context.Context, user model.User, mention model.Mention, slackThread model.Thread) error

	// Slack interaction handlers
	HandleSlackInteractionViewSubmissionResolveAlert(ctx context.Context, user model.User, metadata string, values map[string]map[string]slack.BlockAction) error
	HandleSlackInteractionViewSubmissionResolveList(ctx context.Context, user model.User, metadata string, values map[string]map[string]slack.BlockAction) error
	HandleSlackInteractionViewSubmissionIgnoreList(ctx context.Context, metadata string, values map[string]map[string]slack.BlockAction) error
	HandleSlackInteractionBlockActions(ctx context.Context, user model.User, slackThread model.Thread, actionID slack.ActionID, value, triggerID string) error

	// Alert related handlers
	HandleAlert(ctx context.Context, schema string, alertData any, policyClient PolicyClient) ([]*alert.Alert, error)
	HandleAlertWithAuth(ctx context.Context, schema string, alertData any) ([]*alert.Alert, error)

	// Workflow related handlers
	RunWorkflow(ctx context.Context, alert alert.Alert) error
}
