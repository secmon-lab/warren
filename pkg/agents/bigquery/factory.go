package bigquery

import (
	"context"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/service/memory"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/urfave/cli/v3"
)

// Factory implements agents.AgentFactory interface.
type Factory struct {
	configPath                string
	projectID                 string
	scanSizeLimitStr          string
	runbookPaths              []string
	impersonateServiceAccount string
}

// Flags implements agents.AgentFactory
func (f *Factory) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "agent-bigquery-config",
			Usage:       "Path to BigQuery Agent configuration file (YAML)",
			Destination: &f.configPath,
			Category:    "Agent:BigQuery",
			Sources:     cli.EnvVars("WARREN_AGENT_BIGQUERY_CONFIG"),
		},
		&cli.StringFlag{
			Name:        "agent-bigquery-project-id",
			Usage:       "Google Cloud Project ID for BigQuery operations",
			Destination: &f.projectID,
			Category:    "Agent:BigQuery",
			Sources:     cli.EnvVars("WARREN_AGENT_BIGQUERY_PROJECT_ID"),
		},
		&cli.StringFlag{
			Name:        "agent-bigquery-scan-size-limit",
			Usage:       "Maximum scan size limit for BigQuery queries (e.g., '10GB', '1TB')",
			Destination: &f.scanSizeLimitStr,
			Category:    "Agent:BigQuery",
			Sources:     cli.EnvVars("WARREN_AGENT_BIGQUERY_SCAN_SIZE_LIMIT"),
		},
		&cli.StringSliceFlag{
			Name:        "agent-bigquery-runbook-dir",
			Usage:       "Path to SQL runbook files or directories",
			Destination: &f.runbookPaths,
			Category:    "Agent:BigQuery",
			Sources:     cli.EnvVars("WARREN_AGENT_BIGQUERY_RUNBOOK_DIR"),
		},
		&cli.StringFlag{
			Name:        "agent-bigquery-impersonate-service-account",
			Usage:       "Service account email to impersonate for BigQuery operations",
			Destination: &f.impersonateServiceAccount,
			Category:    "Agent:BigQuery",
			Sources:     cli.EnvVars("WARREN_AGENT_BIGQUERY_IMPERSONATE_SERVICE_ACCOUNT"),
		},
	}
}

// Configure implements agents.AgentFactory
func (f *Factory) Configure(ctx context.Context, llmClient gollem.LLMClient, repo interfaces.Repository) (*gollem.SubAgent, string, error) {
	if f.configPath == "" {
		return nil, "", nil
	}

	// Load config and runbooks
	cfg, err := LoadConfigWithRunbooks(ctx, f.configPath, f.runbookPaths)
	if err != nil {
		return nil, "", goerr.Wrap(err, "failed to load BigQuery Agent config")
	}

	// Override scan size limit from CLI flag if provided
	if f.scanSizeLimitStr != "" {
		limit, err := ParseScanSizeLimit(f.scanSizeLimitStr)
		if err != nil {
			return nil, "", goerr.Wrap(err, "failed to parse scan size limit")
		}
		cfg.ScanSizeLimit = limit
	}

	// Create internal agent
	a := &agent{
		config:    cfg,
		llmClient: llmClient,
		repo:      repo,
		internalTool: &internalTool{
			config:                    cfg,
			projectID:                 f.projectID,
			impersonateServiceAccount: f.impersonateServiceAccount,
		},
		memoryService: memory.New("bigquery", llmClient, repo),
	}

	// Log configuration
	scanLimit := humanize.Bytes(cfg.ScanSizeLimit)
	var tables []string
	for _, t := range cfg.Tables {
		tables = append(tables, strings.Join([]string{
			t.ProjectID, t.DatasetID, t.TableID,
		}, "."))
	}
	logging.From(ctx).Info("BigQuery Agent configured",
		"tables", tables,
		"scan_limit", scanLimit,
		"runbooks", len(cfg.Runbooks))

	// Build prompt hint for parent agent
	promptHint, err := buildPromptHint(cfg)
	if err != nil {
		return nil, "", goerr.Wrap(err, "failed to build prompt hint")
	}

	// Create and return SubAgent with prompt hint
	subAgent, err := a.subAgent()
	if err != nil {
		return nil, "", err
	}

	return subAgent, promptHint, nil
}
