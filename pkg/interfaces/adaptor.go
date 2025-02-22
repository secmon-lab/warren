package interfaces

import (
	"context"
	"time"

	"cloud.google.com/go/vertexai/genai"
	"github.com/m-mizutani/opaq"
	"github.com/secmon-lab/warren/pkg/model"
)

type GetGeminiStartChat func() GenAIChatSession

type GenAIChatSession interface {
	SendMessage(ctx context.Context, msg ...genai.Part) (*genai.GenerateContentResponse, error)
}

type PolicyClient interface {
	Query(context.Context, string, any, any, ...opaq.QueryOption) error
	Sources() map[string]string
}

type Repository interface {
	PutAlert(ctx context.Context, alert model.Alert) error
	GetAlert(ctx context.Context, alertID model.AlertID) (*model.Alert, error)
	GetAlertBySlackThread(ctx context.Context, thread model.SlackThread) (*model.Alert, error)
	GetAlertsByStatus(ctx context.Context, status model.AlertStatus) ([]model.Alert, error)
	BatchGetAlerts(ctx context.Context, alertIDs []model.AlertID) ([]model.Alert, error)
	PutAlertGroups(ctx context.Context, groups []model.AlertGroup) error
	GetAlertGroup(ctx context.Context, groupID model.AlertGroupID) (*model.AlertGroup, error)

	InsertAlertComment(ctx context.Context, comment model.AlertComment) error
	GetAlertComments(ctx context.Context, alertID model.AlertID) ([]model.AlertComment, error)
	FetchLatestAlerts(ctx context.Context, oldest time.Time, limit int) ([]model.Alert, error)
	GetPolicy(ctx context.Context, hash string) (*model.PolicyData, error)
	SavePolicy(ctx context.Context, policy *model.PolicyData) error
}
