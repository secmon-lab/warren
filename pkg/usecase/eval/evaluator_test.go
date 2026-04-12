package eval_test

import (
	"context"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	evalModel "github.com/secmon-lab/warren/pkg/domain/model/eval"
	"github.com/secmon-lab/warren/pkg/usecase/eval"
)

func newTestTrace(toolCalls []evalModel.ToolCallRecord, agentOutput string) *evalModel.Trace {
	return &evalModel.Trace{
		ScenarioName: "test",
		RunID:        "test-run",
		StartTime:    time.Now(),
		EndTime:      time.Now().Add(10 * time.Second),
		ToolCalls:    toolCalls,
		AgentOutput:  agentOutput,
		TotalTokens:  1000,
	}
}

func TestEvaluate_Outcome_Keywords(t *testing.T) {
	trace := newTestTrace(nil, "This is a Tor exit node with HIGH severity")
	expectations := &evalModel.Expectations{
		Outcome: &evalModel.OutcomeExpectation{
			FindingMustContain: []string{"Tor", "missing_word"},
			Severity:           "high",
		},
	}

	result, err := eval.Evaluate(context.Background(), trace, expectations, nil)
	gt.NoError(t, err)

	gt.V(t, result.Outcome.FindingKeywordsFound).Equal([]string{"Tor"})
	gt.V(t, result.Outcome.FindingKeywordsMissed).Equal([]string{"missing_word"})
	gt.V(t, result.Outcome.SeverityMatch).Equal(true)
}

func TestEvaluate_Trajectory_MustCall(t *testing.T) {
	toolCalls := []evalModel.ToolCallRecord{
		{Sequence: 1, ToolName: "virustotal"},
		{Sequence: 2, ToolName: "bigquery_query"},
		{Sequence: 3, ToolName: "otx"},
	}
	trace := newTestTrace(toolCalls, "output")

	expectations := &evalModel.Expectations{
		Trajectory: &evalModel.TrajectoryExpectation{
			MustCall:    []string{"virustotal", "bigquery_query"},
			MustNotCall: []string{"intune"},
		},
	}

	result, err := eval.Evaluate(context.Background(), trace, expectations, nil)
	gt.NoError(t, err)

	gt.V(t, result.Trajectory.MustCallResults["virustotal"]).Equal(true)
	gt.V(t, result.Trajectory.MustCallResults["bigquery_query"]).Equal(true)
	gt.V(t, result.Trajectory.MustNotCallResults["intune"]).Equal(false) // not violated
}

func TestEvaluate_Trajectory_MustNotCall_Violated(t *testing.T) {
	toolCalls := []evalModel.ToolCallRecord{
		{Sequence: 1, ToolName: "intune"},
	}
	trace := newTestTrace(toolCalls, "output")

	expectations := &evalModel.Expectations{
		Trajectory: &evalModel.TrajectoryExpectation{
			MustNotCall: []string{"intune"},
		},
	}

	result, err := eval.Evaluate(context.Background(), trace, expectations, nil)
	gt.NoError(t, err)

	gt.V(t, result.Trajectory.MustNotCallResults["intune"]).Equal(true) // violated
}

func TestEvaluate_Trajectory_OrderedCalls(t *testing.T) {
	toolCalls := []evalModel.ToolCallRecord{
		{Sequence: 1, ToolName: "virustotal"},
		{Sequence: 2, ToolName: "otx"},
		{Sequence: 3, ToolName: "bigquery_query"},
	}
	trace := newTestTrace(toolCalls, "output")

	expectations := &evalModel.Expectations{
		Trajectory: &evalModel.TrajectoryExpectation{
			OrderedCalls: []string{"virustotal", "bigquery_query"},
		},
	}

	result, err := eval.Evaluate(context.Background(), trace, expectations, nil)
	gt.NoError(t, err)
	gt.V(t, result.Trajectory.OrderedCallsPass).Equal(true)
}

func TestEvaluate_Trajectory_OrderedCalls_Fail(t *testing.T) {
	toolCalls := []evalModel.ToolCallRecord{
		{Sequence: 1, ToolName: "bigquery_query"},
		{Sequence: 2, ToolName: "virustotal"},
	}
	trace := newTestTrace(toolCalls, "output")

	expectations := &evalModel.Expectations{
		Trajectory: &evalModel.TrajectoryExpectation{
			OrderedCalls: []string{"virustotal", "bigquery_query"},
		},
	}

	result, err := eval.Evaluate(context.Background(), trace, expectations, nil)
	gt.NoError(t, err)
	gt.V(t, result.Trajectory.OrderedCallsPass).Equal(false)
}

func TestEvaluate_Efficiency(t *testing.T) {
	toolCalls := []evalModel.ToolCallRecord{
		{Sequence: 1, ToolName: "vt", Args: map[string]any{"ip": "1.2.3.4"}},
		{Sequence: 2, ToolName: "vt", Args: map[string]any{"ip": "1.2.3.4"}}, // duplicate
		{Sequence: 3, ToolName: "vt", Args: map[string]any{"ip": "5.6.7.8"}}, // different args
		{Sequence: 4, ToolName: "bigquery_query", Args: map[string]any{"q": "SELECT 1"}},
	}
	trace := newTestTrace(toolCalls, "output")

	expectations := &evalModel.Expectations{
		Efficiency: &evalModel.EfficiencyExpectation{
			MaxTotalCalls:     5,
			MaxDuplicateCalls: 2,
		},
	}

	result, err := eval.Evaluate(context.Background(), trace, expectations, nil)
	gt.NoError(t, err)

	gt.V(t, result.Efficiency.TotalCalls).Equal(4)
	gt.V(t, result.Efficiency.DuplicateCalls).Equal(1) // vt with same IP called twice
	gt.V(t, result.Efficiency.MaxCallsPass).Equal(true)
	gt.V(t, result.Efficiency.MaxDuplicatePass).Equal(true)
}

func TestEvaluate_Efficiency_Exceeded(t *testing.T) {
	toolCalls := make([]evalModel.ToolCallRecord, 25)
	for i := range toolCalls {
		toolCalls[i] = evalModel.ToolCallRecord{Sequence: i, ToolName: "vt", Args: map[string]any{"i": float64(i)}}
	}
	trace := newTestTrace(toolCalls, "output")

	expectations := &evalModel.Expectations{
		Efficiency: &evalModel.EfficiencyExpectation{
			MaxTotalCalls: 20,
		},
	}

	result, err := eval.Evaluate(context.Background(), trace, expectations, nil)
	gt.NoError(t, err)
	gt.V(t, result.Efficiency.MaxCallsPass).Equal(false)
}

func TestEvaluate_NilExpectations(t *testing.T) {
	trace := newTestTrace(nil, "output")
	result, err := eval.Evaluate(context.Background(), trace, nil, nil)
	gt.NoError(t, err)
	gt.V(t, result.Trace).NotNil()
	gt.V(t, result.Outcome).Nil()
	gt.V(t, result.Trajectory).Nil()
	gt.V(t, result.Efficiency).Nil()
}
