package interfaces

import (
	"context"
	"net/http"

	"github.com/secmon-lab/warren/pkg/domain/model"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/prompt"
)

type SlackService interface {
	NewThread(thread slack.SlackThread) SlackThreadService
	PostAlert(ctx context.Context, alert alert.Alert) (SlackThreadService, error)
	IsBotUser(userID string) bool
}

type SlackThreadService interface {
	ChannelID() string
	ThreadID() string

	UpdateAlert(ctx context.Context, alert alert.Alert) error
	PostNextAction(ctx context.Context, action prompt.ActionPromptResult) error
	PostFinding(ctx context.Context, finding alert.AlertFinding) error
	AttachFile(ctx context.Context, title, fileName string, data []byte) error
	PostPolicyDiff(ctx context.Context, diff *model.PolicyDiff) error
	PostAlerts(ctx context.Context, alerts []alert.Alert) error
	PostAlertList(ctx context.Context, list *alert.List) error
	PostAlertClusters(ctx context.Context, clusters []alert.List) error

	// Reply replies to the thread with a message. It does not return an error because the process should not be stopped even if it fails. Instead, the error should be logged and reported to sentry in the method.
	Reply(ctx context.Context, message string)
}

type SlackPayloadVerifier func(ctx context.Context, header http.Header, payload []byte) error
