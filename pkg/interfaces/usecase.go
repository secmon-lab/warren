package interfaces

import (
	"context"

	"github.com/secmon-lab/warren/pkg/model"
	"github.com/slack-go/slack"
)

type UseCase interface {
	HandleSlackMessage(ctx context.Context, slackThread model.SlackThread, text string, user model.SlackUser, ts string) error
	HandleSlackAppMention(ctx context.Context, user model.SlackUser, mention model.SlackMention, slackThread model.SlackThread) error
	HandleSlackInteractionViewSubmissionResolveAlert(ctx context.Context, user model.SlackUser, metadata string, values map[string]map[string]slack.BlockAction) error
	HandleSlackInteractionViewSubmissionResolveList(ctx context.Context, user model.SlackUser, metadata string, values map[string]map[string]slack.BlockAction) error
	HandleSlackInteractionViewSubmissionIgnoreList(ctx context.Context, slackThread model.SlackThread, metadata string, values map[string]map[string]slack.BlockAction) error
	HandleSlackInteractionBlockActions(ctx context.Context, user model.SlackUser, slackThread model.SlackThread, actionID model.SlackActionID, value, triggerID string) error

	// Alert related handlers
	HandleAlert(ctx context.Context, schema string, alertData any, policyClient PolicyClient) ([]*model.Alert, error)
	HandleAlertWithAuth(ctx context.Context, schema string, alertData any) ([]*model.Alert, error)

	// Workflow related handlers
	RunWorkflow(ctx context.Context, alert model.Alert) error
}
