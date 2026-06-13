package github

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/gollem-dev/gollem"
	extgithub "github.com/gollem-dev/tools/github"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

// Action is the warren-side wrapper around github.com/gollem-dev/tools/github.
// The Specs/Run logic lives in the external module; warren retains the CLI
// flags, the repository-hint YAML loading, and the planner Prompt() generated
// from those hints (the external module has no notion of repository hints).
type Action struct {
	appID          int64
	installationID int64
	privateKey     string
	configFiles    []string
	configs        []*RepositoryConfig

	inner gollem.ToolSet
}

var _ interfaces.Tool = &Action{}

func (x *Action) Helper() *cli.Command {
	return nil
}

func (x *Action) ID() string {
	return "github"
}

func (x *Action) Description() string {
	return "GitHub code and issue search, commit history, and file blame"
}

func (x *Action) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.Int64Flag{
			Name:        "github-app-id",
			Usage:       "GitHub App ID",
			Destination: &x.appID,
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_GITHUB_APP_ID"),
		},
		&cli.Int64Flag{
			Name:        "github-app-installation-id",
			Usage:       "GitHub App Installation ID",
			Destination: &x.installationID,
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_GITHUB_APP_INSTALLATION_ID"),
		},
		&cli.StringFlag{
			Name:        "github-app-private-key",
			Usage:       "GitHub App private key (PEM format)",
			Destination: &x.privateKey,
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_GITHUB_APP_PRIVATE_KEY"),
		},
		&cli.StringSliceFlag{
			Name:        "github-app-config",
			Usage:       "Path to GitHub repository configuration YAML file(s)",
			Destination: &x.configFiles,
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_GITHUB_APP_CONFIG"),
		},
	}
}

func (x *Action) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int64("app_id", x.appID),
		slog.Int64("installation_id", x.installationID),
		slog.Int("config_count", len(x.configs)),
	)
}

// Configure implements interfaces.Tool.
func (x *Action) Configure(ctx context.Context) error {
	logger := logging.From(ctx)

	if x.appID == 0 || x.installationID == 0 || x.privateKey == "" {
		return errutil.ErrActionUnavailable
	}

	// Load repository configurations. Configs are advisory hints surfaced to
	// the LLM via Prompt(); they do not gate API access.
	if len(x.configFiles) == 0 {
		logger.Warn("GitHub App configured without repository hint files; the LLM will see no suggested repositories")
	} else {
		var allConfigs []*RepositoryConfig
		for _, configPath := range x.configFiles {
			configs, err := x.loadConfig(configPath)
			if err != nil {
				return err
			}
			allConfigs = append(allConfigs, configs...)
		}

		if len(allConfigs) == 0 {
			logger.Warn("GitHub repository hint files loaded but contain no entries", slog.Any("files", x.configFiles))
		}
		x.configs = allConfigs
	}

	ts, err := extgithub.New(x.appID, x.installationID, x.privateKey)
	if err != nil {
		return goerr.Wrap(err, "failed to configure GitHub tool")
	}
	x.inner = ts

	return nil
}

func (x *Action) loadConfig(configPath string) ([]*RepositoryConfig, error) {
	fileInfo, err := os.Stat(configPath)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to stat config path", goerr.V("path", configPath))
	}

	var configs []*RepositoryConfig

	if fileInfo.IsDir() {
		entries, err := os.ReadDir(configPath)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to read directory", goerr.V("path", configPath))
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if ext != ".yaml" && ext != ".yml" {
				continue
			}

			filePath := filepath.Join(configPath, entry.Name())
			fileConfigs, err := x.loadConfigFile(filePath)
			if err != nil {
				return nil, err
			}
			configs = append(configs, fileConfigs...)
		}
	} else {
		fileConfigs, err := x.loadConfigFile(configPath)
		if err != nil {
			return nil, err
		}
		configs = append(configs, fileConfigs...)
	}

	return configs, nil
}

func (x *Action) loadConfigFile(filePath string) ([]*RepositoryConfig, error) {
	data, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to read config file", goerr.V("path", filePath))
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, goerr.Wrap(err, "failed to parse config file", goerr.V("path", filePath))
	}

	for _, repo := range config.Repositories {
		if repo.Owner == "" || repo.Repository == "" {
			return nil, goerr.New("invalid repository config: owner and repository are required",
				goerr.V("path", filePath),
				goerr.V("owner", repo.Owner),
				goerr.V("repository", repo.Repository))
		}
	}

	return config.Repositories, nil
}

// Specs implements gollem.ToolSet.
func (x *Action) Specs(ctx context.Context) ([]gollem.ToolSpec, error) {
	if x.inner == nil {
		return nil, goerr.New("GitHub tool is not configured")
	}
	return x.inner.Specs(ctx)
}

// Run implements gollem.ToolSet.
func (x *Action) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	if x.inner == nil {
		return nil, goerr.New("GitHub tool is not configured")
	}
	return x.inner.Run(ctx, name, args)
}

// Prompt implements interfaces.Tool. It surfaces the configured repository
// hints to the planner.
func (x *Action) Prompt(_ context.Context) (string, error) {
	if len(x.configs) == 0 {
		return "", nil
	}

	var sb strings.Builder
	sb.WriteString("## GitHub Repositories\n\n")
	sb.WriteString("The following repositories are recommended starting points for investigation. They are hints, not an access allowlist: any repository reachable by the GitHub App installation can also be queried by passing its owner/name explicitly.\n\n")

	for _, config := range x.configs {
		fmt.Fprintf(&sb, "- **%s/%s**", config.Owner, config.Repository)
		if config.Description != "" {
			fmt.Fprintf(&sb, ": %s", config.Description)
		}
		if config.DefaultBranch != "" {
			fmt.Fprintf(&sb, " (default branch: %s)", config.DefaultBranch)
		}
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString("Use the GitHub tools to search code, issues/PRs, retrieve file contents, list commit history, or get file blame. For search tools, use the `repo_filter` parameter (comma-separated `owner/name` list) or include `repo:`/`org:`/`user:` qualifiers in the query to scope results.\n")

	return sb.String(), nil
}
