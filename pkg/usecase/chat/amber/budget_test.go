package amber_test

import (
	"context"
	"testing"
	"time"

	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/usecase/chat/amber"
)

// mockBudgetStrategy is a configurable budget strategy for testing.
type mockBudgetStrategy struct {
	initialBudget   float64
	beforeCost      float64
	afterCost       float64
	hardLimitMargin int
}

func (s *mockBudgetStrategy) InitialBudget() float64                         { return s.initialBudget }
func (s *mockBudgetStrategy) BeforeToolCall(_ amber.ToolCallContext) float64 { return s.beforeCost }
func (s *mockBudgetStrategy) AfterToolCall(_ amber.ToolCallContext) float64  { return s.afterCost }
func (s *mockBudgetStrategy) ShouldExit(state amber.BudgetState) bool {
	return state.CallsAfterSoft > s.hardLimitMargin
}

func TestDefaultBudgetStrategy_InitialBudget(t *testing.T) {
	s := amber.NewDefaultBudgetStrategy()
	gt.Equal(t, s.InitialBudget(), 100.0)
}

func TestDefaultBudgetStrategy_BeforeToolCall(t *testing.T) {
	s := amber.NewDefaultBudgetStrategy()

	t.Run("sub-agent tool costs 0", func(t *testing.T) {
		cost := s.BeforeToolCall(amber.ToolCallContext{
			ToolName:  "query_bigquery",
			CallCount: 1,
		})
		gt.Equal(t, cost, 0.0)
	})

	t.Run("bigquery_query costs 15", func(t *testing.T) {
		cost := s.BeforeToolCall(amber.ToolCallContext{
			ToolName:  "bigquery_query",
			CallCount: 1,
		})
		gt.Equal(t, cost, 15.0)
	})

	t.Run("bigquery_list_datasets costs 3", func(t *testing.T) {
		cost := s.BeforeToolCall(amber.ToolCallContext{
			ToolName:  "bigquery_list_datasets",
			CallCount: 1,
		})
		gt.Equal(t, cost, 3.0)
	})

	t.Run("bigquery_get_table_schema costs 3", func(t *testing.T) {
		cost := s.BeforeToolCall(amber.ToolCallContext{
			ToolName:  "bigquery_get_table_schema",
			CallCount: 1,
		})
		gt.Equal(t, cost, 3.0)
	})

	t.Run("default tool costs 6.25", func(t *testing.T) {
		cost := s.BeforeToolCall(amber.ToolCallContext{
			ToolName:  "vt_ip_lookup",
			CallCount: 1,
		})
		gt.Equal(t, cost, 6.25)
	})

	t.Run("time cost delta is included", func(t *testing.T) {
		// At 210 seconds elapsed, time cost cumulative = 50.0
		cost := s.BeforeToolCall(amber.ToolCallContext{
			ToolName:    "vt_ip_lookup",
			Elapsed:     210 * time.Second,
			PrevElapsed: 0,
			CallCount:   1,
		})
		// Tool cost (6.25) + time cost delta (50.0)
		gt.Equal(t, cost, 56.25)
	})

	t.Run("time cost delta from prev elapsed", func(t *testing.T) {
		cost := s.BeforeToolCall(amber.ToolCallContext{
			ToolName:    "vt_ip_lookup",
			Elapsed:     210 * time.Second,
			PrevElapsed: 105 * time.Second,
			CallCount:   1,
		})
		// Tool cost (6.25) + time cost delta (50.0 - 25.0 = 25.0)
		gt.Equal(t, cost, 31.25)
	})
}

func TestDefaultBudgetStrategy_AfterToolCall(t *testing.T) {
	s := amber.NewDefaultBudgetStrategy()
	cost := s.AfterToolCall(amber.ToolCallContext{
		ToolName: "bigquery_query",
		Result:   map[string]any{"rows": 100},
	})
	gt.Equal(t, cost, 0.0)
}

func TestDefaultBudgetStrategy_ShouldExit(t *testing.T) {
	s := amber.NewDefaultBudgetStrategy()

	// 3 calls after soft: should not exit yet
	gt.B(t, s.ShouldExit(amber.BudgetState{CallsAfterSoft: 3})).False()
	// 4 calls after soft: should exit
	gt.B(t, s.ShouldExit(amber.BudgetState{CallsAfterSoft: 4})).True()
}

