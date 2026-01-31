package slack

import (
	"context"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	slackSDK "github.com/slack-go/slack"

	"github.com/secmon-lab/warren/pkg/domain/interfaces"
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

// Name returns the agent name
func (a *Agent) Name() string {
	return "search_slack"
}

// Description returns the agent description
func (a *Agent) Description() string {
	return "Search for messages in Slack workspace. This tool delegates to a specialized Slack search agent that will understand your request, search comprehensively, and return a response containing the relevant raw message data organized to fulfill your request. The agent will include actual message content (text, user, channel, timestamp) as raw data. You should use this data to answer the user's question."
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
	systemPrompt, err := buildSystemPrompt()
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

			// Get request parameter
			request, ok := args["request"].(string)
			if !ok || request == "" {
				return next(ctx, args)
			}

			// Get limit parameter (default: 50)
			limit := 50
			if l, ok := args["limit"].(float64); ok {
				limit = int(l)
			}
			if limit > 200 {
				limit = 200
			}

			log.Debug("Processing request", "request", request, "limit", limit)
			msg.Trace(ctx, "🔵 *[Slack Search Agent]* Request: `%s` (limit: %d)", request, limit)

			startTime := time.Now()

			// Pre-execution: Memory search (agent's responsibility)
			log.Debug("Searching for relevant memories", "agent_id", "slack_search", "limit", 16)
			memories, err := a.memoryService.SearchAndSelectMemories(ctx, request, 16)
			if err != nil {
				// Memory search failure is non-critical - continue with empty memories
				log.Warn("memory search failed, continuing without memories", "error", err)
				memories = nil
			} else {
				log.Debug("Memories found", "count", len(memories))
			}

			// Inject memory context and limit into args
			if len(memories) > 0 {
				args["_memory_context"] = formatMemoryContext(memories)
			}
			args["_original_request"] = request
			args["_memories"] = memories
			args["_limit"] = limit

			// Set limit in internal tool
			a.internalTool.(*internalTool).maxLimit = limit

			// Execute internal agent
			result, err := next(ctx, args)
			duration := time.Since(startTime)
			log.Debug("Request task execution completed", "duration", duration, "has_error", err != nil)

			if err != nil {
				return gollem.SubAgentResult{}, err
			}

			// Post-execution: Memory extraction (agent's responsibility) - NON-CRITICAL
			history, err := result.Session.History()
			if err != nil {
				log.Warn("failed to get history", "error", err)
			} else {
				if err := a.memoryService.ExtractAndSaveMemories(ctx, request, memories, history); err != nil {
					log.Warn("memory extraction failed", "error", err)
					msg.Warn(ctx, "⚠️ *Warning:* Failed to save execution memories")
				}
			}

			// Message extraction (agent's responsibility) - CRITICAL
			records, err := a.extractRecords(ctx, request, result.Session)
			if err != nil {
				// Fallback to text response
				log.Warn("Failed to extract messages, falling back to text response", "error", err)
				msg.Warn(ctx, "⚠️ *Warning:* Failed to extract messages, returning text response")

				// Get text from session if available
				if textResp, ok := result.Data["response"].(string); ok && textResp != "" {
					result.Data["response"] = textResp
				} else {
					result.Data["response"] = ""
				}
			} else {
				log.Debug("Successfully extracted messages", "count", len(records))
				result.Data["records"] = records
			}

			// Clean up internal fields
			delete(result.Data, "_original_request")
			delete(result.Data, "_memories")
			delete(result.Data, "_memory_context")
			delete(result.Data, "_limit")

			return result, nil
		}
	}
}
