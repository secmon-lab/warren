package interfaces

import (
	"context"
	"net/http"

	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/prompt"
)

type SlackService interface {
	NewThread(alert model.Alert) SlackThreadService
	PostAlert(ctx context.Context, alert model.Alert) (SlackThreadService, error)
	TrimMention(message string) string
	ShowCloseAlertModal(ctx context.Context, alert model.Alert, triggerID string) error
}

type SlackThreadService interface {
	ChannelID() string
	ThreadID() string

	UpdateAlert(ctx context.Context, alert model.Alert) error
	PostNextAction(ctx context.Context, action prompt.ActionPromptResult) error
	PostFinding(ctx context.Context, finding model.AlertFinding) error
	AttachFile(ctx context.Context, title, fileName string, data []byte) error

	// Reply replies to the thread with a message. It does not return an error because the process should not be stopped even if it fails. Instead, the error should be logged and reported to sentry in the method.
	Reply(ctx context.Context, message string)
}

type SlackPayloadVerifier func(ctx context.Context, header http.Header, payload []byte) error
