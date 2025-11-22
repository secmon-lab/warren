package slack

import (
	"context"
	"log/slog"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	slackSDK "github.com/slack-go/slack"

	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/service/memory"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
	"github.com/urfave/cli/v3"
)

// Agent represents a Slack Search Sub-Agent
type Agent struct {
	internalTool  gollem.ToolSet
	llmClient     gollem.LLMClient
	memoryService *memory.Service
	slackClient   interfaces.SlackClient

	// CLI configuration field
	oauthToken string
}

// New creates a new Slack Search Agent instance
func New(opts ...Option) *Agent {
	a := &Agent{}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// Option is a functional option for configuring Agent
type Option func(*Agent)

// WithSlackClient sets the Slack client
func WithSlackClient(client interfaces.SlackClient) Option {
	return func(a *Agent) {
		a.slackClient = client
		if client != nil {
			a.internalTool = &internalTool{slackClient: client}
		}
	}
}

// WithLLMClient sets the LLM client
func WithLLMClient(client gollem.LLMClient) Option {
	return func(a *Agent) {
		a.llmClient = client
	}
}

// WithMemoryService sets the memory service
func WithMemoryService(svc *memory.Service) Option {
	return func(a *Agent) {
		a.memoryService = svc
	}
}

// ID implements SubAgent interface
func (a *Agent) ID() string {
	return "slack_search"
}

// Flags returns CLI flags for this agent
func (a *Agent) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "agent-slack-user-token",
			Usage:       "Slack User OAuth Token for message search (requires search:read scope)",
			Destination: &a.oauthToken,
			Category:    "Agent:Slack",
			Sources:     cli.EnvVars("WARREN_AGENT_SLACK_USER_TOKEN"),
		},
	}
}

// Init initializes the agent with LLM client and memory service.
// Returns (true, nil) if initialized successfully, (false, nil) if not configured, or (false, error) on error.
func (a *Agent) Init(ctx context.Context, llmClient gollem.LLMClient, memoryService *memory.Service) (bool, error) {
	// If no OAuth token provided, agent is not configured
	if a.oauthToken == "" && a.slackClient == nil {
		return false, nil // Agent is optional
	}

	// Create Slack client from OAuth token if not already set
	if a.slackClient == nil {
		a.slackClient = slackSDK.New(a.oauthToken)
	}

	a.internalTool = &internalTool{
		slackClient: a.slackClient,
	}
	a.llmClient = llmClient
	a.memoryService = memoryService

	logging.From(ctx).Info("Slack Search Agent configured")

	return true, nil
}

// IsEnabled returns true if the agent is configured and initialized
func (a *Agent) IsEnabled() bool {
	return a.slackClient != nil && a.llmClient != nil
}

// SetSlackClient sets the Slack client
func (a *Agent) SetSlackClient(client interfaces.SlackClient) {
	a.slackClient = client
}

// Specs implements gollem.ToolSet
func (a *Agent) Specs(ctx context.Context) ([]gollem.ToolSpec, error) {
	// Return empty specs if agent is not enabled
	if !a.IsEnabled() {
		return []gollem.ToolSpec{}, nil
	}

	return []gollem.ToolSpec{
		{
			Name:        "search_slack",
			Description: "Search for messages in Slack workspace. This tool ONLY searches and retrieves messages - it does NOT analyze or interpret the data. After receiving the search results, YOU must analyze them yourself and provide a complete answer to the user based on the retrieved messages.",
			Parameters: map[string]*gollem.Parameter{
				"query": {
					Type:        gollem.TypeString,
					Description: "ONLY specify the search conditions (e.g., 'messages in #security-alerts from last week', 'messages containing error from @user'). Do NOT include analysis instructions or interpretation requests - ONLY search conditions.",
				},
				"limit": {
					Type:        gollem.TypeNumber,
					Description: "Maximum number of messages to return in the response (default: 50, max: 200). Use this to control response size.",
				},
			},
			Required: []string{"query"},
		},
	}, nil
}

// Run implements gollem.ToolSet
func (a *Agent) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	log := logging.From(ctx)
	log.Debug("Slack Search agent run started", "function", name, "args", args)

	if name != "search_slack" {
		log.Debug("Unknown function name", "name", name)
		return nil, goerr.New("unknown function", goerr.V("name", name))
	}

	query, ok := args["query"].(string)
	if !ok {
		log.Debug("Query parameter is missing or invalid")
		return nil, goerr.New("query parameter is required")
	}

	// Get limit parameter
	limit := 50 // default
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}
	if limit > 200 {
		limit = 200
	}

	log.Debug("Processing query", "query", query, "limit", limit)
	msg.Trace(ctx, "🔵 *[Slack Search Agent]* Query: `%s` (limit: %d)", query, limit)

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

	// Step 2: Build system prompt with memories and limit
	log.Debug("Building system prompt with memories and limit", "memory_count", len(memories), "limit", limit)
	systemPrompt, err := buildSystemPrompt(ctx, limit, memories)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to build system prompt")
	}
	log.Debug("System prompt built", "prompt_length", len(systemPrompt))

	// Step 3: Set limit in internal tool and construct internal agent with Slack tools
	log.Debug("Setting limit in internal tool", "limit", limit)
	a.internalTool.(*internalTool).maxLimit = limit

	log.Debug("Constructing internal agent with Slack tools")
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
							msg.Trace(ctx, "  🔹 %s", text)
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

	// Step 5: Save execution memory (metadata only)
	log.Debug("Saving execution memory", "has_error", execErr != nil)
	if err := a.saveExecutionMemory(ctx, query, resp, execErr, duration, agent.Session()); err != nil {
		// Memory save failure is non-critical for the main task
		log.Warn("Failed to save execution memory", "error", err)
		msg.Trace(ctx, "  ⚠️ *Warning:* Failed to save execution memory")
	} else {
		msg.Trace(ctx, "  ✅ *[Slack Search Agent]* Execution memory saved (Duration: %s)",
			duration.Round(time.Millisecond))
	}

	// Step 5.5: Collect and apply feedback for used memories
	if len(memories) > 0 {
		log.Debug("Collecting feedback for used memories", "count", len(memories))
		if err := a.memoryService.CollectAndApplyFeedback(ctx, a.ID(), memories, query, agent.Session(), resp, execErr); err != nil {
			// Feedback collection failure is non-critical
			log.Warn("Failed to collect and apply feedback", "error", err)
			msg.Trace(ctx, "  ⚠️ *Warning:* Failed to collect feedback for memories")
		}
	}

	// Step 6: Return execution result
	if execErr != nil {
		log.Debug("Execution failed, returning error", "error", execErr)
		return nil, execErr
	}

	result := map[string]any{
		"data": "",
	}
	if resp != nil && !resp.IsEmpty() {
		dataStr := resp.String()
		result["data"] = dataStr
		log.Debug("Execution successful", "data_length", len(dataStr))
	} else {
		log.Debug("Execution returned empty response")
	}
	return result, nil
}

// Name implements interfaces.Tool
func (a *Agent) Name() string {
	return "slack_search"
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
	)
}

// Helper implements interfaces.Tool
func (a *Agent) Helper() *cli.Command {
	return nil
}

// Prompt implements interfaces.Tool
// Returns basic description for system prompt
func (a *Agent) Prompt(ctx context.Context) (string, error) {
	if !a.IsEnabled() {
		return "", nil
	}

	return "Slack message search is available for searching historical Slack messages.", nil
}

var _ interfaces.Tool = (*Agent)(nil)
