package githubapp

import (
	"context"
	"encoding/json"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/policy"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

type Service struct {
	client Client
	config Config
}

type Config struct {
	Owner string
	Repo  string

	PolicyRootDir string
	DetectTestDir string
	IgnoreTestDir string
}

func New(client Client, config Config) *Service {
	return &Service{
		client: client,
		config: config,
	}
}

func (x *Service) CreatePolicyDiffPullRequest(ctx context.Context, diff *policy.Diff) (*url.URL, error) {
	logger := logging.From(ctx)

	eb := goerr.NewBuilder(goerr.V("config", x.config))
	defaultBranch, err := x.client.GetDefaultBranch(ctx, x.config.Owner, x.config.Repo)
	if err != nil {
		return nil, eb.Wrap(err, "failed to get default branch")
	}

	now := clock.Now(ctx)
	newBranch := "warren/" + now.Format("2006-01-02") + "/" + diff.ID.String()
	eb = eb.With(goerr.V("new_branch", newBranch))

	if ref, err := x.client.LookupBranch(ctx, x.config.Owner, x.config.Repo, newBranch); err != nil {
		return nil, eb.Wrap(err, "failed to lookup branch")
	} else if ref == nil {
		if err := x.client.CreateBranch(ctx, x.config.Owner, x.config.Repo, defaultBranch, newBranch); err != nil {
			return nil, eb.Wrap(err, "failed to create branch")
		}
	}

	files := map[string]string{}

	// Set policy files
	for path, content := range diff.New {
		fpath := filepath.Join(x.config.PolicyRootDir, path)
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		files[fpath] = content
	}

	setTestData := func(dir string, testData *policy.TestData) error {
		for schema, data := range testData.Data {
			for fname, content := range data {
				fpath := filepath.Join(dir, schema.String(), fname)
				raw, err := json.MarshalIndent(content, "", "  ")
				if err != nil {
					return eb.Wrap(err, "failed to marshal test data", goerr.V("fname", fname), goerr.V("content", content))
				}
				data := string(raw)
				if !strings.HasSuffix(data, "\n") {
					data += "\n"
				}
				files[fpath] = data
			}
		}

		for schema, metaFiles := range testData.Metafiles {
			for fname, content := range metaFiles {
				fpath := filepath.Join(dir, schema.String(), fname)
				files[fpath] = content
			}
		}
		return nil
	}

	if err := setTestData(x.config.DetectTestDir, diff.NewTestDataSet.Detect); err != nil {
		return nil, eb.Wrap(err, "failed to set detect test data")
	}
	if err := setTestData(x.config.IgnoreTestDir, diff.NewTestDataSet.Ignore); err != nil {
		return nil, eb.Wrap(err, "failed to set ignore test data")
	}

	logger.Debug("Set files", "files", files)
	eb = eb.With(goerr.V("files", files))

	binData := map[string][]byte{}
	for fpath, content := range files {
		binData[fpath] = []byte(content)
	}

	if err := x.client.CommitChanges(ctx, x.config.Owner, x.config.Repo, newBranch, binData, diff.Title); err != nil {
		return nil, eb.Wrap(err, "failed to commit changes")
	}

	// Create pull request
	pr, err := x.client.CreatePullRequest(ctx,
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
