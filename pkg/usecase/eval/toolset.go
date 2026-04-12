package eval

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/eval"
	"github.com/secmon-lab/warren/pkg/usecase/eval/mock"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

// TraceRecorder collects tool call records during an evaluation run.
type TraceRecorder struct {
	mu       sync.Mutex
	records  []eval.ToolCallRecord
	sequence atomic.Int32
}

// NewTraceRecorder creates a new TraceRecorder.
func NewTraceRecorder() *TraceRecorder {
	return &TraceRecorder{}
}

// Record adds a tool call record.
func (r *TraceRecorder) Record(record eval.ToolCallRecord) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = append(r.records, record)
}

// Records returns all recorded tool calls.
func (r *TraceRecorder) Records() []eval.ToolCallRecord {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make([]eval.ToolCallRecord, len(r.records))
	copy(result, r.records)
	return result
}

// NextSequence returns the next sequence number.
func (r *TraceRecorder) NextSequence() int {
	return int(r.sequence.Add(1))
}

// EvalToolSet wraps a real ToolSet and intercepts Run() calls.
// Specs(), ID(), Description(), Prompt() are all delegated to the real tool.
type EvalToolSet struct {
	real      interfaces.ToolSet
	store     *ResponseStore
	mockAgent *mock.Agent
	recorder  *TraceRecorder
}

// Specs delegates to the real ToolSet so the LLM sees correct tool definitions.
func (e *EvalToolSet) Specs(ctx context.Context) ([]gollem.ToolSpec, error) {
	return e.real.Specs(ctx)
}

// Run intercepts the tool call and resolves the response from:
// 1. ResponseStore (pre-defined or previously generated)
// 2. MockAgent (dynamic LLM generation, saved to ResponseStore)
// The real tool's Run() is never called.
func (e *EvalToolSet) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	start := time.Now()
	seq := e.recorder.NextSequence()

	// Try ResponseStore first
	if rf := e.store.Lookup(name, args); rf != nil {
		record := eval.ToolCallRecord{
			Sequence: seq,
			ToolName: name,
			Args:     args,
			Response: rf.Response,
			Source:   eval.ResponseSourcePreDefined,
			Duration: time.Since(start),
		}
		e.recorder.Record(record)
		return rf.Response, nil
	}

	// Generate with MockAgent
	response, tokens, err := e.mockAgent.Generate(ctx, name, args)
	if err != nil {
		logging.From(ctx).Error("mock agent generation failed",
			"tool", name,
			"error", err,
		)
		// Return error as a tool response so the Warren agent can see it
		response = map[string]any{
			"error": err.Error(),
		}
	}

	// Save to ResponseStore for future runs
	if saveErr := e.store.Save(name, args, response); saveErr != nil {
		logging.From(ctx).Error("failed to save generated response",
			"tool", name,
			"error", saveErr,
		)
	}

	record := eval.ToolCallRecord{
		Sequence:   seq,
		ToolName:   name,
		Args:       args,
		Response:   response,
		Source:     eval.ResponseSourceGenerated,
		Duration:   time.Since(start),
		TokensUsed: tokens,
	}
	e.recorder.Record(record)

	return response, nil
}

// ID delegates to the real ToolSet.
func (e *EvalToolSet) ID() string { return e.real.ID() }

// Description delegates to the real ToolSet.
func (e *EvalToolSet) Description() string { return e.real.Description() }

// Prompt delegates to the real ToolSet.
func (e *EvalToolSet) Prompt(ctx context.Context) (string, error) { return e.real.Prompt(ctx) }

// WrapForEval wraps all ToolSets with EvalToolSet for evaluation.
// The returned ToolSets have the same Specs/ID/Description/Prompt as the originals,
// but Run() is intercepted to use ResponseStore + MockAgent.
func WrapForEval(tools []interfaces.ToolSet, store *ResponseStore, agent *mock.Agent, recorder *TraceRecorder) []interfaces.ToolSet {
	wrapped := make([]interfaces.ToolSet, len(tools))
	for i, ts := range tools {
		wrapped[i] = &EvalToolSet{
			real:      ts,
			store:     store,
			mockAgent: agent,
			recorder:  recorder,
		}
	}
	return wrapped
}