func TestBudgetTracker_BasicConsumption(t *testing.T) {
	strategy := &mockBudgetStrategy{
		initialBudget:   100.0,
		beforeCost:      10.0,
		afterCost:       0.0,
		hardLimitMargin: 3,
	}
	tracker := amber.NewBudgetTracker(strategy)

	// First call: 100 - 10 = 90 remaining
	status := tracker.TestBeforeToolCall("tool1", 0)
	gt.Equal(t, status, amber.BudgetOK)
	gt.Equal(t, tracker.Remaining(), 90.0)

	// 5th call: 100 - 50 = 50 remaining
	for i := 0; i < 4; i++ {
		status = tracker.TestBeforeToolCall("tool1", 0)
	}
	gt.Equal(t, status, amber.BudgetOK)
	gt.Equal(t, tracker.Remaining(), 50.0)
}

func TestBudgetTracker_AfterToolCallConsumption(t *testing.T) {
	strategy := &mockBudgetStrategy{
		initialBudget:   100.0,
		beforeCost:      10.0,
		afterCost:       5.0,
		hardLimitMargin: 3,
	}
	tracker := amber.NewBudgetTracker(strategy)

	// Before: 100 - 10 = 90
	tracker.TestBeforeToolCall("tool1", 0)
	gt.Equal(t, tracker.Remaining(), 90.0)

	// After: 90 - 5 = 85
	tracker.TestAfterToolCall("tool1", 0, map[string]any{"key": "value"}, nil)
	gt.Equal(t, tracker.Remaining(), 85.0)
}

func TestBudgetTracker_SoftLimit(t *testing.T) {
	strategy := &mockBudgetStrategy{
		initialBudget:   20.0,
		beforeCost:      10.0,
		afterCost:       0.0,
		hardLimitMargin: 2,
	}
	tracker := amber.NewBudgetTracker(strategy)

	// Call 1: 20 - 10 = 10 → OK
	status := tracker.TestBeforeToolCall("tool1", 0)
	gt.Equal(t, status, amber.BudgetOK)

	// Call 2: 10 - 10 = 0 → SoftLimit
	status = tracker.TestBeforeToolCall("tool1", 0)
	gt.Equal(t, status, amber.BudgetSoftLimit)
}

func TestBudgetTracker_HardLimit(t *testing.T) {
	strategy := &mockBudgetStrategy{
		initialBudget:   20.0,
		beforeCost:      10.0,
		afterCost:       0.0,
		hardLimitMargin: 2,
	}
	tracker := amber.NewBudgetTracker(strategy)

	// Call 1: OK
	tracker.TestBeforeToolCall("tool1", 0)
	// Call 2: SoftLimit (budget = 0)
	status := tracker.TestBeforeToolCall("tool1", 0)
	gt.Equal(t, status, amber.BudgetSoftLimit)

	// Call 3: SoftLimit (1st after soft, margin=2)
	status = tracker.TestBeforeToolCall("tool1", 0)
	gt.Equal(t, status, amber.BudgetSoftLimit)

	// Call 4: SoftLimit (2nd after soft, margin=2)
	status = tracker.TestBeforeToolCall("tool1", 0)
	gt.Equal(t, status, amber.BudgetSoftLimit)

	// Call 5: HardLimit (3rd after soft, > margin=2)
	status = tracker.TestBeforeToolCall("tool1", 0)
	gt.Equal(t, status, amber.BudgetHardLimit)
}

func TestBudgetTracker_GenerateHandoverInfo(t *testing.T) {
	strategy := &mockBudgetStrategy{
		initialBudget:   100.0,
		beforeCost:      50.0,
		afterCost:       0.0,
		hardLimitMargin: 1,
	}
	tracker := amber.NewBudgetTracker(strategy)

	tracker.TestBeforeToolCall("vt_ip_lookup", 10*time.Second)
	tracker.TestBeforeToolCall("bigquery_query", 30*time.Second)

	info := tracker.GenerateHandoverInfo()
	gt.S(t, info).Contains("Budget Exceeded")
	gt.S(t, info).Contains("vt_ip_lookup")
	gt.S(t, info).Contains("bigquery_query")
	gt.S(t, info).Contains("Tool calls")
}

