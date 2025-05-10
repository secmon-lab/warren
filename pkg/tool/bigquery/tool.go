package bigquery

import (
	"context"
	"os"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

type Action struct {
	projectID     string
	credentials   string
	configFile    string
	storageBucket string
	storagePrefix string
	timeout       time.Duration
	scanLimitStr  string
	scanLimit     int64
	config        *Config
}

var _ interfaces.Tool = &Action{}

type Config struct {
	Datasets map[string]DatasetConfig `yaml:"datasets" json:"datasets"`
}

type DatasetConfig struct {
	Tables []TableConfig `yaml:"tables" json:"tables"`
}

type TableConfig struct {
	// TableID
	TableID string `yaml:"table_id" json:"table_id"`

	// Description of the table
	Description string `yaml:"description" json:"description"`

	// Columns of the table. It's not required to describe all column.
	Columns []ColumnConfig `yaml:"columns" json:"columns"`
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
			Name:        "bigquery-config",
			Usage:       "Path to configuration YAML file",
			Destination: &x.configFile,
			Category:    "Tool",
			Sources:     cli.EnvVars("WARREN_BIGQUERY_CONFIG"),
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

	if x.configFile == "" {
		return goerr.New("configuration file is required")
	}

	// Read and parse the configuration file
	data, err := os.ReadFile(x.configFile)
	if err != nil {
		return goerr.Wrap(err, "failed to read configuration file")
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return goerr.Wrap(err, "failed to parse configuration file")
	}

	scanLimit, err := humanize.ParseBytes(x.scanLimitStr)
	if err != nil {
		return goerr.Wrap(err, "failed to parse scan limit")
	}
	x.scanLimit = int64(scanLimit)

	x.config = &config
	return nil
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
			Description: "Get the results of a previously executed query",
			Parameters: map[string]*gollem.Parameter{
				"query_id": {
					Type:        gollem.TypeString,
					Description: "The ID of the query to get results for",
				},
				"limit": {
					Type:        gollem.TypeInteger,
					Description: "Maximum number of rows to return (default: 100)",
					Required:    []string{},
				},
				"offset": {
					Type:        gollem.TypeInteger,
					Description: "Number of rows to skip (default: 0)",
					Required:    []string{},
				},
			},
			Required: []string{"query_id"},
		},
		{
			Name:        "bigquery_schema",
			Description: "Get schema information for a specific table",
			Parameters: map[string]*gollem.Parameter{
				"dataset_id": {
					Type:        gollem.TypeString,
					Description: "The dataset ID containing the table",
				},
				"table_id": {
					Type:        gollem.TypeString,
					Description: "The table ID to get schema for",
				},
			},
			Required: []string{"dataset_id", "table_id"},
		},
	}, nil
}
