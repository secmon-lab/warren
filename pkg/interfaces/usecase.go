package interfaces

import (
	"context"

	"github.com/secmon-lab/warren/pkg/model"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

type UseCase interface {
	// Slack related handlers
	HandleSlackAppMention(ctx context.Context, event *slackevents.AppMentionEvent) error
	HandleSlackMessage(ctx context.Context, event *slackevents.MessageEvent) error
	HandleSlackInteraction(ctx context.Context, interaction slack.InteractionCallback) error

	// Alert related handlers
	HandleAlert(ctx context.Context, schema string, alertData any, policyClient PolicyClient) ([]*model.Alert, error)
	HandleAlertWithAuth(ctx context.Context, schema string, alertData any) ([]*model.Alert, error)

	// Workflow related handlers
	RunWorkflow(ctx context.Context, alert model.Alert) error
}
