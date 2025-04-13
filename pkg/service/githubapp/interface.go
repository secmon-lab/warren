package githubapp

import (
	"context"

	"github.com/google/go-github/v69/github"
)

type Client interface {
	GetDefaultBranch(ctx context.Context, owner, repo string) (string, error)
	CommitChanges(ctx context.Context, owner, repo, branch string, files map[string][]byte, message string) error
	LookupBranch(ctx context.Context, owner, repo, branch string) (*github.Reference, error)
	CreateBranch(ctx context.Context, owner, repo, baseBranch, newBranch string) error
	CreatePullRequest(ctx context.Context, owner, repo, title, body, head, base string) (*github.PullRequest, error)
}
