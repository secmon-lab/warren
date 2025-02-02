package interfaces

import (
	"context"
	"net/http"

	"github.com/secmon-lab/warren/pkg/model"
)

type SlackService interface {
	PostAlert(ctx context.Context, alert model.Alert) (string, string, error)
	UpdateAlert(ctx context.Context, alert model.Alert) error
	VerifyRequest(header http.Header, body []byte) error
}
