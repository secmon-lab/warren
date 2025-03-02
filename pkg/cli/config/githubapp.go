package config

import (
	"context"
	"log/slog"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/adapter/githubapp"
	"github.com/secmon-lab/warren/pkg/service"
	"github.com/urfave/cli/v3"
)

type GitHubAppCfg struct {
	appID          int64
	installationID int64
	privateKey     string

	owner string
	repo  string

	policyRootDir string
	detectTestDir string
	ignoreTestDir string
}

func (c *GitHubAppCfg) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.IntFlag{
			Name:        "github-app-id",
			Usage:       "GitHub App ID",
			Category:    "GitHub App",
			Sources:     cli.EnvVars("WARREN_GITHUB_APP_ID"),
			Destination: &c.appID,
		},
		&cli.IntFlag{
			Name:        "github-app-installation-id",
			Usage:       "GitHub App Installation ID",
			Category:    "GitHub App",
			Sources:     cli.EnvVars("WARREN_GITHUB_APP_INSTALLATION_ID"),
			Destination: &c.installationID,
		},
		&cli.StringFlag{
			Name:        "github-app-private-key",
			Usage:       "GitHub App Private Key",
			Category:    "GitHub App",
			Sources:     cli.EnvVars("WARREN_GITHUB_APP_PRIVATE_KEY"),
			Destination: &c.privateKey,
		},
		&cli.StringFlag{
			Name:        "github-app-owner",
			Usage:       "GitHub App Owner",
			Category:    "GitHub App",
			Sources:     cli.EnvVars("WARREN_GITHUB_APP_OWNER"),
			Destination: &c.owner,
		},
		&cli.StringFlag{
			Name:        "github-app-repo",
			Usage:       "GitHub App Repository",
			Category:    "GitHub App",
			Sources:     cli.EnvVars("WARREN_GITHUB_APP_REPO"),
			Destination: &c.repo,
		},
		&cli.StringFlag{
			Name:        "github-app-policy-root-dir",
			Usage:       "GitHub App Policy Root Directory",
			Category:    "GitHub App",
			Sources:     cli.EnvVars("WARREN_GITHUB_APP_POLICY_ROOT_DIR"),
			Destination: &c.policyRootDir,
		},
		&cli.StringFlag{
			Name:        "github-app-detect-test-dir",
			Usage:       "GitHub App Detect Test Directory",
			Category:    "GitHub App",
			Sources:     cli.EnvVars("WARREN_GITHUB_APP_DETECT_TEST_DIR"),
			Destination: &c.detectTestDir,
		},
		&cli.StringFlag{
			Name:        "github-app-ignore-test-dir",
			Usage:       "GitHub App Ignore Test Directory",
			Category:    "GitHub App",
			Sources:     cli.EnvVars("WARREN_GITHUB_APP_IGNORE_TEST_DIR"),
			Destination: &c.ignoreTestDir,
		},
	}
}

func (c GitHubAppCfg) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int("app-id", int(c.appID)),
		slog.Int("installation-id", int(c.installationID)),
		slog.Int("private-key.len", len(c.privateKey)),
		slog.String("owner", c.owner),
		slog.String("repo", c.repo),
		slog.String("policy-root-dir", c.policyRootDir),
		slog.String("detect-test-dir", c.detectTestDir),
		slog.String("ignore-test-dir", c.ignoreTestDir),
	)
}

func (c GitHubAppCfg) Configure(ctx context.Context) (*service.GitHubApp, error) {
	if c.appID == 0 {
		return nil, nil
	}

	if c.installationID == 0 || c.privateKey == "" || c.owner == "" || c.repo == "" {
		return nil, goerr.New("github app config is not set")
	}

	appClient, err := githubapp.New(ctx, c.appID, c.installationID, []byte(c.privateKey))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create github app client")
	}

	svc := service.NewGitHubApp(appClient, service.GitHubAppConfig{
		Owner: c.owner,
		Repo:  c.repo,

		PolicyRootDir: c.policyRootDir,
		DetectTestDir: c.detectTestDir,
		IgnoreTestDir: c.ignoreTestDir,
	})

	return svc, nil
}
