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
	VerifyRequest(header http.Header, body []byte) error
}

type SlackThreadService interface {
	ChannelID() string
	ThreadID() string

	UpdateAlert(ctx context.Context, alert model.Alert) error
	PostNextAction(ctx context.Context, action prompt.ActionPromptResult) error
	PostFinding(ctx context.Context, finding model.AlertFinding) error
	AttachFile(ctx context.Context, title, fileName string, data []byte) error
	Reply(ctx context.Context, message string) error
}
