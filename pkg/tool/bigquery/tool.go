package bigquery

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/bigquery"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

type Action struct {
	projectID                 string
	impersonateServiceAccount string
	credentials               string
	configFiles               []string
	storageBucket             string
	storagePrefix             string
	timeout                   time.Duration
	scanLimitStr              string
	scanLimit                 uint64
	configs                   []*Config
	runbookPaths              []string

	// In-memory storage for runbooks
	runbooks map[types.RunbookID]*bigquery.RunbookEntry
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

func (x *Action) Name() string {
	return "bigquery"
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
		},
		&cli.StringFlag{
			Name:        "bigquery-impersonate-service-account",
			Usage:       "Service account email for impersonation",
			Destination: &x.impersonateServiceAccount,
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_BIGQUERY_IMPERSONATE_SERVICE_ACCOUNT"),
		},
		&cli.StringSliceFlag{
			Name:        "bigquery-config",
			Usage:       "Path to configuration YAML file",
			Destination: &x.configFiles,
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_BIGQUERY_CONFIG"),
		},
		&cli.StringSliceFlag{
			Name:        "bigquery-runbook-path",
			Usage:       "Path to SQL runbook files or directories",
			Destination: &x.runbookPaths,
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_BIGQUERY_RUNBOOK_PATH"),
		},
		&cli.StringFlag{
			Name:        "bigquery-storage-bucket",
			Usage:       "GCS bucket name for storing query results",
			Destination: &x.storageBucket,
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_BIGQUERY_STORAGE_BUCKET"),
		},
		&cli.StringFlag{
			Name:        "bigquery-storage-prefix",
			Usage:       "Prefix for GCS object path for storing query results",
			Destination: &x.storagePrefix,
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_BIGQUERY_STORAGE_PREFIX"),
		},
		&cli.DurationFlag{
			Name:        "bigquery-timeout",
			Usage:       "Timeout for query execution",
			Destination: &x.timeout,
			Category:    "Tool",
			Value:       5 * time.Minute,
			Sources:     cli.EnvVars("WARREN_BIGQUERY_TIMEOUT"),
		},
		&cli.StringFlag{
			Name:        "bigquery-scan-limit",
			Usage:       "Scan limit for query execution",
			Destination: &x.scanLimitStr,
			Category:    "Tool",
			Value:       "10GB",
			Sources:     cli.EnvVars("WARREN_BIGQUERY_SCAN_LIMIT"),
		},
	}
}

func (x *Action) Configure(ctx context.Context) error {
	if x.projectID == "" {
		return errs.ErrActionUnavailable
	}

	if len(x.configFiles) == 0 {
		logging.Default().Warn("project ID is provided, but no configuration file is provided. bigquery tool is disabled.")
		return errs.ErrActionUnavailable
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

	scanLimit, err := humanize.ParseBytes(x.scanLimitStr)
	if err != nil {
		return goerr.Wrap(err, "failed to parse scan limit")
	}
	x.scanLimit = scanLimit

	x.configs = configs

	// Load runbooks if paths are configured
	if len(x.runbookPaths) > 0 {
		if err := x.loadRunbooks(ctx); err != nil {
			return goerr.Wrap(err, "failed to load runbooks")
		}
	}

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
	return []gollem.ToolSpec{
		{
			Name:        "bigquery_list_dataset",
			Description: "List available BigQuery datasets, tables and partial schema that is necessary for investigation",
			Parameters:  map[string]*gollem.Parameter{},
		},
		{
			Name:        "bigquery_query",
			Description: bigqueryQueryPrompt(x.scanLimitStr),
			Parameters: map[string]*gollem.Parameter{
				"query": {
					Type:        gollem.TypeString,
					Description: "The SQL query to execute",
				},
			},
			Required: []string{"query"},
		},
		{
			Name:        "bigquery_result",
			Description: "Get the results of a previously executed query. Returns rows as JSON string in 'rows_json' field due to Vertex AI type limitations.",
			Parameters: map[string]*gollem.Parameter{
				"query_id": {
					Type:        gollem.TypeString,
					Description: "The ID of the query to get results for",
				},
				"limit": {
					Type:        gollem.TypeInteger,
					Description: "Maximum number of rows to return",
				},
				"offset": {
					Type:        gollem.TypeInteger,
					Description: "Number of rows to skip",
				},
			},
			Required: []string{"query_id"},
		},
		{
			Name:        "bigquery_table_summary",
			Description: "Get a summary of available BigQuery tables including all fields, examples, and descriptions",
			Parameters: map[string]*gollem.Parameter{
				"project_id": {
					Type:        gollem.TypeString,
					Description: "The project ID to filter by (optional)",
				},
				"dataset_id": {
					Type:        gollem.TypeString,
					Description: "The dataset ID to filter by (optional)",
				},
				"table_id": {
					Type:        gollem.TypeString,
					Description: "The table ID to filter by (optional)",
				},
			},
		},
		{
			Name:        "bigquery_schema",
			Description: "Get detailed schema information for a specific BigQuery table",
			Parameters: map[string]*gollem.Parameter{
				"project_id": {
					Type:        gollem.TypeString,
					Description: "The project ID of the table",
				},
				"dataset_id": {
					Type:        gollem.TypeString,
					Description: "The dataset ID of the table",
				},
				"table_id": {
					Type:        gollem.TypeString,
					Description: "The table ID to get schema for",
				},
			},
			Required: []string{"project_id", "dataset_id", "table_id"},
		},
		{
			Name:        "get_runbook_entry",
			Description: "Get a specific runbook entry by its ID. Returns the SQL content and description of the runbook.",
			Parameters: map[string]*gollem.Parameter{
				"runbook_id": {
					Type:        gollem.TypeString,
					Description: "The ID of the runbook entry to retrieve",
				},
			},
			Required: []string{"runbook_id"},
		},
	}, nil
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