func TestBudgetTracker_CustomAfterToolCallStrategy(t *testing.T) {
	strategy := &mockBudgetStrategy{
		initialBudget:   100.0,
		beforeCost:      5.0,
		afterCost:       20.0,
		hardLimitMargin: 2,
	}
	tracker := amber.NewBudgetTracker(strategy)

	// Before: 100 - 5 = 95
	status := tracker.TestBeforeToolCall("tool1", 0)
	gt.Equal(t, status, amber.BudgetOK)
	gt.Equal(t, tracker.Remaining(), 95.0)

	// After: 95 - 20 = 75
	tracker.TestAfterToolCall("tool1", 0, nil, nil)
	gt.Equal(t, tracker.Remaining(), 75.0)
}

func TestBudgetToolMiddleware_OK(t *testing.T) {
	strategy := &mockBudgetStrategy{
		initialBudget:   100.0,
		beforeCost:      10.0,
		afterCost:       0.0,
		hardLimitMargin: 3,
	}
	tracker := amber.NewBudgetTracker(strategy)
	mw := amber.NewBudgetToolMiddleware(tracker)

	handler := mw(func(_ context.Context, _ *gollem.ToolExecRequest) (*gollem.ToolExecResponse, error) {
		return &gollem.ToolExecResponse{
			Result: map[string]any{"data": "test"},
		}, nil
	})

	resp, err := handler(context.Background(), &gollem.ToolExecRequest{
		Tool: &gollem.FunctionCall{Name: "test_tool"},
	})

	gt.NoError(t, err)
	gt.V(t, resp.Result["_budget_info"]).NotNil()
}

func TestBudgetToolMiddleware_SoftLimit(t *testing.T) {
	strategy := &mockBudgetStrategy{
		initialBudget:   10.0,
		beforeCost:      10.0,
		afterCost:       0.0,
		hardLimitMargin: 2,
	}
	tracker := amber.NewBudgetTracker(strategy)
	mw := amber.NewBudgetToolMiddleware(tracker)

	handler := mw(func(_ context.Context, _ *gollem.ToolExecRequest) (*gollem.ToolExecResponse, error) {
		return &gollem.ToolExecResponse{
			Result: map[string]any{"data": "test"},
		}, nil
	})

	resp, err := handler(context.Background(), &gollem.ToolExecRequest{
		Tool: &gollem.FunctionCall{Name: "test_tool"},
	})

	gt.NoError(t, err)
	gt.V(t, resp.Result["_budget_warning"]).NotNil()
}

