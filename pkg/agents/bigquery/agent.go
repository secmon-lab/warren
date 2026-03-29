package bigquery

import (
	"context"
	"strings"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

// agent represents a BigQuery Sub-Agent (private).
// This struct is private and only created through the factory.
type agent struct {
	config       *Config
	internalTool gollem.ToolSet
	llmClient    gollem.LLMClient
	repo         interfaces.Repository
}

// name returns the agent name (private method)
func (a *agent) name() string {
	return "query_bigquery"
}

// description returns the agent description (private method).
// It dynamically includes available table names so the parent agent
// knows which tables this sub-agent has access to.
func (a *agent) description() string {
	base := "Retrieve data from BigQuery tables. This tool ONLY extracts data records - it does NOT analyze or interpret the data. After receiving the data, YOU must analyze it yourself and provide a complete answer to the user based on the retrieved data. The tool handles table selection, query construction, and returns raw data records."

	if len(a.config.Tables) > 0 {
		var tables []string
		for _, t := range a.config.Tables {
			tables = append(tables, strings.Join([]string{t.ProjectID, t.DatasetID, t.TableID}, "."))
		}
		base += " Available tables: " + strings.Join(tables, ", ") + "."
	}

	return base
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

// createMiddleware creates middleware for pre/post-execution processing (private method)
func (a *agent) createMiddleware() func(gollem.SubAgentHandler) gollem.SubAgentHandler {
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

			args["_original_query"] = query

			// Execute internal agent
			result, err := next(ctx, args)
			duration := time.Since(startTime)
			log.Debug("Query task execution completed", "duration", duration, "has_error", err != nil)

			if err != nil {
				return gollem.SubAgentResult{}, err
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
