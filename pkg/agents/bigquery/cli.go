package bigquery

import (
	"github.com/m-mizutani/goerr/v2"
	"github.com/urfave/cli/v3"
)

// CLIConfig represents CLI configuration for BigQuery Agent
type CLIConfig struct {
	ConfigPath       string
	ScanSizeLimitStr string
}

// Flags returns CLI flags for BigQuery Agent configuration
func (x *CLIConfig) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "agent-bigquery-config",
			Usage:       "Path to BigQuery Agent configuration file (YAML)",
			Destination: &x.ConfigPath,
		},
		&cli.StringFlag{
			Name:        "agent-bigquery-scan-size-limit",
			Usage:       "Maximum scan size limit for BigQuery queries (e.g., '10GB', '1TB')",
			Destination: &x.ScanSizeLimitStr,
		},
	}
}

// LoadConfig loads and validates BigQuery Agent configuration
func (x *CLIConfig) LoadConfig() (*Config, error) {
	if x.ConfigPath == "" {
		return nil, nil // BigQuery Agent is optional
	}

	cfg, err := LoadConfig(x.ConfigPath)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to load BigQuery Agent config")
	}

	// Override scan size limit from CLI flag if provided
	if x.ScanSizeLimitStr != "" {
		limit, err := ParseScanSizeLimit(x.ScanSizeLimitStr)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to parse scan size limit")
		}
		cfg.ScanSizeLimit = limit
	}

	return cfg, nil
}
