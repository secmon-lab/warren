package slack

import (
	"context"
	"log/slog"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	slackSDK "github.com/slack-go/slack"

	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/service/memory"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
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
func (a *Agent) Init(ctx context.Context, llmClient gollem.LLMClient, repo interfaces.Repository) (bool, error) {
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
	// Create memory service bound to this agent
	a.memoryService = memory.New(a.ID(), llmClient, repo)

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
			Description: "Search for messages in Slack workspace. This tool delegates to a specialized Slack search agent that will understand your request, search comprehensively, and return a response containing the relevant raw message data organized to fulfill your request. The agent will include actual message content (text, user, channel, timestamp) as raw data. You should use this data to answer the user's question.",
			Parameters: map[string]*gollem.Parameter{
				"request": {
					Type:        gollem.TypeString,
					Description: "DO NOT specify search keywords or terms. Describe ONLY the concept/situation in natural language. The agent will determine all search keywords and variations. ✗ BAD: 'search for authentication keyword', 'messages containing auth error', 'find keyword login' ✓ GOOD: 'people having authentication problems', 'discussions about performance issues', 'error reports in #security-alerts channel'. Include: (1) What concept/situation to find (NOT keywords), (2) Time period if relevant, (3) Channel/user scope if relevant. The Slack agent handles all keyword selection, variations, and multilingual terms automatically.",
					Required:    true,
				},
				"limit": {
					Type:        gollem.TypeNumber,
					Description: "Maximum number of messages to return in the response (default: 50, max: 200). Use this to control response size.",
				},
			},
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

	request, ok := args["request"].(string)
	if !ok {
		log.Debug("Request parameter is missing or invalid")
		return nil, goerr.New("request parameter is required")
	}

	// Get limit parameter
	limit := 50 // default
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}
	if limit > 200 {
		limit = 200
	}

	log.Debug("Processing request", "request", request, "limit", limit)
	msg.Trace(ctx, "🔵 *[Slack Search Agent]* Request: `%s` (limit: %d)", request, limit)

	startTime := time.Now()

	// Step 1: Search for relevant memories
	log.Debug("Searching for relevant memories", "agent_id", a.ID(), "limit", 16)
	memories, err := a.memoryService.SearchAndSelectMemories(ctx, request, 16)
	if err != nil {
		// Memory search failure is non-critical - continue with empty memories
		errutil.Handle(ctx, goerr.Wrap(err, "memory search failed, continuing without memories"))
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
	log.Debug("System prompt built", "prompt", systemPrompt)

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
				// Show tool name and search query for slack_search_messages

				if req.Tool.Name == "slack_search_messages" {
					query, _ := req.Tool.Arguments["query"].(string)
					msg.Trace(ctx, "  🔸 *Tool:* `%s` (query: `%s`)", req.Tool.Name, query)
				} else {
					msg.Trace(ctx, "  🔸 *Tool:* `%s`", req.Tool.Name)
				}
				log.Debug("execute tool", "tool", req.Tool.Name, "args", req.Tool.Arguments)

				resp, err := next(ctx, req)

				if resp != nil && resp.Error != nil {
					msg.Trace(ctx, "  ❌ *Error:* %s", resp.Error.Error())
					log.Error("tool error", "error", resp.Error, "call", req.Tool)
				} else if resp != nil && req.Tool.Name == "slack_search_messages" {
					// For slack_search_messages, only show count instead of full results
					if msgs, ok := resp.Result["messages"].([]any); ok {
						msg.Trace(ctx, "  ✅ *Found:* %d messages", len(msgs))
					}
				}

				return resp, err
			}
		}),
	)

	// Step 4: Execute task
	log.Debug("Executing request task via internal agent", "request", request)
	resp, execErr := agent.Execute(ctx, gollem.Text(request))
	duration := time.Since(startTime)
	log.Debug("Query task execution completed", "duration", duration, "has_error", execErr != nil)

	// Step 5: Extract and save memories from execution
	log.Debug("Extracting and saving memories from execution")
	// Get the full session history for memory extraction
	history, err := agent.Session().History()
	if err != nil {
		errutil.Handle(ctx, goerr.Wrap(err, "failed to get session history"))
	} else if err := a.memoryService.ExtractAndSaveMemories(ctx, request, memories, history); err != nil {
		// Memory extraction failure is non-critical
		errutil.Handle(ctx, goerr.Wrap(err, "failed to extract and save memories"))
		msg.Warn(ctx, "⚠️ *Warning:* Failed to save execution memories")
	}

	// Step 6: Return execution result
	if execErr != nil {
		log.Debug("Execution failed, returning error", "error", execErr)
		return nil, execErr
	}

	// Step 7: Extract records from session history
	log.Debug("Extracting messages from session history")
	records, err := a.extractRecords(ctx, request, agent.Session())
	if err != nil {
		// Fallback to original response
		log.Warn("Failed to extract messages, falling back to text response", "error", err)
		msg.Warn(ctx, "⚠️ *Warning:* Failed to extract messages, returning text response")
		result := map[string]any{"response": ""}
		if resp != nil && !resp.IsEmpty() {
			result["response"] = resp.String()
		}
		return result, nil
	}

	log.Debug("Successfully extracted messages", "count", len(records))
	return map[string]any{"records": records}, nil
}

// Name implements interfaces.Tool
func (a *Agent) Name() string {
	return "slack_search"
}

// Configure implements interfaces.Tool
func (a *Agent) Configure(ctx context.Context) error {
	if !a.IsEnabled() {
		return errutil.ErrActionUnavailable
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
