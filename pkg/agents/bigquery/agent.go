package bigquery

import (
	"context"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/service/memory"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
	"github.com/urfave/cli/v3"
)

// Agent represents a BigQuery Sub-Agent
type Agent struct {
	config        *Config
	internalTool  gollem.ToolSet
	llmClient     gollem.LLMClient
	memoryService *memory.Service
	repo          interfaces.Repository
}

// CLIConfig holds CLI flag values for BigQuery Agent
type CLIConfig struct {
	configPath                string
	projectID                 string
	scanSizeLimitStr          string
	runbookPaths              []string
	impersonateServiceAccount string
}

// Flags returns CLI flags for BigQuery agent configuration
func Flags() ([]cli.Flag, *CLIConfig) {
	cfg := &CLIConfig{}
	flags := []cli.Flag{
		&cli.StringFlag{
			Name:        "agent-bigquery-config",
			Usage:       "Path to BigQuery Agent configuration file (YAML)",
			Category:    "Agent:BigQuery",
			Sources:     cli.EnvVars("WARREN_AGENT_BIGQUERY_CONFIG"),
			Destination: &cfg.configPath,
		},
		&cli.StringFlag{
			Name:        "agent-bigquery-project-id",
			Usage:       "Google Cloud Project ID for BigQuery operations",
			Category:    "Agent:BigQuery",
			Sources:     cli.EnvVars("WARREN_AGENT_BIGQUERY_PROJECT_ID"),
			Destination: &cfg.projectID,
		},
		&cli.StringFlag{
			Name:        "agent-bigquery-scan-size-limit",
			Usage:       "Maximum scan size limit for BigQuery queries (e.g., '10GB', '1TB')",
			Category:    "Agent:BigQuery",
			Sources:     cli.EnvVars("WARREN_AGENT_BIGQUERY_SCAN_SIZE_LIMIT"),
			Destination: &cfg.scanSizeLimitStr,
		},
		&cli.StringSliceFlag{
			Name:        "agent-bigquery-runbook-dir",
			Usage:       "Path to SQL runbook files or directories",
			Category:    "Agent:BigQuery",
			Sources:     cli.EnvVars("WARREN_AGENT_BIGQUERY_RUNBOOK_DIR"),
			Destination: &cfg.runbookPaths,
		},
		&cli.StringFlag{
			Name:        "agent-bigquery-impersonate-service-account",
			Usage:       "Service account email to impersonate for BigQuery operations",
			Category:    "Agent:BigQuery",
			Sources:     cli.EnvVars("WARREN_AGENT_BIGQUERY_IMPERSONATE_SERVICE_ACCOUNT"),
			Destination: &cfg.impersonateServiceAccount,
		},
	}
	return flags, cfg
}

// Init initializes and returns a fully configured immutable Agent
func Init(ctx context.Context, cliCfg *CLIConfig, llmClient gollem.LLMClient, repo interfaces.Repository) (*Agent, error) {
	if cliCfg.configPath == "" {
		return nil, nil // Agent is optional
	}

	// Load config with runbooks
	config, err := LoadConfigWithRunbooks(ctx, cliCfg.configPath, cliCfg.runbookPaths)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to load BigQuery Agent config")
	}

	// Override scan size limit from CLI flag if provided
	if cliCfg.scanSizeLimitStr != "" {
		limit, err := ParseScanSizeLimit(cliCfg.scanSizeLimitStr)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to parse scan size limit")
		}
		config.ScanSizeLimit = limit
	}

	// Override project ID and impersonate service account
	if cliCfg.projectID != "" {
		config.ProjectID = cliCfg.projectID
	}
	if cliCfg.impersonateServiceAccount != "" {
		config.ImpersonateServiceAccount = cliCfg.impersonateServiceAccount
	}

	scanLimit := humanize.Bytes(config.ScanSizeLimit)
	var tables []string
	for _, t := range config.Tables {
		tables = append(tables, strings.Join([]string{
			t.ProjectID, t.DatasetID, t.TableID,
		}, "."))
	}
	logging.From(ctx).Info("BigQuery Agent configured",
		"tables", tables,
		"scan_limit", scanLimit,
		"runbooks", len(config.Runbooks))

	// Return immutable Agent
	return New(llmClient, repo, config), nil
}

// New creates a new immutable BigQuery Agent instance
func New(llmClient gollem.LLMClient, repo interfaces.Repository, config *Config) *Agent {
	return &Agent{
		llmClient: llmClient,
		repo:      repo,
		config:    config,
		internalTool: &internalTool{
			config:                    config,
			projectID:                 config.ProjectID,
			impersonateServiceAccount: config.ImpersonateServiceAccount,
		},
		memoryService: memory.New("bigquery", llmClient, repo),
	}
}

