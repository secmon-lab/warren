package config

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/m-mizutani/goerr/v2"
	policyadapter "github.com/secmon-lab/warren/pkg/adapter/policy"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/urfave/cli/v3"
)

type Policy struct {
	filePaths []string

	githubRepo           string
	githubPaths          []string
	githubAppID          int64
	githubInstallationID int64
	githubPrivateKey     string
	githubCacheTTL       time.Duration
}

func (x *Policy) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringSliceFlag{
			Name:        "policy",
			Usage:       "Policy file/dir path",
			Aliases:     []string{"p"},
			Destination: &x.filePaths,
			Category:    "Policy",
			Sources:     cli.EnvVars("WARREN_POLICY"),
		},
		&cli.StringFlag{
			Name:        "policy-github-repo",
			Usage:       "GitHub repository hosting Rego policy files (format: owner/repo). Loaded from default branch HEAD.",
			Destination: &x.githubRepo,
			Category:    "Policy",
			Sources:     cli.EnvVars("WARREN_POLICY_GITHUB_REPO"),
		},
		&cli.StringSliceFlag{
			Name:        "policy-github-path",
			Usage:       "Path within the GitHub repository to scan recursively for .rego files. May be specified multiple times. Defaults to repository root.",
			Destination: &x.githubPaths,
			Category:    "Policy",
			Sources:     cli.EnvVars("WARREN_POLICY_GITHUB_PATH"),
		},
		&cli.Int64Flag{
			Name:        "policy-github-app-id",
			Usage:       "GitHub App ID for policy repository access (separate App recommended; see doc/operation/policy.md).",
			Destination: &x.githubAppID,
			Category:    "Policy",
			Sources:     cli.EnvVars("WARREN_POLICY_GITHUB_APP_ID"),
		},
		&cli.Int64Flag{
			Name:        "policy-github-app-installation-id",
			Usage:       "GitHub App Installation ID for policy repository access.",
			Destination: &x.githubInstallationID,
			Category:    "Policy",
			Sources:     cli.EnvVars("WARREN_POLICY_GITHUB_APP_INSTALLATION_ID"),
		},
		&cli.StringFlag{
			Name:        "policy-github-app-private-key",
			Usage:       "GitHub App private key (PEM format) for policy repository access.",
			Destination: &x.githubPrivateKey,
			Category:    "Policy",
			Sources:     cli.EnvVars("WARREN_POLICY_GITHUB_APP_PRIVATE_KEY"),
		},
		&cli.DurationFlag{
			Name:        "policy-github-cache-ttl",
			Usage:       "TTL for cached GitHub policy contents. The HEAD commit sha is checked at most once per TTL; identical sha skips re-fetching.",
			Value:       policyadapter.DefaultGitHubCacheTTL,
			Destination: &x.githubCacheTTL,
			Category:    "Policy",
			Sources:     cli.EnvVars("WARREN_POLICY_GITHUB_CACHE_TTL"),
		},
	}
}

func (x Policy) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Any("file_paths", x.filePaths),
		slog.String("github_repo", x.githubRepo),
		slog.Any("github_paths", x.githubPaths),
		slog.Int64("github_app_id", x.githubAppID),
		slog.Int64("github_app_installation_id", x.githubInstallationID),
		slog.Bool("github_private_key_set", x.githubPrivateKey != ""),
		slog.Duration("github_cache_ttl", x.githubCacheTTL),
	)
}

// Configure builds a PolicyClient that aggregates configured policy sources.
// File and GitHub sources may be combined; if neither is configured the
// returned client behaves as a no-op (HasPolicies reports false beforehand).
func (x *Policy) Configure() (interfaces.PolicyClient, error) {
	var sources []policyadapter.Source

	if len(x.filePaths) > 0 {
		fs, err := policyadapter.NewFileSource(x.filePaths)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to create file policy source",
				goerr.V("file_paths", x.filePaths))
		}
		sources = append(sources, fs)
	}

	if x.githubRepo != "" {
		owner, repo, ok := splitGitHubRepo(x.githubRepo)
		if !ok {
			return nil, goerr.New("invalid policy-github-repo, expected owner/repo",
				goerr.V("value", x.githubRepo))
		}
		if x.githubAppID == 0 || x.githubInstallationID == 0 || x.githubPrivateKey == "" {
			return nil, goerr.New("policy-github-app-id, policy-github-app-installation-id, and policy-github-app-private-key are all required when policy-github-repo is set")
		}
		gs, err := policyadapter.NewGitHubSource(policyadapter.GitHubSourceOpts{
			Owner:          owner,
			Repo:           repo,
			Paths:          x.githubPaths,
			AppID:          x.githubAppID,
			InstallationID: x.githubInstallationID,
			PrivateKey:     []byte(x.githubPrivateKey),
			TTL:            x.githubCacheTTL,
		})
		if err != nil {
			return nil, goerr.Wrap(err, "failed to create GitHub policy source",
				goerr.V("repo", x.githubRepo))
		}
		sources = append(sources, gs)
	}

	loader := policyadapter.NewLoader(sources...)

	if loader.HasSources() {
		if err := loader.Prime(context.Background()); err != nil {
			return nil, goerr.Wrap(err, "failed to load initial policy")
		}
	}

	return loader, nil
}

// HasPolicies reports whether at least one policy source is configured.
func (x *Policy) HasPolicies() bool {
	return len(x.filePaths) > 0 || x.githubRepo != ""
}

func splitGitHubRepo(s string) (owner, repo string, ok bool) {
	idx := strings.Index(s, "/")
	if idx <= 0 || idx == len(s)-1 {
		return "", "", false
	}
	owner = s[:idx]
	repo = s[idx+1:]
	if strings.Contains(repo, "/") {
		return "", "", false
	}
	return owner, repo, true
}
