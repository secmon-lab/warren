package falcon

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

// agent represents a CrowdStrike Falcon Sub-Agent (private).
type agent struct {
	internalTool gollem.ToolSet
	llmClient    gollem.LLMClient
	repo         interfaces.Repository
}

func (a *agent) name() string {
	return "query_falcon"
}

func (a *agent) description() string {
	return "Query CrowdStrike Falcon (EDR) for endpoint security data. " +
		"Supports: incidents (status, tactics, hosts, scores), " +
		"alerts (severity, MITRE ATT&CK tactics/techniques, hostname, file hash, command line), " +
		"behaviors (detection patterns, dispositions), " +
		"devices/hosts (hostname, IP, OS, sensor version, containment status), " +
		"CrowdScores (environment threat level), " +
		"and raw EDR events (process executions, network connections, DNS requests, file writes). " +
		"Accepts FQL filters (e.g. status:'new', severity:>50, hostname:'*web*') and " +
		"CQL queries for event search. " +
		"Returns raw API records as structured JSON."
}

// subAgent creates a gollem.SubAgent.
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

// factory creates an internal agent.
func (a *agent) factory() (*gollem.Agent, error) {
	systemPrompt := buildSystemPrompt()

	return gollem.New(
		a.llmClient,
		gollem.WithToolSets(a.internalTool),
		gollem.WithSystemPrompt(systemPrompt),
	), nil
}

// createMiddleware creates middleware for pre/post-execution processing.
func (a *agent) createMiddleware() func(gollem.SubAgentHandler) gollem.SubAgentHandler {
	return func(next gollem.SubAgentHandler) gollem.SubAgentHandler {
		return func(ctx context.Context, args map[string]any) (gollem.SubAgentResult, error) {
			log := logging.From(ctx)

			// Get request parameter
			request, ok := args["request"].(string)
			if !ok || request == "" {
				return next(ctx, args)
			}

			log.Debug("Processing Falcon request", "request", request)
			msg.Trace(ctx, "🦅 *[Falcon Agent]* Request: `%s`", request)

			startTime := time.Now()

			args["_original_request"] = request

			// Execute internal agent
			result, err := next(ctx, args)
			duration := time.Since(startTime)
			log.Debug("Falcon request execution completed", "duration", duration, "has_error", err != nil)

			if err != nil {
				return gollem.SubAgentResult{}, err
			}

			// Record extraction (critical)
			records, err := a.extractRecords(ctx, request, result.Session)
			if err != nil {
				log.Warn("Failed to extract records, falling back to text response", "error", err)
				msg.Warn(ctx, "⚠️ *Warning:* Failed to extract records, returning text response")

				if textResp, ok := result.Data["response"].(string); ok && textResp != "" {
					result.Data["response"] = textResp
				} else {
					result.Data["response"] = ""
				}
			} else {
				log.Debug("Successfully extracted records", "count", len(records))
				result.Data["records"] = records
			}

			// If no records were extracted, check if the response indicates a complete failure
			// (e.g., authentication errors preventing all API calls)
			if len(records) == 0 {
				if textResp, ok := result.Data["response"].(string); ok && textResp != "" && containsErrorIndicators(textResp) {
					log.Warn("Falcon sub-agent returned no records with error response", "response", textResp)
					return gollem.SubAgentResult{}, goerr.New("Falcon API query failed: all operations returned errors",
						goerr.V("response", textResp),
					)
				}
			}

			// Clean up internal fields
			delete(result.Data, "_original_request")
			delete(result.Data, "_memories")
			delete(result.Data, "_memory_context")

			return result, nil
		}
	}
}

// containsErrorIndicators checks if a response text indicates that all API operations
// failed due to authentication or connectivity issues.
func containsErrorIndicators(text string) bool {
	lower := strings.ToLower(text)
	indicators := []string{
		"authentication error",
		"oauth2 token request failed",
		"token request failed",
		"credentials are not configured",
		"credentials have expired",
		"credentials may no longer be valid",
		"api endpoint is unreachable",
		"api credentials",
	}
	for _, indicator := range indicators {
		if strings.Contains(lower, indicator) {
			return true
		}
	}
	return false
}
