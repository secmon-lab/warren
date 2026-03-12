package swarm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/m-mizutani/gollem"
)

// BudgetStatus represents the current state of the action budget.
type BudgetStatus int

const (
	// BudgetOK indicates budget is still available.
	BudgetOK BudgetStatus = iota
	// BudgetSoftLimit indicates budget is exhausted but a few more calls are allowed.
	BudgetSoftLimit
	// BudgetHardLimit indicates the hard limit has been reached and tool execution must stop.
	BudgetHardLimit
)

// ToolCallContext holds all information about a tool call for cost calculation.
type ToolCallContext struct {
	ToolName    string
	Elapsed     time.Duration
	PrevElapsed time.Duration // Elapsed time at the previous tool call (for time cost delta)
	CallCount   int
	Result      map[string]any
	Error       error
}

// BudgetState holds a snapshot of the current budget state for strategy decisions.
type BudgetState struct {
	CallsAfterSoft int
	TotalCalls     int
	Remaining      float64
	Elapsed        time.Duration
}

// BudgetStrategy defines the budget consumption logic.
type BudgetStrategy interface {
	// InitialBudget returns the total budget for a task.
	InitialBudget() float64

	// BeforeToolCall returns the budget cost to consume before executing a tool.
	BeforeToolCall(ctx ToolCallContext) float64

	// AfterToolCall returns additional budget cost based on the tool execution result.
	AfterToolCall(ctx ToolCallContext) float64

	// ShouldExit determines whether tool execution must be forcibly stopped.
	// Called after the soft limit has been hit to decide if the task should be terminated.
	ShouldExit(state BudgetState) bool
}

// ToolRecord records a single tool execution for handover information.
type ToolRecord struct {
	Name      string
	Cost      float64
	Timestamp time.Time
}

// BudgetTracker tracks the budget state for a single task execution.
type BudgetTracker struct {
	strategy            BudgetStrategy
	remaining           float64
	initialBudget       float64
	startTime           time.Time
	toolCalls           int
	softLimitHit        bool
	callsAfterSoft      int
	lastTimeCostElapsed time.Duration
	toolHistory         []ToolRecord
}

// newBudgetTracker creates a new BudgetTracker with the given strategy.
func newBudgetTracker(strategy BudgetStrategy) *BudgetTracker {
	initial := strategy.InitialBudget()
	return &BudgetTracker{
		strategy:      strategy,
		remaining:     initial,
		initialBudget: initial,
		startTime:     time.Now(),
	}
}

// BeforeToolCall consumes budget before tool execution and returns the current status.
func (t *BudgetTracker) BeforeToolCall(toolName string, elapsed time.Duration) BudgetStatus {
	if t.softLimitHit {
		t.callsAfterSoft++
	}
	t.toolCalls++

	cost := t.strategy.BeforeToolCall(ToolCallContext{
		ToolName:    toolName,
		Elapsed:     elapsed,
		PrevElapsed: t.lastTimeCostElapsed,
		CallCount:   t.toolCalls,
	})

	t.remaining -= cost
	t.lastTimeCostElapsed = elapsed

	t.toolHistory = append(t.toolHistory, ToolRecord{
		Name:      toolName,
		Cost:      cost,
		Timestamp: t.startTime.Add(elapsed),
	})

	if !t.softLimitHit && t.remaining <= 0 {
		t.softLimitHit = true
	}

	return t.status()
}

// AfterToolCall consumes additional budget after tool execution based on results.
func (t *BudgetTracker) AfterToolCall(toolName string, elapsed time.Duration, result map[string]any, err error) {
	additionalCost := t.strategy.AfterToolCall(ToolCallContext{
		ToolName:    toolName,
		Elapsed:     elapsed,
		PrevElapsed: t.lastTimeCostElapsed,
		CallCount:   t.toolCalls,
		Result:      result,
		Error:       err,
	})

	if additionalCost > 0 {
		t.remaining -= additionalCost
		if len(t.toolHistory) > 0 {
			t.toolHistory[len(t.toolHistory)-1].Cost += additionalCost
		}
	}

	if !t.softLimitHit && t.remaining <= 0 {
		t.softLimitHit = true
	}
}

