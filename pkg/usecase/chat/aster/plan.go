package aster

import (
	"context"
	"fmt"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/trace"
	"github.com/secmon-lab/warren/pkg/domain/types"
	knowledgeTool "github.com/secmon-lab/warren/pkg/tool/knowledge"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

// plan executes the planning phase and returns a structured plan.
// If a knowledge service is configured, the planner runs as a gollem Agent with
// knowledge_search tool so it can look up prior findings before creating the plan.
func (c *AsterChat) plan(ctx context.Context, session gollem.Session, pc *planningContext, systemPrompt string) (*PlanResult, error) {
	logger := logging.From(ctx)

	// Generate planning prompt
	planPrompt, err := generatePlanPrompt(ctx, pc)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate plan prompt")
	}

	// Trace: wrap planning in a child agent span
	handler := trace.HandlerFrom(ctx)
	if handler != nil {
		ctx = handler.StartChildAgent(ctx, "planning")
		defer func() { handler.EndChildAgent(ctx, err) }()
	}

	logger.Debug("executing planning phase")

	var texts []string

	if c.knowledgeService != nil {
		// Agent mode: planner can call knowledge_search before generating the plan
		texts, err = c.executePlannerAgent(ctx, session, systemPrompt, planPrompt, planSchema)
	} else {
		// Direct mode: generate plan JSON without tool calls
		var resp *gollem.Response
		resp, err = session.Generate(ctx, []gollem.Input{gollem.Text(planPrompt)})
		if err == nil {
			texts = resp.Texts
		}
	}
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate plan")
	}

	result, err := parsePlanResult(texts)
	if err != nil {
		return nil, err
	}

	logger.Info("plan generated",
		"task_count", len(result.Tasks),
		"has_message", result.Message != "")

	return result, nil
}

// executePlannerAgent runs the planner as a gollem Agent with knowledge tool,
// then appends the agent's conversation history back to the planning session.
func (c *AsterChat) executePlannerAgent(ctx context.Context, planSession gollem.Session, systemPrompt string, userPrompt string, schema *gollem.Parameter) ([]string, error) {
	// Build agent options
	opts := []gollem.Option{
		gollem.WithSystemPrompt(systemPrompt),
		gollem.WithContentType(gollem.ContentTypeJSON),
		gollem.WithResponseSchema(schema),
		gollem.WithResponseMode(gollem.ResponseModeBlocking),
	}

	// Add knowledge tool
	kt := knowledgeTool.New(c.knowledgeService, types.KnowledgeCategoryFact, knowledgeTool.ModeSearchOnly)
	opts = append(opts, gollem.WithToolSets(kt))

	// Carry over existing conversation history
	history, err := planSession.History()
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get planning session history")
	}
	if history != nil {
		opts = append(opts, gollem.WithHistory(history))
	}

	// Add trace handler if available
	handler := trace.HandlerFrom(ctx)
	if handler != nil {
		opts = append(opts, gollem.WithTrace(handler))
	}

	agent := gollem.New(c.llmClient, opts...)
	resp, err := agent.Execute(ctx, gollem.Text(userPrompt))
	if err != nil {
		return nil, err
	}

	// Sync only the NEW messages from the agent back to the planning session.
	// The agent was initialized with the existing planSession history, so we must
	// skip those initial messages to avoid duplicating them.
	agentHistory, err := agent.Session().History()
	if err != nil {
		logging.From(ctx).Warn("failed to get agent history after planning", "error", err)
	} else if agentHistory != nil {
		initialLen := 0
		if history != nil {
			initialLen = len(history.Messages)
		}
		if len(agentHistory.Messages) > initialLen {
			newHistory := &gollem.History{
				Version:  agentHistory.Version,
				Messages: agentHistory.Messages[initialLen:],
			}
			if err := planSession.AppendHistory(newHistory); err != nil {
				logging.From(ctx).Warn("failed to append agent history to planning session", "error", err)
			}
		}
	}

	if resp == nil || resp.IsEmpty() {
		return nil, nil
	}

	return resp.Texts, nil
}

