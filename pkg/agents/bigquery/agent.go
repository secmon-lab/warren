package bigquery

import (
	"context"
	"fmt"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/service/memory"
	"github.com/secmon-lab/warren/pkg/utils/logging"
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

	logging.From(ctx).Info("BigQuery Agent configured",
		"tables", len(cfg.Tables),
		"scan_limit", cfg.ScanSizeLimit)

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
			Description: "Execute high-level BigQuery data extraction tasks. Provide a natural language query describing what data you want, and the agent will handle table selection, query construction, and execution using past experiences.",
			Parameters: map[string]*gollem.Parameter{
				"query": {
					Type:        gollem.TypeString,
					Description: "Natural language description of the data you want to retrieve (e.g., 'login errors in the past week')",
				},
				"limit": {
					Type:        gollem.TypeInteger,
					Description: "Maximum number of records expected to retrieve. This helps clarify the scope and set appropriate query limits.",
				},
			},
			Required: []string{"query"},
		},
	}, nil
}

// Run implements gollem.ToolSet
func (a *Agent) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	if name != "query_bigquery" {
		return nil, goerr.New("unknown function", goerr.V("name", name))
	}

	query, ok := args["query"].(string)
	if !ok {
		return nil, goerr.New("query parameter is required")
	}

	// Extract limit parameter if provided
	var limit int
	if limitVal, ok := args["limit"]; ok {
		switch v := limitVal.(type) {
		case int:
			limit = v
		case float64:
			limit = int(v)
		case float32:
			limit = int(v)
		case int64:
			limit = int(v)
		}
	}

	log := logging.From(ctx)
	startTime := time.Now()

	// Step 1: Search for relevant memories
	memories, err := a.memoryService.SearchRelevantAgentMemories(ctx, a.ID(), query, 5)
	if err != nil {
		log.Warn("failed to search memories", "error", err)
	}

	// Step 2: Build system prompt with memories
	systemPrompt, err := a.buildSystemPromptWithMemories(ctx, memories)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to build system prompt")
	}

	// Step 3: Construct gollem.Agent with BigQuery tools
	agent := gollem.New(
		a.llmClient,
		gollem.WithToolSets(a.internalTool),
		gollem.WithSystemPrompt(systemPrompt),
	)

	// Step 4: Build task prompt with limit if specified
	taskPrompt := query
	if limit > 0 {
		taskPrompt = fmt.Sprintf("%s (limit results to %d records)", query, limit)
	}

	// Step 5: Execute task
	resp, execErr := agent.Execute(ctx, gollem.Text(taskPrompt))
	duration := time.Since(startTime)

	// Step 6: Save execution memory (metadata only)
	if err := a.saveExecutionMemory(ctx, query, resp, execErr, duration); err != nil {
		log.Warn("failed to save execution memory", "error", err)
	}

	// Step 7: Return execution result
	if execErr != nil {
		return nil, execErr
	}

	result := map[string]any{
		"result": "",
		"data":   nil,
	}
	if resp != nil && !resp.IsEmpty() {
		result["result"] = resp.String()
	}
	return result, nil
}
