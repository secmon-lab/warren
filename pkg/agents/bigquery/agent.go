package bigquery

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
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

	// CLI configuration fields
	configPath                string
	projectID                 string
	scanSizeLimitStr          string
	runbookPaths              []string
	impersonateServiceAccount string
}

// New creates a new BigQuery Agent instance
func New() *Agent {
	return &Agent{}
}

// NewAgent creates a new BigQuery Agent instance with config (for testing and direct use)
func NewAgent(
	config *Config,
	llmClient gollem.LLMClient,
	repo interfaces.Repository,
) *Agent {
	return &Agent{
		config:        config,
		internalTool:  &internalTool{config: config},
		llmClient:     llmClient,
		memoryService: memory.New("bigquery", llmClient, repo),
	}
}

// Flags returns CLI flags for this agent
func (a *Agent) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "agent-bigquery-config",
			Usage:       "Path to BigQuery Agent configuration file (YAML)",
			Destination: &a.configPath,
			Category:    "Agent:BigQuery",
			Sources:     cli.EnvVars("WARREN_AGENT_BIGQUERY_CONFIG"),
		},
		&cli.StringFlag{
			Name:        "agent-bigquery-project-id",
			Usage:       "Google Cloud Project ID for BigQuery operations",
			Destination: &a.projectID,
			Category:    "Agent:BigQuery",
			Sources:     cli.EnvVars("WARREN_AGENT_BIGQUERY_PROJECT_ID"),
		},
		&cli.StringFlag{
			Name:        "agent-bigquery-scan-size-limit",
			Usage:       "Maximum scan size limit for BigQuery queries (e.g., '10GB', '1TB')",
			Destination: &a.scanSizeLimitStr,
			Category:    "Agent:BigQuery",
			Sources:     cli.EnvVars("WARREN_AGENT_BIGQUERY_SCAN_SIZE_LIMIT"),
		},
		&cli.StringSliceFlag{
			Name:        "agent-bigquery-runbook-dir",
			Usage:       "Path to SQL runbook files or directories",
			Destination: &a.runbookPaths,
			Category:    "Agent:BigQuery",
			Sources:     cli.EnvVars("WARREN_AGENT_BIGQUERY_RUNBOOK_DIR"),
		},
		&cli.StringFlag{
			Name:        "agent-bigquery-impersonate-service-account",
			Usage:       "Service account email to impersonate for BigQuery operations",
			Destination: &a.impersonateServiceAccount,
			Category:    "Agent:BigQuery",
			Sources:     cli.EnvVars("WARREN_AGENT_BIGQUERY_IMPERSONATE_SERVICE_ACCOUNT"),
		},
	}
}

// Init initializes the agent with LLM client and memory service.
// Returns (true, nil) if initialized successfully, (false, nil) if not configured, or (false, error) on error.
func (a *Agent) Init(ctx context.Context, llmClient gollem.LLMClient, repo interfaces.Repository) (bool, error) {
	if a.configPath == "" {
		return false, nil // Agent is optional
	}

	cfg, err := LoadConfigWithRunbooks(ctx, a.configPath, a.runbookPaths)
	if err != nil {
		return false, goerr.Wrap(err, "failed to load BigQuery Agent config")
	}

	// Override scan size limit from CLI flag if provided
	if a.scanSizeLimitStr != "" {
		limit, err := ParseScanSizeLimit(a.scanSizeLimitStr)
		if err != nil {
			return false, goerr.Wrap(err, "failed to parse scan size limit")
		}
		cfg.ScanSizeLimit = limit
	}

	a.config = cfg
	a.internalTool = &internalTool{
		config:                    cfg,
		projectID:                 a.projectID,
		impersonateServiceAccount: a.impersonateServiceAccount,
	}
	a.llmClient = llmClient
	// Create memory service bound to this agent
	a.memoryService = memory.New(a.ID(), llmClient, repo)

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

	return true, nil
}

