package legacy

import (
	"context"
	"fmt"
	"strings"

	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/strategy/planexec"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	ssnutil "github.com/secmon-lab/warren/pkg/utils/session"

	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

// StrategyFactory creates a gollem execution strategy for a chat session.
// It returns the strategy and an optional PlanProgressReporter for tracking plan progress.
type StrategyFactory func(ctx context.Context, params *StrategyParams) (gollem.Strategy, PlanProgressReporter)

// StrategyParams holds the parameters needed to create a strategy.
type StrategyParams struct {
	LLMClient  gollem.LLMClient
	Session    *session.Session
	Repository interfaces.Repository
	RequestID  string
	PlanFunc   func(context.Context, string)
}

// PlanProgressReporter reports whether planning occurred during execution.
type PlanProgressReporter interface {
	Planned() bool
}

// DefaultStrategyFactory creates the standard Plan & Execute strategy with progress tracking.
func DefaultStrategyFactory() StrategyFactory {
	return func(ctx context.Context, params *StrategyParams) (gollem.Strategy, PlanProgressReporter) {
		hooks := &chatPlanHooks{
			ctx:        ctx,
			planFunc:   params.PlanFunc,
			requestID:  params.RequestID,
			session:    params.Session,
			repository: params.Repository,
		}

		traceMW := newTraceLLMMiddleware()

		strategy := planexec.New(params.LLMClient,
			planexec.WithHooks(hooks),
			planexec.WithMaxIterations(30),
			planexec.WithMiddleware(traceMW),
		)

		return strategy, hooks
	}
}

// newTraceLLMMiddleware creates a content block middleware that traces LLM text responses.
func newTraceLLMMiddleware() gollem.ContentBlockMiddleware {
	return gollem.ContentBlockMiddleware(func(next gollem.ContentBlockHandler) gollem.ContentBlockHandler {
		return func(ctx context.Context, req *gollem.ContentRequest) (*gollem.ContentResponse, error) {
			resp, err := next(ctx, req)
			if err == nil && resp != nil && len(resp.Texts) > 0 {
				for _, text := range resp.Texts {
					msg.Trace(ctx, "💭 %s", text)
				}
			}
			return resp, err
		}
	})
}

// chatPlanHooks implements planexec.PlanExecuteHooks for chat progress tracking
type chatPlanHooks struct {
	ctx        context.Context
	planned    bool
	planFunc   func(context.Context, string)
	requestID  string
	session    *session.Session
	repository interfaces.Repository
}

var _ planexec.PlanExecuteHooks = &chatPlanHooks{}

// Planned returns whether any planning occurred during execution.
func (h *chatPlanHooks) Planned() bool {
	return h.planned
}

func (h *chatPlanHooks) OnPlanCreated(ctx context.Context, plan *planexec.Plan) error {
	if err := ssnutil.CheckStatus(ctx); err != nil {
		return err
	}

	h.planned = len(plan.Tasks) > 0

	if plan.Goal != "" && h.session != nil && h.repository != nil {
		h.session.UpdateIntent(ctx, plan.Goal)
		if err := h.repository.PutSession(ctx, h.session); err != nil {
			logging.From(ctx).Error("failed to update session intent", "error", err)
		}
	}

	return postPlanProgress(h.ctx, h.planFunc, h.requestID, plan)
}

func (h *chatPlanHooks) OnPlanUpdated(ctx context.Context, plan *planexec.Plan) error {
	if err := ssnutil.CheckStatus(ctx); err != nil {
		return err
	}

	h.planned = len(plan.Tasks) > 0
	return postPlanProgress(h.ctx, h.planFunc, h.requestID, plan)
}

func (h *chatPlanHooks) OnTaskDone(ctx context.Context, plan *planexec.Plan, _ *planexec.Task) error {
	if err := ssnutil.CheckStatus(ctx); err != nil {
		return err
	}

	h.planned = len(plan.Tasks) > 0
	if len(plan.Tasks) == 0 {
		return nil
	}
	return postPlanProgress(h.ctx, h.planFunc, h.requestID, plan)
}

// postPlanProgress posts the plan progress as a new message (not an update)
func postPlanProgress(ctx context.Context, planFunc func(context.Context, string), requestID string, plan *planexec.Plan) error {
	if len(plan.Tasks) == 0 {
		return nil
	}

	completedCount := 0
	inProgressCount := 0
	for _, task := range plan.Tasks {
		switch task.State {
		case planexec.TaskStateCompleted:
			completedCount++
		case planexec.TaskStateInProgress:
			inProgressCount++
		}
	}

	var messageBuilder strings.Builder

	var statusMessage string
	switch {
	case completedCount == len(plan.Tasks):
		statusMessage = "✅ Completed"
	case inProgressCount > 0:
		statusMessage = "⟳ Working..."
	default:
		statusMessage = "🚀 Thinking..."
	}
	fmt.Fprintf(&messageBuilder, "%s (Request ID: %s)\n", statusMessage, requestID)

	fmt.Fprintf(&messageBuilder, "🎯 Objective *%s*\n", plan.Goal)
	messageBuilder.WriteString("\n")
	fmt.Fprintf(&messageBuilder, "*Progress: %d/%d tasks completed*\n", completedCount, len(plan.Tasks))

	for _, task := range plan.Tasks {
		var icon string
		var status string

		switch task.State {
		case planexec.TaskStatePending:
			icon = "☑️"
			status = task.Description
		case planexec.TaskStateInProgress:
			icon = "⟳"
			status = task.Description
		case planexec.TaskStateCompleted:
			icon = "✅"
			status = fmt.Sprintf("~%s~", task.Description)
		default:
			icon = "?"
			status = task.Description
		}

		fmt.Fprintf(&messageBuilder, "%s %s\n", icon, status)
	}

	planFunc(ctx, messageBuilder.String())
	return nil
}
