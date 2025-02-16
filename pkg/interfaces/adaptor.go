package interfaces

import (
	"context"
	"time"

	"cloud.google.com/go/vertexai/genai"
	"github.com/m-mizutani/opac"
	"github.com/secmon-lab/warren/pkg/model"
)

type GetGeminiStartChat func() GenAIChatSession

type GenAIChatSession interface {
	SendMessage(ctx context.Context, msg ...genai.Part) (*genai.GenerateContentResponse, error)
}

type PolicyClient interface {
	Query(context.Context, string, any, any, ...opac.QueryOption) error
}

type Repository interface {
	PutAlert(ctx context.Context, alert model.Alert) error
	GetAlert(ctx context.Context, alertID model.AlertID) (*model.Alert, error)
	GetAlertBySlackThread(ctx context.Context, thread model.SlackThread) (*model.Alert, error)
	InsertAlertComment(ctx context.Context, comment model.AlertComment) error
	GetAlertComments(ctx context.Context, alertID model.AlertID) ([]model.AlertComment, error)
	FetchLatestAlerts(ctx context.Context, oldest time.Time, limit int) ([]model.Alert, error)
}
