package githubapp_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-github/v69/github"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/policy"
	"github.com/secmon-lab/warren/pkg/domain/types"

	"github.com/secmon-lab/warren/pkg/service/githubapp"
	"github.com/secmon-lab/warren/pkg/utils/clock"
)

func TestGitHubApp_CreatePullRequest(t *testing.T) {
	ctx := t.Context()

	mockClient := &githubapp.ClientMock{
		GetDefaultBranchFunc: func(ctx context.Context, owner, repo string) (string, error) {
			return "main", nil
		},
		LookupBranchFunc: func(ctx context.Context, owner, repo, branch string) (*github.Reference, error) {
			return nil, nil
		},
		CreateBranchFunc: func(ctx context.Context, owner, repo, baseBranch, newBranch string) error {
			return nil
		},
		CommitChangesFunc: func(ctx context.Context, owner, repo, branch string, files map[string][]byte, message string) error {
			return nil
		},
		CreatePullRequestFunc: func(ctx context.Context, owner, repo, title, body, head, base string) (*github.PullRequest, error) {
			return &github.PullRequest{
				HTMLURL: github.Ptr("https://github.com/owner/repo/pull/1"),
			}, nil
		},
	}

	app := githubapp.New(mockClient, githubapp.Config{
		Owner:         "owner",
		Repo:          "repo",
		PolicyRootDir: "policies",
		DetectTestDir: "test/detect",
		IgnoreTestDir: "test/ignore",
	})

	diff := &policy.Diff{
		ID:    policy.NewPolicyDiffID(),
		Title: "Test PR",
		New: map[string]string{
			"test.rego": "package color\n\nalert contains {} if {\n  input.color == \"red\"\n}",
		},
		NewTestDataSet: &policy.TestDataSet{
			Detect: &policy.TestData{
				Data: map[types.AlertSchema]map[string]any{
					types.AlertSchema("schema"): {
						"test.json": map[string]any{"test": "data"},
					},
				},
			},
			Ignore: &policy.TestData{
				Data: map[types.AlertSchema]map[string]any{
					types.AlertSchema("schema"): {
						"ignore.json": map[string]any{"ignore": "data"},
					},
				},
			},
		},
	}

	ctx = clock.With(ctx, func() time.Time {
		return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	})

	url, err := app.CreatePolicyDiffPullRequest(ctx, diff)
	gt.NoError(t, err)
	gt.Equal(t, url.String(), "https://github.com/owner/repo/pull/1")

	gt.A(t, mockClient.GetDefaultBranchCalls()).Length(1)
	gt.A(t, mockClient.LookupBranchCalls()).Length(1)
	gt.A(t, mockClient.CreateBranchCalls()).Length(1).
		At(0, func(t testing.TB, v struct {
			Ctx        context.Context
			Owner      string
			Repo       string
			BaseBranch string
			NewBranch  string
		}) {
			gt.Equal(t, v.Owner, "owner")
			gt.Equal(t, v.Repo, "repo")
			gt.Equal(t, v.BaseBranch, "main")
			gt.Equal(t, v.NewBranch, "warren/2025-01-01/"+diff.ID.String())
		})
	gt.A(t, mockClient.CommitChangesCalls()).Length(1).
		At(0, func(t testing.TB, v struct {
			Ctx     context.Context
			Owner   string
			Repo    string
			Branch  string
			Files   map[string][]byte
			Message string
		}) {
			gt.Equal(t, v.Owner, "owner")
			gt.Equal(t, v.Repo, "repo")
			gt.Equal(t, v.Branch, "warren/2025-01-01/"+diff.ID.String())
			gt.Equal(t, v.Message, diff.Title)
			gt.M(t, v.Files).
				HaveKey("policies/test.rego").
				HaveKey("test/detect/schema/test.json").
				HaveKey("test/ignore/schema/ignore.json")
		})
	gt.A(t, mockClient.CreatePullRequestCalls()).Length(1).
		At(0, func(t testing.TB, v struct {
			Ctx   context.Context
			Owner string
			Repo  string
			Title string
			Body  string
			Head  string
			Base  string
		}) {
			gt.Equal(t, v.Owner, "owner")
			gt.Equal(t, v.Repo, "repo")
			gt.Equal(t, v.Title, diff.Title)
		})
}
