package bigquery

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/urfave/cli/v3"
)

// CLI integration for BigQuery Agent

// Flags returns CLI flags for BigQuery Agent configuration
func Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:     "agent-bigquery-config",
			Usage:    "Path to BigQuery Agent configuration file (YAML)",
			Category: "Agent:BigQuery",
			Sources:  cli.EnvVars("WARREN_AGENT_BIGQUERY_CONFIG"),
		},
		&cli.StringFlag{
			Name:     "agent-bigquery-project-id",
			Usage:    "Google Cloud Project ID for BigQuery operations",
			Category: "Agent:BigQuery",
			Sources:  cli.EnvVars("WARREN_AGENT_BIGQUERY_PROJECT_ID"),
		},
		&cli.StringFlag{
			Name:     "agent-bigquery-scan-size-limit",
			Usage:    "Maximum scan size limit for BigQuery queries (e.g., '10GB', '1TB')",
			Category: "Agent:BigQuery",
			Sources:  cli.EnvVars("WARREN_AGENT_BIGQUERY_SCAN_SIZE_LIMIT"),
		},
		&cli.StringSliceFlag{
			Name:     "agent-bigquery-runbook-dir",
			Usage:    "Path to SQL runbook files or directories",
			Category: "Agent:BigQuery",
			Sources:  cli.EnvVars("WARREN_AGENT_BIGQUERY_RUNBOOK_DIR"),
		},
		&cli.StringFlag{
			Name:     "agent-bigquery-impersonate-service-account",
			Usage:    "Service account email to impersonate for BigQuery operations",
			Category: "Agent:BigQuery",
			Sources:  cli.EnvVars("WARREN_AGENT_BIGQUERY_IMPERSONATE_SERVICE_ACCOUNT"),
		},
	}
}

// NewSubAgentFromCLI creates a BigQuery SubAgent from CLI context
// Returns nil if the agent is not configured (no config path provided)
func NewSubAgentFromCLI(ctx context.Context, c *cli.Command, llmClient gollem.LLMClient, repo interfaces.Repository) (*gollem.SubAgent, error) {
	configPath := c.String("agent-bigquery-config")
	if configPath == "" {
		return nil, nil
	}

	// Load config with runbooks
	runbookPaths := c.StringSlice("agent-bigquery-runbook-dir")
	config, err := LoadConfigWithRunbooks(ctx, configPath, runbookPaths)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to load BigQuery Agent config")
	}

	// Override scan size limit from CLI flag if provided
	scanSizeLimitStr := c.String("agent-bigquery-scan-size-limit")
	if scanSizeLimitStr != "" {
		limit, err := ParseScanSizeLimit(scanSizeLimitStr)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to parse scan size limit")
		}
		config.ScanSizeLimit = limit
	}

	// Set project ID and impersonate service account
	config.ProjectID = c.String("agent-bigquery-project-id")
	config.ImpersonateServiceAccount = c.String("agent-bigquery-impersonate-service-account")

	// Create agent
	agent := New(ctx, config, llmClient, repo)
	subAgent, err := agent.SubAgent()
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create BigQuery SubAgent")
	}

	return subAgent, nil
}
