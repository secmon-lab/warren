package bigquery

import (
	"os"

	"github.com/dustin/go-humanize"
	"github.com/m-mizutani/goerr/v2"
	"gopkg.in/yaml.v3"
)

// Config represents BigQuery Agent configuration
type Config struct {
	Tables        []TableConfig `yaml:"tables"`
	ScanSizeLimit uint64        `yaml:"scan_size_limit"`
}

// TableConfig represents a BigQuery table configuration
type TableConfig struct {
	ProjectID   string `yaml:"project_id"`
	DatasetID   string `yaml:"dataset_id"`
	TableID     string `yaml:"table_id"`
	Description string `yaml:"description"`
}

// LoadConfig loads BigQuery Agent configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to read config file", goerr.V("path", path))
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, goerr.Wrap(err, "failed to unmarshal config", goerr.V("path", path))
	}

	return &cfg, nil
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