// status returns the current budget status without side effects.
func (t *BudgetTracker) status() BudgetStatus {
	if t.softLimitHit {
		if t.strategy.ShouldExit(BudgetState{
			CallsAfterSoft: t.callsAfterSoft,
			TotalCalls:     t.toolCalls,
			Remaining:      t.remaining,
			Elapsed:        time.Since(t.startTime),
		}) {
			return BudgetHardLimit
		}
		return BudgetSoftLimit
	}

	if t.remaining <= 0 {
		return BudgetSoftLimit
	}

	return BudgetOK
}

// Remaining returns the remaining budget.
func (t *BudgetTracker) Remaining() float64 {
	return t.remaining
}

// GenerateHandoverInfo generates handover information from the tool execution history.
func (t *BudgetTracker) GenerateHandoverInfo() string {
	var b strings.Builder
	fmt.Fprintf(&b, "## Budget Exceeded — Task Handover\n\n")
	fmt.Fprintf(&b, "This task was terminated because the action budget was exhausted.\n\n")
	var consumedPct float64
	if t.initialBudget > 0 {
		consumedPct = (1 - t.remaining/t.initialBudget) * 100
	}
	fmt.Fprintf(&b, "**Budget**: %.1f/%.1f (%.0f%% consumed)\n", t.remaining, t.initialBudget, consumedPct)
	fmt.Fprintf(&b, "**Tool calls**: %d\n", t.toolCalls)
	if len(t.toolHistory) > 0 {
		elapsed := t.toolHistory[len(t.toolHistory)-1].Timestamp.Sub(t.startTime)
		fmt.Fprintf(&b, "**Elapsed**: %s\n\n", elapsed.Truncate(time.Second))
	}

	fmt.Fprintf(&b, "### Tools Executed\n\n")
	for _, rec := range t.toolHistory {
		fmt.Fprintf(&b, "- `%s` (cost: %.1f)\n", rec.Name, rec.Cost)
	}

	return b.String()
}

// budgetTrackerCtxKeyType is a context key type for budget tracker.
type budgetTrackerCtxKeyType struct{}

// withBudgetTracker stores a BudgetTracker in the context.
func withBudgetTracker(ctx context.Context, tracker *BudgetTracker) context.Context {
	return context.WithValue(ctx, budgetTrackerCtxKeyType{}, tracker)
}

// budgetTrackerFrom retrieves a BudgetTracker from the context.
// Returns nil if no tracker is present.
func budgetTrackerFrom(ctx context.Context) *BudgetTracker {
	v, _ := ctx.Value(budgetTrackerCtxKeyType{}).(*BudgetTracker)
	return v
}

// newContextAwareBudgetMiddleware creates a gollem.ToolMiddleware that reads
// the BudgetTracker from the context on each invocation. This allows a single
// middleware instance to be shared across tasks while each task provides its
// own tracker via the context.
// If no tracker is found in the context, the middleware passes through.
func newContextAwareBudgetMiddleware() gollem.ToolMiddleware {
	return func(next gollem.ToolHandler) gollem.ToolHandler {
		return func(ctx context.Context, req *gollem.ToolExecRequest) (*gollem.ToolExecResponse, error) {
			tracker := budgetTrackerFrom(ctx)
			if tracker == nil {
				// No tracker in context — pass through (budget disabled)
				return next(ctx, req)
			}
			return executeBudgetedToolCall(ctx, req, tracker, next)
		}
	}
}

// subAgentToolNames contains tool names for sub-agent invocations.
// These have zero cost because their internal tool calls are tracked individually.
var subAgentToolNames = map[string]bool{
	"query_bigquery": true,
	"query_falcon":   true,
	"query_slack":    true,
}

// DefaultBudgetStrategy implements BudgetStrategy with sensible defaults.
// Target: ~16 tool calls or ~3.5 minutes per task.
type DefaultBudgetStrategy struct{}

// NewDefaultBudgetStrategy creates a new DefaultBudgetStrategy.
func NewDefaultBudgetStrategy() *DefaultBudgetStrategy {
	return &DefaultBudgetStrategy{}
}

// InitialBudget returns 100.0 as the total budget.
func (s *DefaultBudgetStrategy) InitialBudget() float64 {
	return 100.0
}