func TestBudgetToolMiddleware_SharedTrackerAcrossParentAndSubAgent(t *testing.T) {
	strategy := &mockBudgetStrategy{
		initialBudget:   100.0,
		beforeCost:      25.0,
		afterCost:       0.0,
		hardLimitMargin: 1,
	}
	tracker := amber.NewBudgetTracker(strategy)

	// Create two middleware instances from the same tracker,
	// simulating the pattern in exec.go where:
	// - parentMW is added to the parent agent (line 171)
	// - subAgentMW is injected into sub-agents' child agents (line 131)
	parentMW := amber.NewBudgetToolMiddleware(tracker)
	subAgentMW := amber.NewBudgetToolMiddleware(tracker)

	passthrough := func(_ context.Context, _ *gollem.ToolExecRequest) (*gollem.ToolExecResponse, error) {
		return &gollem.ToolExecResponse{
			Result: map[string]any{"data": "ok"},
		}, nil
	}

	parentHandler := parentMW(passthrough)
	subAgentHandler := subAgentMW(passthrough)

	ctx := context.Background()

	// Parent tool call: 100 - 25 = 75
	resp, err := parentHandler(ctx, &gollem.ToolExecRequest{
		Tool: &gollem.FunctionCall{Name: "parent_tool"},
	})
	gt.NoError(t, err)
	gt.V(t, resp.Result["_budget_info"]).NotNil()
	gt.Equal(t, tracker.Remaining(), 75.0)

	// Sub-agent internal tool call: 75 - 25 = 50 (shared budget)
	_, err = subAgentHandler(ctx, &gollem.ToolExecRequest{
		Tool: &gollem.FunctionCall{Name: "bigquery_query"},
	})
	gt.NoError(t, err)
	gt.Equal(t, tracker.Remaining(), 50.0)

	// Another parent tool call: 50 - 25 = 25
	_, err = parentHandler(ctx, &gollem.ToolExecRequest{
		Tool: &gollem.FunctionCall{Name: "parent_tool"},
	})
	gt.NoError(t, err)
	gt.Equal(t, tracker.Remaining(), 25.0)

	// Sub-agent tool call: 25 - 25 = 0 → SoftLimit
	resp, err = subAgentHandler(ctx, &gollem.ToolExecRequest{
		Tool: &gollem.FunctionCall{Name: "bigquery_query"},
	})
	gt.NoError(t, err)
	gt.V(t, resp.Result["_budget_warning"]).NotNil() // soft limit warning
	gt.Equal(t, tracker.Remaining(), 0.0)

	// One more sub-agent call after soft: still allowed (margin=1)
	_, err = subAgentHandler(ctx, &gollem.ToolExecRequest{
		Tool: &gollem.FunctionCall{Name: "bigquery_query"},
	})
	gt.NoError(t, err)

	// Next call triggers hard limit regardless of caller (parent or sub-agent)
	subAgentCalled := false
	hardLimitHandler := subAgentMW(func(_ context.Context, _ *gollem.ToolExecRequest) (*gollem.ToolExecResponse, error) {
		subAgentCalled = true
		return &gollem.ToolExecResponse{Result: map[string]any{}}, nil
	})
	resp, err = hardLimitHandler(ctx, &gollem.ToolExecRequest{
		Tool: &gollem.FunctionCall{Name: "bigquery_query"},
	})
	gt.NoError(t, err)
	gt.B(t, subAgentCalled).False() // tool should be blocked
	gt.V(t, resp.Result["error"]).NotNil()
}

func TestContextAwareBudgetMiddleware_ReadsTrackerFromContext(t *testing.T) {
	strategy := &mockBudgetStrategy{
		initialBudget:   100.0,
		beforeCost:      10.0,
		afterCost:       0.0,
		hardLimitMargin: 3,
	}
	tracker := amber.NewBudgetTracker(strategy)
	mw := amber.NewContextAwareBudgetMiddleware()

	handler := mw(func(_ context.Context, _ *gollem.ToolExecRequest) (*gollem.ToolExecResponse, error) {
		return &gollem.ToolExecResponse{
			Result: map[string]any{"data": "test"},
		}, nil
	})

	ctx := amber.WithBudgetTracker(context.Background(), tracker)
	resp, err := handler(ctx, &gollem.ToolExecRequest{
		Tool: &gollem.FunctionCall{Name: "test_tool"},
	})

	gt.NoError(t, err)
	gt.V(t, resp.Result["_budget_info"]).NotNil()
	gt.Equal(t, tracker.Remaining(), 90.0)
}

func TestContextAwareBudgetMiddleware_NoTrackerInContext(t *testing.T) {
	mw := amber.NewContextAwareBudgetMiddleware()

	toolCalled := false
	handler := mw(func(_ context.Context, _ *gollem.ToolExecRequest) (*gollem.ToolExecResponse, error) {
		toolCalled = true
		return &gollem.ToolExecResponse{
			Result: map[string]any{"data": "test"},
		}, nil
	})

	// No tracker in context — should pass through
	resp, err := handler(context.Background(), &gollem.ToolExecRequest{
		Tool: &gollem.FunctionCall{Name: "test_tool"},
	})

	gt.NoError(t, err)
	gt.B(t, toolCalled).True()
	// No budget info should be added
	gt.V(t, resp.Result["_budget_info"]).Nil()
	gt.V(t, resp.Result["_budget_warning"]).Nil()
}

