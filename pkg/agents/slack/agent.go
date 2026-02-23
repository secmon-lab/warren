package slack

import (
	"context"
	"fmt"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/service/memory"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
	"github.com/secmon-lab/warren/pkg/utils/slackctx"
)

// agent represents a Slack Search Sub-Agent (private).
// This struct is private and only created through the factory.
type agent struct {
	internalTool  gollem.ToolSet
	llmClient     gollem.LLMClient
	memoryService *memory.Service
	slackClient   interfaces.SlackClient
	repo          interfaces.Repository
}

// name returns the agent name (private method)
func (a *agent) name() string {
	return "search_slack"
}

// description returns the agent description (private method)
func (a *agent) description() string {
	return "Search for messages in Slack workspace. This tool delegates to a specialized Slack search agent that will understand your request, search comprehensively, and return a response containing the relevant raw message data organized to fulfill your request. The agent will include actual message content (text, user, channel, timestamp) as raw data. You should use this data to answer the user's question."
}

// subAgent creates a gollem.SubAgent (private method)
func (a *agent) subAgent() (*gollem.SubAgent, error) {
	promptTemplate, err := newPromptTemplate()
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create prompt template")
	}

	return gollem.NewSubAgent(
		a.name(),
		a.description(),
		a.factory,
		gollem.WithPromptTemplate(promptTemplate),
		gollem.WithSubAgentMiddleware(a.createMiddleware()),
	), nil
}

// factory creates an internal agent (private method)
func (a *agent) factory() (*gollem.Agent, error) {
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

// createMiddleware creates middleware for pre/post-execution processing (private method)
func (a *agent) createMiddleware() func(gollem.SubAgentHandler) gollem.SubAgentHandler {
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

			// Inject Slack context from parent if available
			if thread := slackctx.ThreadFrom(ctx); thread != nil && thread.ChannelID != "" {
				args["_slack_context"] = fmt.Sprintf(
					"Current Slack context: channel_id=%s, thread_ts=%s, team_id=%s",
					thread.ChannelID, thread.ThreadID, thread.TeamID,
				)
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
			delete(result.Data, "_slack_context")
			delete(result.Data, "_limit")

			return result, nil
		}
	}
}
