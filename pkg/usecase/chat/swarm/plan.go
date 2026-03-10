package swarm

import (
	"context"
	"fmt"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/trace"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

// plan executes the planning phase and returns a structured plan.
func (c *SwarmChat) plan(ctx context.Context, session gollem.Session, pc *planningContext) (*PlanResult, error) {
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

	// Generate plan via LLM
	resp, err := session.GenerateContent(ctx, gollem.Text(planPrompt))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate plan")
	}

	result, err := parsePlanResult(resp.Texts)
	if err != nil {
		return nil, err
	}

	logger.Info("plan generated",
		"task_count", len(result.Tasks),
		"has_message", result.Message != "")

	return result, nil
}

// replan evaluates completed results and determines next steps.
func (c *SwarmChat) replan(ctx context.Context, session gollem.Session, pc *planningContext, allResults []*phaseResult, currentPhase int, systemPrompt string) (*ReplanResult, error) {
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

	// Switch session to replan schema for this call
	replanSession, err := c.llmClient.NewSession(ctx,
		gollem.WithSessionContentType(gollem.ContentTypeJSON),
		gollem.WithSessionResponseSchema(replanSchema),
		gollem.WithSessionSystemPrompt(systemPrompt),
	)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create replan session")
	}

	// Copy history from planning session
	history, err := session.History()
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get history from planning session")
	}
	if history != nil {
		if err := replanSession.AppendHistory(history); err != nil {
			logger.Warn("failed to append history to replan session", "error", err)
		}
	}

	resp, err := replanSession.GenerateContent(ctx, gollem.Text(replanPrompt))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate replan")
	}

	result, err := parseReplanResult(resp.Texts)
	if err != nil {
		return nil, err
	}

	logger.Info("replan completed",
		"phase", currentPhase,
		"new_task_count", len(result.Tasks))

	return result, nil
}

// generateFinalResponse generates the final response after all tasks are done.
func (c *SwarmChat) generateFinalResponse(ctx context.Context, session gollem.Session, pc *planningContext, allResults []*phaseResult, systemPrompt string) (string, error) {
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

	resp, err := finalSession.GenerateContent(ctx, gollem.Text(finalPrompt))
	if err != nil {
		return "", goerr.Wrap(err, "failed to generate final response")
	}

	if len(resp.Texts) == 0 {
		return "", goerr.New("no response from final LLM")
	}

	logger.Debug("final response generated")
	return resp.Texts[0], nil
}
