package bluebell

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
// In bluebell, planning always runs as an agent with knowledge_search tool
// (knowledge service is required).
func (c *BluebellChat) plan(ctx context.Context, session gollem.Session, pc *planningContext, systemPrompt string) (*PlanResult, error) {
	logger := logging.From(ctx)

	planPrompt, err := generatePlanPrompt(ctx, pc)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate plan prompt")
	}

	handler := trace.HandlerFrom(ctx)
	if handler != nil {
		ctx = handler.StartChildAgent(ctx, "planning")
		defer func() { handler.EndChildAgent(ctx, err) }()
	}

	logger.Debug("executing planning phase")

	// Always agent mode: planner can call knowledge_search before generating the plan
	texts, err := c.executePlannerAgent(ctx, session, systemPrompt, planPrompt, planSchema)
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
func (c *BluebellChat) executePlannerAgent(ctx context.Context, planSession gollem.Session, systemPrompt string, userPrompt string, schema *gollem.Parameter) ([]string, error) {
	opts := []gollem.Option{
		gollem.WithSystemPrompt(systemPrompt),
		gollem.WithContentType(gollem.ContentTypeJSON),
		gollem.WithResponseSchema(schema),
		gollem.WithResponseMode(gollem.ResponseModeBlocking),
	}

	kt := knowledgeTool.New(c.knowledgeService, types.KnowledgeCategoryFact, knowledgeTool.ModeSearchOnly)
	opts = append(opts, gollem.WithToolSets(kt))

	history, err := planSession.History()
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get planning session history")
	}
	if history != nil {
		opts = append(opts, gollem.WithHistory(history))
	}

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
func (c *BluebellChat) replan(ctx context.Context, session gollem.Session, pc *planningContext, allResults []*phaseResult, currentPhase int, systemPrompt string) (*ReplanResult, error) {
	logger := logging.From(ctx)

	replanPrompt, err := generateReplanPrompt(ctx, pc, allResults, currentPhase)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate replan prompt")
	}

	handler := trace.HandlerFrom(ctx)
	if handler != nil {
		ctx = handler.StartChildAgent(ctx, fmt.Sprintf("replan-phase-%d", currentPhase))
		defer func() { handler.EndChildAgent(ctx, err) }()
	}

	logger.Debug("executing replan phase", "phase", currentPhase)

	// Always agent mode with knowledge search
	texts, err := c.executePlannerAgent(ctx, session, systemPrompt, replanPrompt, replanSchema)
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
func (c *BluebellChat) generateFinalResponse(ctx context.Context, session gollem.Session, pc *planningContext, allResults []*phaseResult, systemPrompt string) (string, error) {
	logger := logging.From(ctx)

	finalPrompt, err := generateFinalPrompt(ctx, pc, allResults)
	if err != nil {
		return "", goerr.Wrap(err, "failed to generate final prompt")
	}

	handler := trace.HandlerFrom(ctx)
	if handler != nil {
		ctx = handler.StartChildAgent(ctx, "final-response")
		defer func() { handler.EndChildAgent(ctx, err) }()
	}

	logger.Debug("generating final response")

	finalSession, err := c.llmClient.NewSession(ctx,
		gollem.WithSessionSystemPrompt(systemPrompt),
	)
	if err != nil {
		return "", goerr.Wrap(err, "failed to create final response session")
	}

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

	// Sync final session's new messages back to the planning session so the
	// final response is included in saved history for multi-turn conversations.
	finalHistory, err := finalSession.History()
	if err != nil {
		logger.Warn("failed to get final session history", "error", err)
	} else if finalHistory != nil {
		initialLen := 0
		if history != nil {
			initialLen = len(history.Messages)
		}
		if len(finalHistory.Messages) > initialLen {
			newHistory := &gollem.History{
				Version:  finalHistory.Version,
				Messages: finalHistory.Messages[initialLen:],
			}
			if err := session.AppendHistory(newHistory); err != nil {
				logger.Warn("failed to append final history to planning session", "error", err)
			}
		}
	}

	logger.Debug("final response generated")
	return resp.Texts[0], nil
}