// BeforeToolCall calculates the cost before tool execution.
// Cost = tool fixed cost + time cost delta.
func (s *DefaultBudgetStrategy) BeforeToolCall(ctx ToolCallContext) float64 {
	var toolCost float64
	switch {
	case subAgentToolNames[ctx.ToolName]:
		toolCost = 0
	case ctx.ToolName == "bigquery_query":
		toolCost = 15.0
	case ctx.ToolName == "bigquery_list_datasets" || ctx.ToolName == "bigquery_get_table_schema":
		toolCost = 3.0
	default:
		toolCost = 6.25
	}

	// Time cost: cumulative (elapsed_seconds / 210) * 50.0, take delta
	timeCostNow := (ctx.Elapsed.Seconds() / 210.0) * 50.0
	timeCostPrev := (ctx.PrevElapsed.Seconds() / 210.0) * 50.0
	timeCostDelta := timeCostNow - timeCostPrev

	return toolCost + timeCostDelta
}

// AfterToolCall always returns 0 for the default strategy.
func (s *DefaultBudgetStrategy) AfterToolCall(_ ToolCallContext) float64 {
	return 0
}

// ShouldExit returns true when more than 3 tool calls have been made after the soft limit.
func (s *DefaultBudgetStrategy) ShouldExit(state BudgetState) bool {
	return state.CallsAfterSoft > 3
}

// executeBudgetedToolCall runs a tool call through budget tracking: consume budget
// before execution, block on hard limit, consume additional cost after execution,
// and append budget info to the response.
func executeBudgetedToolCall(ctx context.Context, req *gollem.ToolExecRequest, tracker *BudgetTracker, next gollem.ToolHandler) (*gollem.ToolExecResponse, error) {
	elapsed := time.Since(tracker.startTime)
	status := tracker.BeforeToolCall(req.Tool.Name, elapsed)

	// Hard limit: block tool execution
	if status == BudgetHardLimit {
		return &gollem.ToolExecResponse{
			Result: map[string]any{
				"error": "ACTION BUDGET HARD LIMIT REACHED. Tool execution blocked. Your response will be used as handover information for the next task.",
			},
		}, nil
	}

	// Execute the tool
	resp, err := next(ctx, req)

	// After tool call: consume additional cost based on results
	if resp != nil {
		tracker.AfterToolCall(req.Tool.Name, time.Since(tracker.startTime), resp.Result, resp.Error)
	}

	// Append budget info to response
	if resp != nil && err == nil {
		currentStatus := tracker.status()
		appendBudgetInfo(resp, tracker, currentStatus)
	}

	return resp, err
}

// newBudgetToolMiddleware creates a gollem.ToolMiddleware that enforces budget limits
// using a fixed tracker provided at creation time.
func newBudgetToolMiddleware(tracker *BudgetTracker) gollem.ToolMiddleware {
	return func(next gollem.ToolHandler) gollem.ToolHandler {
		return func(ctx context.Context, req *gollem.ToolExecRequest) (*gollem.ToolExecResponse, error) {
			return executeBudgetedToolCall(ctx, req, tracker, next)
		}
	}
}

// appendBudgetInfo appends budget status information to the tool response.
func appendBudgetInfo(resp *gollem.ToolExecResponse, tracker *BudgetTracker, status BudgetStatus) {
	if resp.Result == nil {
		resp.Result = make(map[string]any)
	}

	switch status {
	case BudgetOK:
		resp.Result["_budget_info"] = fmt.Sprintf(
			"[Action Budget: %.1f/%.1f remaining (%d tool calls used)]",
			tracker.remaining, tracker.initialBudget, tracker.toolCalls,
		)
	case BudgetSoftLimit:
		elapsed := time.Since(tracker.startTime).Truncate(time.Second)
		resp.Result["_budget_warning"] = fmt.Sprintf(
			"[⚠️ ACTION BUDGET EXHAUSTED (%.1f/%.1f). You have used %d tool calls in %s. Wrap up immediately: summarize your findings and end your response. Forced termination may occur at any time.]",
			tracker.remaining, tracker.initialBudget, tracker.toolCalls, elapsed,
		)
	}
}
