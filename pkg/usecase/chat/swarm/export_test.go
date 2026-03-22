package swarm

import (
	"context"
	"time"

	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/model/agent"
	slackModel "github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	hitlSvc "github.com/secmon-lab/warren/pkg/service/hitl"
)

// NewBudgetTracker exposes newBudgetTracker for testing.
func NewBudgetTracker(strategy BudgetStrategy) *BudgetTracker {
	return newBudgetTracker(strategy)
}

// BudgetTrackerBeforeToolCall exposes BeforeToolCall for testing.
func (t *BudgetTracker) TestBeforeToolCall(toolName string, elapsed time.Duration) BudgetStatus {
	return t.BeforeToolCall(toolName, elapsed)
}

// BudgetTrackerAfterToolCall exposes AfterToolCall for testing.
func (t *BudgetTracker) TestAfterToolCall(toolName string, elapsed time.Duration, result map[string]any, err error) {
	t.AfterToolCall(toolName, elapsed, result, err)
}

// TestStatus exposes status for testing.
func (t *BudgetTracker) TestStatus() BudgetStatus {
	return t.status()
}

// NewBudgetToolMiddleware exposes newBudgetToolMiddleware for testing.
func NewBudgetToolMiddleware(tracker *BudgetTracker) gollem.ToolMiddleware {
	return newBudgetToolMiddleware(tracker)
}

// NewContextAwareBudgetMiddleware exposes newContextAwareBudgetMiddleware for testing.
func NewContextAwareBudgetMiddleware() gollem.ToolMiddleware {
	return newContextAwareBudgetMiddleware()
}

// WithBudgetTracker exposes withBudgetTracker for testing.
func WithBudgetTracker(ctx context.Context, tracker *BudgetTracker) context.Context {
	return withBudgetTracker(ctx, tracker)
}

// FilterToolSets exposes filterToolSets for testing.
func FilterToolSets(ctx context.Context, allTools []gollem.ToolSet, allowedNames []string) []gollem.ToolSet {
	return filterToolSets(ctx, allTools, allowedNames)
}

// FilterSubAgents exposes filterSubAgents for testing.
func FilterSubAgents(allAgents []*agent.SubAgent, allowedNames []string) []*agent.SubAgent {
	return filterSubAgents(allAgents, allowedNames)
}

// StartSessionMonitor exposes startSessionMonitor for testing.
func (c *SwarmChat) StartSessionMonitor(ctx context.Context, sessionID types.SessionID) (context.Context, func()) {
	return c.startSessionMonitor(ctx, sessionID)
}

// HITLConfig exposes hitlConfig for testing.
type HITLConfig = hitlConfig

// NewHITLMiddleware exposes newHITLMiddleware for testing.
func NewHITLMiddleware(cfg HITLConfig) gollem.ToolMiddleware {
	return newHITLMiddleware(cfg)
}

// NewHITLConfig creates an hitlConfig for testing.
func NewHITLConfig(requireApproval map[string]bool, service *hitlSvc.Service, presenter hitlSvc.Presenter, userID string, sessionID types.SessionID, slackThread *slackModel.Thread) HITLConfig {
	return hitlConfig{
		requireApproval: requireApproval,
		service:         service,
		presenter:       presenter,
		userID:          userID,
		sessionID:       sessionID,
		slackThread:     slackThread,
	}
}
