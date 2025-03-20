package interfaces

import (
	"context"

	"cloud.google.com/go/vertexai/genai"
	"github.com/google/go-github/v69/github"
	"github.com/m-mizutani/opaq"
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

type GitHubAppClient interface {
	GetDefaultBranch(ctx context.Context, owner, repo string) (string, error)
	CommitChanges(ctx context.Context, owner, repo, branch string, files map[string][]byte, message string) error
	LookupBranch(ctx context.Context, owner, repo, branch string) (*github.Reference, error)
	CreateBranch(ctx context.Context, owner, repo, baseBranch, newBranch string) error
	CreatePullRequest(ctx context.Context, owner, repo, title, body, head, base string) (*github.PullRequest, error)
}