// IsEnabled returns true if the agent is initialized
func (a *Agent) IsEnabled() bool {
	return a.config != nil
}

// ID implements SubAgent interface
func (a *Agent) ID() string {
	return "bigquery"
}

// Name returns the agent name
func (a *Agent) Name() string {
	return "query_bigquery"
}

// Description returns the agent description
func (a *Agent) Description() string {
	return "Retrieve data from BigQuery tables. This tool ONLY extracts data records - it does NOT analyze or interpret the data. After receiving the data, YOU must analyze it yourself and provide a complete answer to the user based on the retrieved data. The tool handles table selection, query construction, and returns raw data records."
}

// SubAgent returns a gollem.SubAgent
func (a *Agent) SubAgent() (*gollem.SubAgent, error) {
	promptTemplate, err := newPromptTemplate()
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create prompt template")
	}

	return gollem.NewSubAgent(
		a.Name(),
		a.Description(),
		a.factory,
		gollem.WithPromptTemplate(promptTemplate),
		gollem.WithSubAgentMiddleware(a.createMiddleware()),
	), nil
}

// factory creates an internal agent
func (a *Agent) factory() (*gollem.Agent, error) {
	systemPrompt, err := buildSystemPrompt(a.config)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to build system prompt")
	}

	return gollem.New(
		a.llmClient,
		gollem.WithToolSets(a.internalTool),
		gollem.WithSystemPrompt(systemPrompt),
	), nil
}

// createMiddleware creates middleware for pre/post-execution processing
func (a *Agent) createMiddleware() func(gollem.SubAgentHandler) gollem.SubAgentHandler {
	return func(next gollem.SubAgentHandler) gollem.SubAgentHandler {
		return func(ctx context.Context, args map[string]any) (gollem.SubAgentResult, error) {
			log := logging.From(ctx)

			// Get query parameter
			query, ok := args["query"].(string)
			if !ok || query == "" {
				return next(ctx, args)
			}

			log.Debug("Processing query", "query", query)
			msg.Trace(ctx, "🔷 *[BigQuery Agent]* Query: `%s`", query)

			startTime := time.Now()

			// Pre-execution: Memory search (agent's responsibility)
			log.Debug("Searching for relevant memories", "agent_id", "bigquery", "limit", 16)
			memories, err := a.memoryService.SearchAndSelectMemories(ctx, query, 16)
			if err != nil {
				// Memory search failure is non-critical - continue with empty memories
				log.Warn("memory search failed, continuing without memories", "error", err)
				memories = nil
			} else {
				log.Debug("Memories found", "count", len(memories))
			}

			// Inject memory context into args
			if len(memories) > 0 {
				args["_memory_context"] = formatMemoryContext(memories)
			}
			args["_original_query"] = query
			args["_memories"] = memories

			// Execute internal agent
			result, err := next(ctx, args)
			duration := time.Since(startTime)
			log.Debug("Query task execution completed", "duration", duration, "has_error", err != nil)

			if err != nil {
				return gollem.SubAgentResult{}, err
			}

			// Post-execution: Memory extraction (agent's responsibility) - NON-CRITICAL
			history, err := result.Session.History()
			if err != nil {
				log.Warn("failed to get history", "error", err)
			} else {
				if err := a.memoryService.ExtractAndSaveMemories(ctx, query, memories, history); err != nil {
					log.Warn("memory extraction failed", "error", err)
					msg.Warn(ctx, "⚠️ *Warning:* Failed to save execution memories")
				}
			}

			// Record extraction (agent's responsibility) - CRITICAL
			records, err := a.extractRecords(ctx, query, result.Session)
			if err != nil {
				// Fallback to text response
				log.Warn("Failed to extract records, falling back to text response", "error", err)
				msg.Warn(ctx, "⚠️ *Warning:* Failed to extract records, returning text response")

				// Get text from session if available
				if textResp, ok := result.Data["response"].(string); ok && textResp != "" {
					result.Data["data"] = textResp
				} else {
					result.Data["data"] = ""
				}
				delete(result.Data, "response")
			} else {
				log.Debug("Successfully extracted records", "count", len(records))
				result.Data["records"] = records
			}

			// Clean up internal fields
			delete(result.Data, "_original_query")
			delete(result.Data, "_memories")
			delete(result.Data, "_memory_context")

			return result, nil
		}
	}
}
