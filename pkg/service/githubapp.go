package service

import (
	"context"
	"encoding/json"
	"net/url"
	"path/filepath"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/adapter/githubapp"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

type GitHubApp struct {
	appClient *githubapp.Client
	config    GitHubAppConfig
}

type GitHubAppConfig struct {
	Owner string
	Repo  string

	PolicyBaseDir string
	DetectTestDir string
	IgnoreTestDir string
}

func NewGitHubApp(appClient *githubapp.Client, config GitHubAppConfig) *GitHubApp {
	return &GitHubApp{
		appClient: appClient,
		config:    config,
	}
}

func (x *GitHubApp) CreatePullRequest(ctx context.Context, diff *model.PolicyDiff) (*url.URL, error) {
	logger := logging.From(ctx)

	eb := goerr.NewBuilder(goerr.V("config", x.config))
	defaultBranch, err := x.appClient.GetDefaultBranch(ctx, x.config.Owner, x.config.Repo)
	if err != nil {
		return nil, eb.Wrap(err, "failed to get default branch")
	}

	now := clock.Now(ctx)
	newBranch := "warren/" + now.Format("2006-01-02") + "/" + diff.ID.String()
	eb = eb.With(goerr.V("new_branch", newBranch))
	err = x.appClient.CreateBranch(ctx, x.config.Owner, x.config.Repo, defaultBranch, newBranch)
	if err != nil {
		return nil, eb.Wrap(err, "failed to create branch")
	}

	// Add test line to README.md

	files := map[string][]byte{}

	// Set policy files
	for path, content := range diff.New {
		fpath := filepath.Join(x.config.PolicyBaseDir, path)
		files[fpath] = []byte(content)
	}

	for schema, testData := range diff.TestDataSet.Detect.Data {
		for fname, content := range testData {
			fpath := filepath.Join(x.config.DetectTestDir, schema, fname)
			raw, err := json.MarshalIndent(content, "", "  ")
			if err != nil {
				return nil, eb.Wrap(err, "failed to marshal test data", goerr.V("schema", schema), goerr.V("fname", fname), goerr.V("content", content))
			}
			files[fpath] = raw
		}
	}

	for schema, testData := range diff.TestDataSet.Ignore.Data {
		for fname, content := range testData {
			fpath := filepath.Join(x.config.IgnoreTestDir, schema, fname)
			raw, err := json.MarshalIndent(content, "", "  ")
			if err != nil {
				return nil, eb.Wrap(err, "failed to marshal test data", goerr.V("schema", schema), goerr.V("fname", fname), goerr.V("content", content))
			}
			files[fpath] = raw
		}
	}

	logger.Debug("Set files", "files", files)
	eb = eb.With(goerr.V("files", files))

	if err := x.appClient.CommitChanges(ctx, x.config.Owner, x.config.Repo, newBranch, files, diff.Title); err != nil {
		return nil, eb.Wrap(err, "failed to commit changes")
	}

	// Create pull request
	pr, err := x.appClient.CreatePullRequest(ctx,
		x.config.Owner,
		x.config.Repo,
		diff.Title,
		diff.Description,
		newBranch,
		defaultBranch,
	)
	if err != nil {
		return nil, eb.Wrap(err, "failed to create pull request")
	}

	logger.Debug("Created pull request", "pr", pr)

	prURL, err := url.Parse(pr.GetHTMLURL())
	if err != nil {
		return nil, eb.Wrap(err, "failed to parse pull request URL", goerr.V("pr", pr))
	}

	return prURL, nil
}
