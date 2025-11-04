package bigquery

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/bigquery"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"gopkg.in/yaml.v3"
)

// Config represents BigQuery Agent configuration
type Config struct {
	Tables           []TableConfig                              `yaml:"-"`        // Internal use only (expanded from Projects)
	Projects         []ProjectConfig                            `yaml:"projects"` // Hierarchical configuration
	ScanSizeLimit    uint64                                     `yaml:"-"`        // Parsed from ScanSizeLimitStr
	ScanSizeLimitStr string                                     `yaml:"scan_size_limit"`
	QueryTimeout     time.Duration                              `yaml:"query_timeout"` // Timeout for waiting for BigQuery job completion (default: 5 minutes)
	Runbooks         map[types.RunbookID]*bigquery.RunbookEntry `yaml:"-"`             // Loaded runbooks (not in YAML)
}

// ProjectConfig represents a GCP project with datasets
type ProjectConfig struct {
	ID          string          `yaml:"id"`
	Description string          `yaml:"description,omitempty"`
	Datasets    []DatasetConfig `yaml:"datasets"`
}

// DatasetConfig represents a BigQuery dataset with tables
type DatasetConfig struct {
	ID          string        `yaml:"id"`
	Description string        `yaml:"description,omitempty"`
	Tables      []TableDetail `yaml:"tables"`
}

// TableDetail represents a table within a dataset
type TableDetail struct {
	ID          string `yaml:"id"`
	Description string `yaml:"description,omitempty"`
}

// TableConfig represents a BigQuery table configuration (flat structure for backward compatibility)
type TableConfig struct {
	ProjectID   string `yaml:"project_id"`
	DatasetID   string `yaml:"dataset_id"`
	TableID     string `yaml:"table_id"`
	Description string `yaml:"description"`
}

// LoadConfig loads BigQuery Agent configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(filepath.Clean(path)) // #nosec G304 - path is provided by user configuration
	if err != nil {
		return nil, goerr.Wrap(err, "failed to read config file", goerr.V("path", path))
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, goerr.Wrap(err, "failed to unmarshal config", goerr.V("path", path))
	}

	if err := cfg.setDefaults(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// LoadConfigWithRunbooks loads configuration and runbooks
func LoadConfigWithRunbooks(ctx context.Context, path string, runbookPaths []string) (*Config, error) {
	cfg, err := LoadConfig(path)
	if err != nil {
		return nil, err
	}

	if len(runbookPaths) > 0 {
		if err := cfg.loadRunbooks(ctx, runbookPaths); err != nil {
			return nil, goerr.Wrap(err, "failed to load runbooks")
		}
	}

	return cfg, nil
}

// loadRunbooks loads runbooks from specified paths
func (c *Config) loadRunbooks(ctx context.Context, paths []string) error {
	loader := newRunbookLoader(paths)
	entries, err := loader.loadRunbooks(ctx)
	if err != nil {
		return goerr.Wrap(err, "failed to load runbook entries")
	}

	c.Runbooks = make(map[types.RunbookID]*bigquery.RunbookEntry)
	for _, entry := range entries {
		c.Runbooks[entry.ID] = entry
	}

	return nil
}

// setDefaults sets default values for config fields if not specified
func (c *Config) setDefaults() error {
	if c.QueryTimeout == 0 {
		c.QueryTimeout = 5 * time.Minute
	}

	// Parse scan size limit from string
	if c.ScanSizeLimitStr != "" {
		limit, err := ParseScanSizeLimit(c.ScanSizeLimitStr)
		if err != nil {
			return goerr.Wrap(err, "failed to parse scan_size_limit")
		}
		c.ScanSizeLimit = limit
	}

	// Expand hierarchical projects into flat tables list
	c.expandProjects()
	return nil
}

// expandProjects expands hierarchical project/dataset/table structure into flat Tables list
func (c *Config) expandProjects() {
	for _, project := range c.Projects {
		for _, dataset := range project.Datasets {
			for _, table := range dataset.Tables {
				// Build description with context
				description := table.Description
				if description == "" && dataset.Description != "" {
					description = dataset.Description
				}
				if description == "" && project.Description != "" {
					description = project.Description
				}

				c.Tables = append(c.Tables, TableConfig{
					ProjectID:   project.ID,
					DatasetID:   dataset.ID,
					TableID:     table.ID,
					Description: description,
				})
			}
		}
	}
}

// GetQueryTimeout returns the query timeout with fallback to default
func (c *Config) GetQueryTimeout() time.Duration {
	if c.QueryTimeout == 0 {
		return 5 * time.Minute
	}
	return c.QueryTimeout
}

// ParseScanSizeLimit parses human-readable size string (e.g., "10GB") into bytes
func ParseScanSizeLimit(sizeStr string) (uint64, error) {
	if sizeStr == "" {
		return 0, nil
	}

	bytes, err := humanize.ParseBytes(sizeStr)
	if err != nil {
		return 0, goerr.Wrap(err, "failed to parse scan size limit", goerr.V("size_str", sizeStr))
	}

	return bytes, nil
}