func TestContextAwareBudgetMiddleware_DifferentTrackersPerTask(t *testing.T) {
	strategy := &mockBudgetStrategy{
		initialBudget:   20.0,
		beforeCost:      10.0,
		afterCost:       0.0,
		hardLimitMargin: 0, // hard limit immediately after soft
	}

	mw := amber.NewContextAwareBudgetMiddleware()

	passthrough := func(_ context.Context, _ *gollem.ToolExecRequest) (*gollem.ToolExecResponse, error) {
		return &gollem.ToolExecResponse{
			Result: map[string]any{"data": "ok"},
		}, nil
	}
	handler := mw(passthrough)

	// === Task 1: exhaust the budget ===
	tracker1 := amber.NewBudgetTracker(strategy)
	ctx1 := amber.WithBudgetTracker(context.Background(), tracker1)

	// Call 1: 20 - 10 = 10 (OK)
	resp, err := handler(ctx1, &gollem.ToolExecRequest{
		Tool: &gollem.FunctionCall{Name: "tool1"},
	})
	gt.NoError(t, err)
	gt.V(t, resp.Result["_budget_info"]).NotNil()

	// Call 2: 10 - 10 = 0 (SoftLimit)
	resp, err = handler(ctx1, &gollem.ToolExecRequest{
		Tool: &gollem.FunctionCall{Name: "tool1"},
	})
	gt.NoError(t, err)
	gt.V(t, resp.Result["_budget_warning"]).NotNil()

	// Call 3: HardLimit (margin=0, 1st call after soft)
	toolCalled := false
	hardHandler := mw(func(_ context.Context, _ *gollem.ToolExecRequest) (*gollem.ToolExecResponse, error) {
		toolCalled = true
		return &gollem.ToolExecResponse{Result: map[string]any{}}, nil
	})
	resp, err = hardHandler(ctx1, &gollem.ToolExecRequest{
		Tool: &gollem.FunctionCall{Name: "tool1"},
	})
	gt.NoError(t, err)
	gt.B(t, toolCalled).False() // blocked by hard limit
	gt.V(t, resp.Result["error"]).NotNil()

	// === Task 2: new tracker, same middleware — should work fine ===
	tracker2 := amber.NewBudgetTracker(strategy)
	ctx2 := amber.WithBudgetTracker(context.Background(), tracker2)

	// This call must succeed with the fresh tracker, NOT be blocked by tracker1
	toolCalled = false
	freshHandler := mw(func(_ context.Context, _ *gollem.ToolExecRequest) (*gollem.ToolExecResponse, error) {
		toolCalled = true
		return &gollem.ToolExecResponse{
			Result: map[string]any{"data": "fresh"},
		}, nil
	})
	resp, err = freshHandler(ctx2, &gollem.ToolExecRequest{
		Tool: &gollem.FunctionCall{Name: "tool1"},
	})
	gt.NoError(t, err)
	gt.B(t, toolCalled).True() // tool should execute successfully
	gt.V(t, resp.Result["_budget_info"]).NotNil()
	gt.Equal(t, tracker2.Remaining(), 10.0) // 20 - 10 = 10
}

func TestBudgetToolMiddleware_HardLimit(t *testing.T) {
	strategy := &mockBudgetStrategy{
		initialBudget:   10.0,
		beforeCost:      10.0,
		afterCost:       0.0,
		hardLimitMargin: 0,
	}
	tracker := amber.NewBudgetTracker(strategy)
	mw := amber.NewBudgetToolMiddleware(tracker)

	toolCalled := false
	handler := mw(func(_ context.Context, _ *gollem.ToolExecRequest) (*gollem.ToolExecResponse, error) {
		toolCalled = true
		return &gollem.ToolExecResponse{
			Result: map[string]any{"data": "test"},
		}, nil
	})

	// First call: soft limit (budget hits 0)
	_, _ = handler(context.Background(), &gollem.ToolExecRequest{
		Tool: &gollem.FunctionCall{Name: "test_tool"},
	})

	toolCalled = false
	// Second call: hard limit (exceeds margin=0)
	resp, err := handler(context.Background(), &gollem.ToolExecRequest{
		Tool: &gollem.FunctionCall{Name: "test_tool"},
	})

	gt.NoError(t, err)
	gt.B(t, toolCalled).False() // Tool should NOT have been called
	gt.V(t, resp.Result["error"]).NotNil()
}