// IsEnabled returns true if the agent is configured and initialized
func (a *Agent) IsEnabled() bool {
	return a.config != nil
}

// ID implements SubAgent interface
func (a *Agent) ID() string {
	return "bigquery"
}

// Specs implements gollem.ToolSet
func (a *Agent) Specs(ctx context.Context) ([]gollem.ToolSpec, error) {
	// Return empty specs if agent is not enabled
	if !a.IsEnabled() {
		return []gollem.ToolSpec{}, nil
	}

	return []gollem.ToolSpec{
		{
			Name:        "query_bigquery",
			Description: "Retrieve data from BigQuery tables. This tool ONLY extracts data records - it does NOT analyze or interpret the data. After receiving the data, YOU must analyze it yourself and provide a complete answer to the user based on the retrieved data. The tool handles table selection, query construction, and returns raw data records.",
			Parameters: map[string]*gollem.Parameter{
				"query": {
					Type:        gollem.TypeString,
					Description: "ONLY specify the conditions for data retrieval (e.g., 'records containing package name X from the last 7 days', 'login events in the past week'). Do NOT include analysis instructions, interpretation requests, or questions - ONLY data retrieval conditions.",
				},
			},
			Required: []string{"query"},
		},
	}, nil
}

// Run implements gollem.ToolSet
func (a *Agent) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	log := logging.From(ctx)
	log.Debug("BigQuery agent run started", "function", name, "args", args)

	if name != "query_bigquery" {
		log.Debug("Unknown function name", "name", name)
		return nil, goerr.New("unknown function", goerr.V("name", name))
	}

	query, ok := args["query"].(string)
	if !ok {
		log.Debug("Query parameter is missing or invalid")
		return nil, goerr.New("query parameter is required")
	}

	log.Debug("Processing query", "query", query)
	msg.Trace(ctx, "🔷 *[BigQuery Agent]* Query: `%s`", query)

	startTime := time.Now()

	// Step 1: Search for relevant memories
	log.Debug("Searching for relevant memories", "agent_id", a.ID(), "limit", 16)
	memories, err := a.memoryService.SearchAndSelectMemories(ctx, query, 16)
	if err != nil {
		// Memory search failure is non-critical - continue with empty memories
		errs.Handle(ctx, goerr.Wrap(err, "memory search failed, continuing without memories"))
		memories = nil
	} else {
		log.Debug("Memories found", "count", len(memories))
	}

	// Step 2: Build system prompt with memories
	log.Debug("Building system prompt with memories", "memory_count", len(memories))
	systemPrompt, err := a.buildSystemPromptWithMemories(ctx, memories)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to build system prompt")
	}
	log.Debug("System prompt built", "prompt", systemPrompt)

	// Step 3: Construct gollem.Agent with BigQuery tools
	log.Debug("Constructing internal agent with BigQuery tools")
	agent := gollem.New(
		a.llmClient,
		gollem.WithToolSets(a.internalTool),
		gollem.WithSystemPrompt(systemPrompt),
		gollem.WithLogger(log),
		// Trace middleware for sub-agent messages
		gollem.WithContentBlockMiddleware(
			func(next gollem.ContentBlockHandler) gollem.ContentBlockHandler {
				return func(ctx context.Context, req *gollem.ContentRequest) (*gollem.ContentResponse, error) {
					resp, err := next(ctx, req)
					if err == nil && len(resp.Texts) > 0 {
						for _, text := range resp.Texts {
							msg.Trace(ctx, "  💭 %s", text)
						}
					}
					return resp, err
				}
			},
		),
		// Tool execution middleware
		gollem.WithToolMiddleware(func(next gollem.ToolHandler) gollem.ToolHandler {
			return func(ctx context.Context, req *gollem.ToolExecRequest) (*gollem.ToolExecResponse, error) {
				msg.Trace(ctx, "  🔸 *Tool:* `%s`", req.Tool.Name)
				log.Debug("execute tool", "tool", req.Tool.Name, "args", req.Tool.Arguments)

				resp, err := next(ctx, req)

				if resp != nil && resp.Error != nil {
					msg.Trace(ctx, "  ❌ *Error:* %s", resp.Error.Error())
					log.Error("tool error", "error", resp.Error, "call", req.Tool)
				}

				return resp, err
			}
		}),
	)

	// Step 4: Execute task
	log.Debug("Executing query task via internal agent", "query", query)
	resp, execErr := agent.Execute(ctx, gollem.Text(query))
	duration := time.Since(startTime)
	log.Debug("Query task execution completed", "duration", duration, "has_error", execErr != nil)

	// Step 5: Extract and save memories from execution
	log.Debug("Extracting and saving memories from execution")
	// Get the full session history for memory extraction
	history, err := agent.Session().History()
	if err != nil {
		errs.Handle(ctx, goerr.Wrap(err, "failed to get session history"))
	} else if err := a.memoryService.ExtractAndSaveMemories(ctx, query, memories, history); err != nil {
		// Memory extraction failure is non-critical
		errs.Handle(ctx, goerr.Wrap(err, "failed to extract and save memories"))
		msg.Warn(ctx, "⚠️ *Warning:* Failed to save execution memories")
	}

	// Step 6: Return execution result
	if execErr != nil {
		log.Debug("Execution failed, returning error", "error", execErr)
		return nil, execErr
	}

	// Step 7: Extract records from session history
	log.Debug("Extracting records from session history")
	records, err := a.extractRecords(ctx, query, agent.Session())
	if err != nil {
		// Fallback to original response
		log.Warn("Failed to extract records, falling back to text response", "error", err)
		msg.Warn(ctx, "⚠️ *Warning:* Failed to extract records, returning text response")
		result := map[string]any{"data": ""}
		if resp != nil && !resp.IsEmpty() {
			result["data"] = resp.String()
		}
		return result, nil
	}

	log.Debug("Successfully extracted records", "count", len(records))
	return map[string]any{"records": records}, nil
}

