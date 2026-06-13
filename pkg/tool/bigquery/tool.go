package bigquery

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gollem-dev/gollem"
	extbq "github.com/gollem-dev/tools/bigquery"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/bigquery"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

// Action is the warren-side wrapper around github.com/gollem-dev/tools/bigquery.
// The runtime Specs/Run logic (query execution, dataset/schema inspection,
// runbook retrieval) lives in the external module. warren retains the CLI
// flags, the planner Prompt() built from the configured tables/runbooks, and
// the `generate-config` Helper() subcommand, which the external module does not
// provide.
type Action struct {
	projectID                 string
	impersonateServiceAccount string
	credentials               string
	configFiles               []string
	storageBucket             string
	storagePrefix             string
	timeout                   time.Duration
	scanLimitStr              string
	configs                   []*Config
	runbookPaths              []string

	// In-memory storage for runbooks, used to render Prompt().
	runbooks map[types.RunbookID]*bigquery.RunbookEntry

	opts  []extbq.Option
	inner gollem.ToolSet
}

var _ interfaces.Tool = &Action{}

type Config struct {
	// ProjectID
	// DatasetID
	DatasetID string `yaml:"dataset_id" json:"dataset_id"`

	// TableID
	TableID string `yaml:"table_id" json:"table_id"`

	// Description of the table
	Description string `yaml:"description" json:"description"`

	// Columns of the table. It's not required to describe all column.
	Columns []ColumnConfig `yaml:"columns" json:"columns"`

	// Partitioning
	Partitioning PartitioningConfig `yaml:"partitioning" json:"partitioning"`

	// Runbook paths - list of SQL files or directories
	RunbookPaths []string `yaml:"runbook_paths" json:"runbook_paths"`
}

type PartitioningConfig struct {
	Field string `yaml:"field" json:"field"`

	// Type of the partitioning. `integer`, `time`
	Type string `yaml:"type" json:"type"`

	// Time unit. `hourly`, `daily` or `monthly`
	TimeUnit string `yaml:"time_unit" json:"time_unit"`
}

type ColumnConfig struct {
	Name         string         `yaml:"name" json:"name"`
	Description  string         `yaml:"description" json:"description"`
	ValueExample string         `yaml:"value_example" json:"value_example"`
	Type         string         `yaml:"type" json:"type"`     // STRING, INTEGER, FLOAT, BOOLEAN, TIMESTAMP, DATE, TIME, DATETIME, BYTES, RECORD
	Fields       []ColumnConfig `yaml:"fields" json:"fields"` // for RECORD type
}

func (x *Action) ID() string {
	return "bigquery"
}

func (x *Action) Description() string {
	return "BigQuery SQL data queries and schema inspection"
}

func (x *Action) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "bigquery-project-id",
			Usage:       "Google Cloud Project ID",
			Destination: &x.projectID,
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_BIGQUERY_PROJECT_ID"),
		},
		&cli.StringFlag{
			Name:        "bigquery-credentials",
			Usage:       "Path to Google Cloud credentials JSON file (optional)",
			Destination: &x.credentials,
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_BIGQUERY_CREDENTIALS"),
			Action: func(_ context.Context, _ *cli.Command, v string) error {
				x.opts = append(x.opts, extbq.WithCredentials(v))
				return nil
			},
		},
		&cli.StringFlag{
			Name:        "bigquery-impersonate-service-account",
			Usage:       "Service account email for impersonation",
			Destination: &x.impersonateServiceAccount,
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_BIGQUERY_IMPERSONATE_SERVICE_ACCOUNT"),
			Action: func(_ context.Context, _ *cli.Command, v string) error {
				x.opts = append(x.opts, extbq.WithImpersonateServiceAccount(v))
				return nil
			},
		},
		&cli.StringSliceFlag{
			Name:        "bigquery-config",
			Usage:       "Path to configuration YAML file",
			Destination: &x.configFiles,
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_BIGQUERY_CONFIG"),
			Action: func(_ context.Context, _ *cli.Command, v []string) error {
				x.opts = append(x.opts, extbq.WithConfigFiles(v))
				return nil
			},
		},
		&cli.StringSliceFlag{
			Name:        "bigquery-runbook-path",
			Usage:       "Path to SQL runbook files or directories",
			Destination: &x.runbookPaths,
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_BIGQUERY_RUNBOOK_PATH"),
			Action: func(_ context.Context, _ *cli.Command, v []string) error {
				x.opts = append(x.opts, extbq.WithRunbookPaths(v))
				return nil
			},
		},
		&cli.StringFlag{
			Name:        "bigquery-storage-bucket",
			Usage:       "GCS bucket name for storing query results",
			Destination: &x.storageBucket,
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_BIGQUERY_STORAGE_BUCKET"),
			Action: func(_ context.Context, _ *cli.Command, v string) error {
				x.opts = append(x.opts, extbq.WithStorageBucket(v))
				return nil
			},
		},
		&cli.StringFlag{
			Name:        "bigquery-storage-prefix",
			Usage:       "Prefix for GCS object path for storing query results",
			Destination: &x.storagePrefix,
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_BIGQUERY_STORAGE_PREFIX"),
			Action: func(_ context.Context, _ *cli.Command, v string) error {
				x.opts = append(x.opts, extbq.WithStoragePrefix(v))
				return nil
			},
		},
		&cli.DurationFlag{
			Name:        "bigquery-timeout",
			Usage:       "Timeout for query execution",
			Destination: &x.timeout,
			Category:    "Tool",
			Value:       5 * time.Minute,
			Sources:     cli.EnvVars("WARREN_BIGQUERY_TIMEOUT"),
			Action: func(_ context.Context, _ *cli.Command, v time.Duration) error {
				x.opts = append(x.opts, extbq.WithTimeout(v))
				return nil
			},
		},
		&cli.StringFlag{
			Name:        "bigquery-scan-limit",
			Usage:       "Scan limit for query execution",
			Destination: &x.scanLimitStr,
			Category:    "Tool",
			Value:       "10GB",
			Sources:     cli.EnvVars("WARREN_BIGQUERY_SCAN_LIMIT"),
			Action: func(_ context.Context, _ *cli.Command, v string) error {
				x.opts = append(x.opts, extbq.WithScanLimit(v))
				return nil
			},
		},
	}
}