// replan evaluates completed results and determines next steps.
func (c *AsterChat) replan(ctx context.Context, session gollem.Session, pc *planningContext, allResults []*phaseResult, currentPhase int, systemPrompt string) (*ReplanResult, error) {
	logger := logging.From(ctx)

	// Generate replan prompt
	replanPrompt, err := generateReplanPrompt(ctx, pc, allResults, currentPhase)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate replan prompt")
	}

	// Trace: wrap replan in a child agent span
	handler := trace.HandlerFrom(ctx)
	if handler != nil {
		ctx = handler.StartChildAgent(ctx, fmt.Sprintf("replan-phase-%d", currentPhase))
		defer func() { handler.EndChildAgent(ctx, err) }()
	}

	logger.Debug("executing replan phase", "phase", currentPhase)

	var texts []string

	if c.knowledgeService != nil {
		// Agent mode: replan with knowledge search
		texts, err = c.executePlannerAgent(ctx, session, systemPrompt, replanPrompt, replanSchema)
	} else {
		// Direct mode: create a replan session and generate
		var replanSession gollem.Session
		replanSession, err = c.llmClient.NewSession(ctx,
			gollem.WithSessionContentType(gollem.ContentTypeJSON),
			gollem.WithSessionResponseSchema(replanSchema),
			gollem.WithSessionSystemPrompt(systemPrompt),
		)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to create replan session")
		}

		// Copy history from planning session
		history, hErr := session.History()
		if hErr != nil {
			return nil, goerr.Wrap(hErr, "failed to get history from planning session")
		}
		if history != nil {
			if aErr := replanSession.AppendHistory(history); aErr != nil {
				logger.Warn("failed to append history to replan session", "error", aErr)
			}
		}

		var resp *gollem.Response
		resp, err = replanSession.Generate(ctx, []gollem.Input{gollem.Text(replanPrompt)})
		if err == nil {
			texts = resp.Texts
		}
	}
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate replan")
	}

	result, err := parseReplanResult(texts)
	if err != nil {
		return nil, err
	}

	logger.Info("replan completed",
		"phase", currentPhase,
		"new_task_count", len(result.Tasks))

	return result, nil
}

// generateFinalResponse generates the final response after all tasks are done.
func (c *AsterChat) generateFinalResponse(ctx context.Context, session gollem.Session, pc *planningContext, allResults []*phaseResult, systemPrompt string) (string, error) {
	logger := logging.From(ctx)

	finalPrompt, err := generateFinalPrompt(ctx, pc, allResults)
	if err != nil {
		return "", goerr.Wrap(err, "failed to generate final prompt")
	}

	// Trace: wrap final response in a child agent span
	handler := trace.HandlerFrom(ctx)
	if handler != nil {
		ctx = handler.StartChildAgent(ctx, "final-response")
		defer func() { handler.EndChildAgent(ctx, err) }()
	}

	logger.Debug("generating final response")

	// Create a text session for the final response (not JSON)
	finalSession, err := c.llmClient.NewSession(ctx,
		gollem.WithSessionSystemPrompt(systemPrompt),
	)
	if err != nil {
		return "", goerr.Wrap(err, "failed to create final response session")
	}

	// Copy history from planning session
	history, err := session.History()
	if err != nil {
		return "", goerr.Wrap(err, "failed to get history from planning session")
	}
	if history != nil {
		if err := finalSession.AppendHistory(history); err != nil {
			logger.Warn("failed to append history to final session", "error", err)
		}
	}

	resp, err := finalSession.Generate(ctx, []gollem.Input{gollem.Text(finalPrompt)})
	if err != nil {
		return "", goerr.Wrap(err, "failed to generate final response")
	}

	if len(resp.Texts) == 0 {
		return "", goerr.New("no response from final LLM")
	}

	logger.Debug("final response generated")
	return resp.Texts[0], nil
}
