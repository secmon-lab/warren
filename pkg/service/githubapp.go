package service

import (
	"context"
	"encoding/json"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/interfaces"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

type GitHubApp struct {
	appClient interfaces.GitHubAppClient
	config    GitHubAppConfig
}

type GitHubAppConfig struct {
	Owner string
	Repo  string

	PolicyRootDir string
	DetectTestDir string
	IgnoreTestDir string
}

func NewGitHubApp(appClient interfaces.GitHubAppClient, config GitHubAppConfig) *GitHubApp {
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

	if ref, err := x.appClient.LookupBranch(ctx, x.config.Owner, x.config.Repo, newBranch); err != nil {
		return nil, eb.Wrap(err, "failed to lookup branch")
	} else if ref == nil {
		if err := x.appClient.CreateBranch(ctx, x.config.Owner, x.config.Repo, defaultBranch, newBranch); err != nil {
			return nil, eb.Wrap(err, "failed to create branch")
		}
	}

	files := map[string][]byte{}

	// Set policy files
	for path, content := range diff.New {
		fpath := filepath.Join(x.config.PolicyRootDir, path)
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		files[fpath] = []byte(content)
	}

	setTestData := func(dir string, testData map[string]map[string]any) error {
		for schema, testData := range testData {
			for fname, content := range testData {
				fpath := filepath.Join(dir, schema, fname)
				raw, err := json.MarshalIndent(content, "", "  ")
				if err != nil {
					return eb.Wrap(err, "failed to marshal test data", goerr.V("fname", fname), goerr.V("content", content))
				}
				if !strings.HasSuffix(string(raw), "\n") {
					raw = append(raw, '\n')
				}
				files[fpath] = raw
			}
		}
		return nil
	}

	if err := setTestData(x.config.DetectTestDir, diff.NewTestDataSet.Detect.Data); err != nil {
		return nil, eb.Wrap(err, "failed to set detect test data")
	}
	if err := setTestData(x.config.IgnoreTestDir, diff.NewTestDataSet.Ignore.Data); err != nil {
		return nil, eb.Wrap(err, "failed to set ignore test data")
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