func (x *Action) Configure(ctx context.Context) error {
	if x.projectID == "" {
		return errutil.ErrActionUnavailable
	}

	if len(x.configFiles) == 0 {
		logging.Default().Warn("project ID is provided, but no configuration file is provided. bigquery tool is disabled.")
		return errutil.ErrActionUnavailable
	}

	// Initialize runbooks map if not already done
	if x.runbooks == nil {
		x.runbooks = make(map[types.RunbookID]*bigquery.RunbookEntry)
	}

	var configs []*Config
	for _, configPath := range x.configFiles {
		fileInfo, err := os.Stat(configPath)
		if err != nil {
			return goerr.Wrap(err, "failed to stat config path", goerr.V("path", configPath))
		}

		if fileInfo.IsDir() {
			dirConfigs, err := loadConfigsFromDirInternal(configPath)
			if err != nil {
				return err
			}
			configs = append(configs, dirConfigs...)
		} else {
			config, err := loadConfigFromFileInternal(configPath)
			if err != nil {
				return err
			}
			configs = append(configs, config)
		}
	}

	if len(configs) == 0 {
		return goerr.New("no valid configuration files found")
	}

	x.configs = configs

	// Load runbooks if paths are configured. warren keeps its own copy to
	// render Prompt(); the external toolset loads them independently for the
	// get_runbook_entry tool.
	if len(x.runbookPaths) > 0 {
		if err := x.loadRunbooks(ctx); err != nil {
			return goerr.Wrap(err, "failed to load runbooks")
		}
	}

	ts, err := extbq.New(x.projectID, x.opts...)
	if err != nil {
		return goerr.Wrap(err, "failed to configure BigQuery tool")
	}
	x.inner = ts

	return nil
}

func loadConfigFromFileInternal(filePath string) (*Config, error) {
	data, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to read configuration file", goerr.V("path", filePath))
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, goerr.Wrap(err, "failed to parse configuration file", goerr.V("path", filePath))
	}

	return &config, nil
}

func loadConfigsFromDirInternal(dirPath string) ([]*Config, error) {
	var configs []*Config

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return goerr.Wrap(err, "failed to walk directory", goerr.V("path", path))
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		config, err := loadConfigFromFileInternal(path)
		if err != nil {
			return err
		}

		configs = append(configs, config)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return configs, nil
}

func (x *Action) Specs(ctx context.Context) ([]gollem.ToolSpec, error) {
	if x.inner == nil {
		return nil, goerr.New("BigQuery tool is not configured")
	}
	return x.inner.Specs(ctx)
}

func (x *Action) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	if x.inner == nil {
		return nil, goerr.New("BigQuery tool is not configured")
	}
	return x.inner.Run(ctx, name, args)
}

func (x *Action) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("project_id", x.projectID),
		slog.String("credentials", x.credentials),
		slog.Any("config_files", x.configFiles),
		slog.String("storage_bucket", x.storageBucket),
		slog.Duration("timeout", x.timeout),
	)
}

// Prompt returns additional instructions for the system prompt
// It provides information about available BigQuery tables and their descriptions
func (x *Action) Prompt(ctx context.Context) (string, error) {
	if len(x.configs) == 0 {
		return "", nil
	}

	data := map[string]any{
		"projectID": x.projectID,
		"configs":   x.configs,
		"runbooks":  x.runbooks,
	}

	return bigquerySystemPrompt(data)
}

// loadRunbooks loads SQL runbooks from configured paths and stores them in memory
func (x *Action) loadRunbooks(ctx context.Context) error {
	x.runbooks = make(map[types.RunbookID]*bigquery.RunbookEntry)

	// Create runbook loader
	loader := NewRunbookLoader(x.runbookPaths)

	// Load runbooks from files
	entries, err := loader.LoadRunbooks(ctx)
	if err != nil {
		return goerr.Wrap(err, "failed to load runbooks from files")
	}

	// Store entries in memory with their IDs
	for _, entry := range entries {
		x.runbooks[entry.ID] = entry
	}

	return nil
}