// Name implements interfaces.Tool
func (a *Agent) Name() string {
	return "bigquery"
}

// Configure implements interfaces.Tool
func (a *Agent) Configure(ctx context.Context) error {
	if !a.IsEnabled() {
		return errs.ErrActionUnavailable
	}
	return nil
}

// LogValue implements interfaces.Tool
func (a *Agent) LogValue() slog.Value {
	if !a.IsEnabled() {
		return slog.GroupValue(slog.Bool("enabled", false))
	}
	return slog.GroupValue(
		slog.Bool("enabled", true),
		slog.Int("tables", len(a.config.Tables)),
		slog.String("scan_limit", humanize.Bytes(a.config.ScanSizeLimit)),
		slog.Duration("query_timeout", a.config.QueryTimeout),
	)
}

// Helper implements interfaces.Tool
func (a *Agent) Helper() *cli.Command {
	return nil
}

// Prompt implements interfaces.Tool
// Returns table descriptions for system prompt
func (a *Agent) Prompt(ctx context.Context) (string, error) {
	if !a.IsEnabled() {
		return "", nil
	}

	data := struct {
		HasTables     bool
		Tables        []TableConfig
		ScanSizeLimit string
		QueryTimeout  string
	}{
		HasTables:     len(a.config.Tables) > 0,
		Tables:        a.config.Tables,
		ScanSizeLimit: "",
		QueryTimeout:  "",
	}

	if a.config.ScanSizeLimit > 0 {
		data.ScanSizeLimit = humanize.Bytes(a.config.ScanSizeLimit)
	}
	if a.config.QueryTimeout > 0 {
		data.QueryTimeout = a.config.QueryTimeout.String()
	}

	var buf strings.Builder
	if err := toolDescriptionTmpl.Execute(&buf, data); err != nil {
		return "", goerr.Wrap(err, "failed to execute tool description template")
	}

	return buf.String(), nil
}

var _ interfaces.Tool = (*Agent)(nil)
