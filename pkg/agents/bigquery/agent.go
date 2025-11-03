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
	configPath       string
	projectID        string
	scanSizeLimitStr string
}

// New creates a new BigQuery Agent instance
func New() *Agent {
	return &Agent{}
}

// NewAgent creates a new BigQuery Agent instance with config (for testing and direct use)
func NewAgent(
	config *Config,
	llmClient gollem.LLMClient,
	memoryService *memory.Service,
) *Agent {
	return &Agent{
		config:        config,
		internalTool:  &internalTool{config: config},
		llmClient:     llmClient,
		memoryService: memoryService,
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
	}
}

// Init initializes the agent with LLM client and memory service.
// Returns (true, nil) if initialized successfully, (false, nil) if not configured, or (false, error) on error.
func (a *Agent) Init(ctx context.Context, llmClient gollem.LLMClient, memoryService *memory.Service) (bool, error) {
	if a.configPath == "" {
		return false, nil // Agent is optional
	}

	cfg, err := LoadConfig(a.configPath)
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
		config:    cfg,
		projectID: a.projectID,
	}
	a.llmClient = llmClient
	a.memoryService = memoryService

	scanLimit := humanize.Bytes(cfg.ScanSizeLimit)
	var tables []string
	for _, t := range cfg.Tables {
		tables = append(tables, strings.Join([]string{
			t.ProjectID, t.DatasetID, t.TableID,
		}, "."))
	}
	logging.From(ctx).Info("BigQuery Agent configured",
		"tables", tables,
		"scan_limit", scanLimit)

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
			Description: "Execute high-level BigQuery data extraction tasks. Provide a natural language query describing what data you want, and the agent will handle table selection, query construction, and execution using past experiences. The agent will automatically check table schemas before constructing queries and return raw data records.",
			Parameters: map[string]*gollem.Parameter{
				"query": {
					Type:        gollem.TypeString,
					Description: "Natural language description of the data you want to retrieve (e.g., 'login errors in the past week')",
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
	msg.Trace(ctx, "[bigquery agent] query: `%s`", query)

	startTime := time.Now()

	// Step 1: Search for relevant memories
	log.Debug("Searching for relevant memories", "agent_id", a.ID(), "limit", 5)
	memories, err := a.memoryService.SearchRelevantAgentMemories(ctx, a.ID(), query, 5)
	if err != nil {
		// Memory search failure is non-critical - continue with empty memories
		log.Warn("Memory search failed, continuing without memories", "error", err)
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
	log.Debug("System prompt built", "prompt_length", len(systemPrompt))

	// Step 3: Construct gollem.Agent with BigQuery tools
	log.Debug("Constructing internal agent with BigQuery tools")
	agent := gollem.New(
		a.llmClient,
		gollem.WithToolSets(a.internalTool),
		gollem.WithSystemPrompt(systemPrompt),
	)

	// Step 4: Execute task
	log.Debug("Executing query task via internal agent", "query", query)
	resp, execErr := agent.Execute(ctx, gollem.Text(query))
	duration := time.Since(startTime)
	log.Debug("Query task execution completed", "duration", duration, "has_error", execErr != nil)

	// Step 5: Save execution memory (metadata only)
	log.Debug("Saving execution memory", "has_error", execErr != nil)
	if err := a.saveExecutionMemory(ctx, query, resp, execErr, duration, agent.Session()); err != nil {
		// Memory save failure is non-critical for the main task
		log.Warn("Failed to save execution memory", "error", err)
	}

	// Step 6: Return execution result
	if execErr != nil {
		log.Debug("Execution failed, returning error", "error", execErr)
		return nil, execErr
	}

	result := map[string]any{
		"result": "",
		"data":   nil,
	}
	if resp != nil && !resp.IsEmpty() {
		result["result"] = resp.String()
		log.Debug("Execution successful", "result_length", len(resp.String()))
	} else {
		log.Debug("Execution returned empty response")
	}
	return result, nil
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
