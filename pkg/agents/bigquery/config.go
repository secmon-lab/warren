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
	Tables                    []TableConfig                              `yaml:"tables"`
	ScanSizeLimit             uint64                                     `yaml:"-"` // Parsed from ScanSizeLimitStr
	ScanSizeLimitStr          string                                     `yaml:"scan_size_limit"`
	QueryTimeout              time.Duration                              `yaml:"query_timeout"` // Timeout for waiting for BigQuery job completion (default: 5 minutes)
	Runbooks                  map[types.RunbookID]*bigquery.RunbookEntry `yaml:"-"`             // Loaded runbooks (not in YAML)
	ProjectID                 string                                     `yaml:"-"`             // Google Cloud Project ID (set by Warren)
	ImpersonateServiceAccount string                                     `yaml:"-"`             // Service account to impersonate (set by Warren)
}

// TableConfig represents a BigQuery table configuration
type TableConfig struct {
	ProjectID   string `yaml:"project_id"`
	DatasetID   string `yaml:"dataset_id"`
	TableID     string `yaml:"table_id"`
	Description string `yaml:"description,omitempty"`
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

	return nil
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
