package interfaces

import (
	"context"
	"io"
	"time"

	"cloud.google.com/go/vertexai/genai"
	"github.com/google/go-github/v69/github"
	"github.com/m-mizutani/opaq"
	"github.com/secmon-lab/warren/pkg/model"
)

type LLMClient interface {
	StartChat() LLMSession
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

type GitHubAppClient interface {
	GetDefaultBranch(ctx context.Context, owner, repo string) (string, error)
	DownloadArchive(ctx context.Context, owner, repo, ref string) (io.ReadCloser, error)
	CommitChanges(ctx context.Context, owner, repo, branch string, files map[string][]byte, message string) error
	CreateBranch(ctx context.Context, owner, repo, baseBranch, newBranch string) error
	CreatePullRequest(ctx context.Context, owner, repo, title, body, head, base string) (*github.PullRequest, error)
}
