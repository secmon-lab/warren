package github

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v74/github"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

type Action struct {
	appID          int64
	installationID int64
	privateKey     string
	configFiles    []string
	configs        []*RepositoryConfig
	githubClient   *github.Client
	httpClient     *http.Client
}

var _ interfaces.Tool = &Action{}

func (x *Action) Helper() *cli.Command {
	return nil
}

func (x *Action) Name() string {
	return "github"
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

// Configure implements interfaces.Tool
func (x *Action) Configure(ctx context.Context) error {
	// Validate required settings
	if x.appID == 0 || x.installationID == 0 || x.privateKey == "" {
		return errutil.ErrActionUnavailable
	}

	if len(x.configFiles) == 0 {
		logging.Default().Warn("GitHub App credentials provided but no config files specified")
		return errutil.ErrActionUnavailable
	}

	// Private key is already loaded from flag/env var

	// Load repository configurations
	var allConfigs []*RepositoryConfig
	for _, configPath := range x.configFiles {
		configs, err := x.loadConfig(configPath)
		if err != nil {
			return err
		}
		allConfigs = append(allConfigs, configs...)
	}

	if len(allConfigs) == 0 {
		return goerr.New("no repository configurations found")
	}
	x.configs = allConfigs

	// Create GitHub client with App authentication
	transport, err := ghinstallation.New(http.DefaultTransport, x.appID, x.installationID, []byte(x.privateKey))
	if err != nil {
		return goerr.Wrap(err, "failed to create GitHub App transport")
	}

	// Create HTTP client with transport
	x.httpClient = &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	// Create GitHub client
	x.githubClient = github.NewClient(x.httpClient)

	return nil
}

func (x *Action) loadConfig(configPath string) ([]*RepositoryConfig, error) {
	fileInfo, err := os.Stat(configPath)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to stat config path", goerr.V("path", configPath))
	}

	var configs []*RepositoryConfig

	if fileInfo.IsDir() {
		// Load all YAML files from directory
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
		// Load single file
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

	// Validate configurations
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

// Specs implements gollem.ToolSet
func (x *Action) Specs(ctx context.Context) ([]gollem.ToolSpec, error) {
	return []gollem.ToolSpec{
		{
			Name:        "github_code_search",
			Description: "Search for code in configured GitHub repositories. Query syntax examples: 'function login', 'language:go fmt.Println', 'path:src/ extension:js', 'filename:config NOT test'",
			Parameters: map[string]*gollem.Parameter{
				"query": {
					Type:        gollem.TypeString,
					Description: "Search query using GitHub code search syntax. Supports operators like AND, OR, NOT",
					Required:    true,
					MinLength:   github.Ptr(1),
				},
				"language": {
					Type:        gollem.TypeString,
					Description: "Filter by programming language (e.g., 'go', 'python', 'javascript')",
					Pattern:     "^[a-zA-Z0-9+#-]+$",
				},
				"path": {
					Type:        gollem.TypeString,
					Description: "Filter by file path pattern (e.g., 'src/', 'test/', '*.go')",
				},
				"filename": {
					Type:        gollem.TypeString,
					Description: "Filter by filename (e.g., 'config.yaml', 'main.go')",
					Pattern:     "^[^/]+$",
				},
				"repo_filter": {
					Type:        gollem.TypeString,
					Description: "Filter repositories by name pattern (case-insensitive substring match)",
				},
			},
		},
		{
			Name:        "github_issue_search",
			Description: "Search for issues and pull requests. Query syntax: 'bug in:title', 'label:security state:open', 'author:octocat type:pr'",
			Parameters: map[string]*gollem.Parameter{
				"query": {
					Type:        gollem.TypeString,
					Description: "Search query using GitHub issue search syntax. Supports operators like in:title, in:body",
					Required:    true,
					MinLength:   github.Ptr(1),
				},
				"state": {
					Type:        gollem.TypeString,
					Description: "Filter by state: 'open', 'closed', or 'all'",
					Enum:        []string{"open", "closed", "all"},
					Default:     "all",
				},
				"labels": {
					Type:        gollem.TypeString,
					Description: "Filter by labels (comma-separated list, e.g., 'bug,help wanted')",
					Pattern:     "^[a-zA-Z0-9-_,\\s]+$",
				},
				"author": {
					Type:        gollem.TypeString,
					Description: "Filter by author username (GitHub username)",
					Pattern:     "^[a-zA-Z0-9][a-zA-Z0-9-]*$",
					MaxLength:   github.Ptr(39),
				},
				"type": {
					Type:        gollem.TypeString,
					Description: "Filter by type: 'issue' for issues only, 'pr' for pull requests only, or 'all' for both",
					Enum:        []string{"issue", "pr", "all"},
					Default:     "all",
				},
				"repo_filter": {
					Type:        gollem.TypeString,
					Description: "Filter repositories by name pattern (case-insensitive substring match)",
				},
			},
		},
		{
			Name:        "github_get_content",
			Description: "Get file content from a specific GitHub repository. Returns the decoded content of the file.",
			Parameters: map[string]*gollem.Parameter{
				"owner": {
					Type:        gollem.TypeString,
					Description: "Repository owner (organization or username)",
					Required:    true,
					Pattern:     "^[a-zA-Z0-9][a-zA-Z0-9-]*$",
					MinLength:   github.Ptr(1),
					MaxLength:   github.Ptr(39),
				},
				"repo": {
					Type:        gollem.TypeString,
					Description: "Repository name",
					Required:    true,
					Pattern:     "^[a-zA-Z0-9_.-]+$",
					MinLength:   github.Ptr(1),
					MaxLength:   github.Ptr(100),
				},
				"path": {
					Type:        gollem.TypeString,
					Description: "File path in the repository (e.g., 'src/main.go', 'README.md')",
					Required:    true,
					MinLength:   github.Ptr(1),
				},
				"ref": {
					Type:        gollem.TypeString,
					Description: "Git reference: branch name (e.g., 'main'), tag (e.g., 'v1.0.0'), or commit SHA. Defaults to the default branch if not specified.",
					Pattern:     "^[a-zA-Z0-9/_.-]+$",
				},
			},
		},
	}, nil
}

// Run implements gollem.ToolSet
func (x *Action) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	if x.githubClient == nil {
		return nil, goerr.New("GitHub client not initialized")
	}

	switch name {
	case "github_code_search":
		return x.runCodeSearch(ctx, args)
	case "github_issue_search":
		return x.runIssueSearch(ctx, args)
	case "github_get_content":
		return x.runGetContent(ctx, args)
	default:
		return nil, goerr.New("unknown tool name", goerr.V("name", name))
	}
}

// Prompt implements interfaces.Tool
func (x *Action) Prompt(ctx context.Context) (string, error) {
	if len(x.configs) == 0 {
		return "", nil
	}

	var sb strings.Builder
	sb.WriteString("## GitHub Repositories\n\n")
	sb.WriteString("The following GitHub repositories are configured and accessible:\n\n")

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
	sb.WriteString("Use the GitHub tools to search code, issues/PRs, or retrieve file contents from these repositories.\n")

	return sb.String(), nil
}

// Helper methods for testing
func (x *Action) SetGitHubClient(client *github.Client) {
	x.githubClient = client
}

func (x *Action) SetConfigs(configs []*RepositoryConfig) {
	x.configs = configs
}

func (x *Action) GetConfigs() []*RepositoryConfig {
	return x.configs
}

func (x *Action) SetTestData(appID, installationID int64, privateKey string, configFiles []string) {
	x.appID = appID
	x.installationID = installationID
	x.privateKey = privateKey
	x.configFiles = configFiles
}
