package interfaces

import (
	"context"
	"time"

	"cloud.google.com/go/vertexai/genai"
	"github.com/google/go-github/v69/github"
	"github.com/m-mizutani/opaq"
	"github.com/secmon-lab/warren/pkg/model"
)

type LLMClient interface {
	StartChat() LLMSession
	SendMessage(ctx context.Context, msg ...genai.Part) (*genai.GenerateContentResponse, error)
}

type EmbeddingClient interface {
	Embeddings(ctx context.Context, texts []string, dimensionality int) ([][]float32, error)
}

type LLMSession interface {
	SendMessage(ctx context.Context, msg ...genai.Part) (*genai.GenerateContentResponse, error)
}

type PolicyClient interface {
	Query(context.Context, string, any, any, ...opaq.QueryOption) error
	Sources() map[string]string
}

type Repository interface {
	PutAlert(ctx context.Context, alert model.Alert) error
	GetAlert(ctx context.Context, alertID model.AlertID) (*model.Alert, error)
	GetAlertsBySlackThread(ctx context.Context, thread model.SlackThread) ([]model.Alert, error)
	GetAlertsByStatus(ctx context.Context, status model.AlertStatus) ([]model.Alert, error)
	GetAlertsByParentID(ctx context.Context, parentID model.AlertID) ([]model.Alert, error)
	BatchGetAlerts(ctx context.Context, alertIDs []model.AlertID) ([]model.Alert, error)
	GetAlertsBySpan(ctx context.Context, begin, end time.Time) ([]model.Alert, error)
	BatchUpdateAlertStatus(ctx context.Context, alertIDs []model.AlertID, status model.AlertStatus) error
	BatchUpdateAlertConclusion(ctx context.Context, alertIDs []model.AlertID, conclusion model.AlertConclusion, reason string) error
	GetAlertListByThread(ctx context.Context, thread model.SlackThread) (*model.AlertList, error)
	GetAlertList(ctx context.Context, listID model.AlertListID) (*model.AlertList, error)
	PutAlertList(ctx context.Context, list model.AlertList) error
	GetLatestAlertListInThread(ctx context.Context, thread model.SlackThread) (*model.AlertList, error)
	GetAlertsWithoutStatus(ctx context.Context, status model.AlertStatus) ([]model.Alert, error)
	InsertAlertComment(ctx context.Context, comment model.AlertComment) error
	GetAlertComments(ctx context.Context, alertID model.AlertID) ([]model.AlertComment, error)
	GetLatestAlerts(ctx context.Context, oldest time.Time, limit int) ([]model.Alert, error)
	GetPolicy(ctx context.Context, hash string) (*model.PolicyData, error)
	SavePolicy(ctx context.Context, policy *model.PolicyData) error
	GetPolicyDiff(ctx context.Context, id model.PolicyDiffID) (*model.PolicyDiff, error)
	PutPolicyDiff(ctx context.Context, diff *model.PolicyDiff) error
}

type GitHubAppClient interface {
	GetDefaultBranch(ctx context.Context, owner, repo string) (string, error)
	CommitChanges(ctx context.Context, owner, repo, branch string, files map[string][]byte, message string) error
	LookupBranch(ctx context.Context, owner, repo, branch string) (*github.Reference, error)
	CreateBranch(ctx context.Context, owner, repo, baseBranch, newBranch string) error
	CreatePullRequest(ctx context.Context, owner, repo, title, body, head, base string) (*github.PullRequest, error)
}
