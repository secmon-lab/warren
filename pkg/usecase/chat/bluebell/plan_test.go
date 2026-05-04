package bluebell_test

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/m-mizutani/gollem"
	gollemtrace "github.com/m-mizutani/gollem/trace"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/cli/config/llm"
	"github.com/secmon-lab/warren/pkg/domain/mock"
	chatModel "github.com/secmon-lab/warren/pkg/domain/model/chat"
	"github.com/secmon-lab/warren/pkg/repository"
	svcknowledge "github.com/secmon-lab/warren/pkg/service/knowledge"
	"github.com/secmon-lab/warren/pkg/usecase/chat"
	"github.com/secmon-lab/warren/pkg/usecase/chat/bluebell"
)

// mockTraceRepo captures saved traces for assertion.
type mockTraceRepo struct {
	mu     sync.Mutex
	traces []*gollemtrace.Trace
}

func (r *mockTraceRepo) Save(_ context.Context, t *gollemtrace.Trace) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.traces = append(r.traces, t)
	return nil
}

func (r *mockTraceRepo) last() *gollemtrace.Trace {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.traces) == 0 {
		return nil
	}
	return r.traces[len(r.traces)-1]
}

// traceAwareSession wraps a mock session and records LLM call spans via trace handler.
// This simulates what real LLM clients (vertex_client, etc.) do internally.
type traceAwareSession struct {
	generateFunc   func(ctx context.Context, input []gollem.Input, opts ...gollem.GenerateOption) (*gollem.Response, error)
	historyFunc    func() (*gollem.History, error)
	appendHistFunc func(h *gollem.History) error
}

func (s *traceAwareSession) Generate(ctx context.Context, input []gollem.Input, opts ...gollem.GenerateOption) (*gollem.Response, error) {
	h := gollemtrace.HandlerFrom(ctx)
	if h != nil {
		ctx = h.StartLLMCall(ctx)
	}
	resp, err := s.generateFunc(ctx, input, opts...)
	if h != nil {
		var data *gollemtrace.LLMCallData
		if resp != nil {
			data = &gollemtrace.LLMCallData{
				InputTokens:  100,
				OutputTokens: 50,
				Request:      &gollemtrace.LLMRequest{},
				Response: &gollemtrace.LLMResponse{
					Texts: resp.Texts,
				},
			}
		}
		h.EndLLMCall(ctx, data, err)
	}
	return resp, err
}

func (s *traceAwareSession) Stream(_ context.Context, _ []gollem.Input, _ ...gollem.GenerateOption) (<-chan *gollem.Response, error) {
	return nil, nil
}

func (s *traceAwareSession) GenerateContent(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
	return s.Generate(ctx, input)
}

func (s *traceAwareSession) GenerateStream(_ context.Context, _ ...gollem.Input) (<-chan *gollem.Response, error) {
	return nil, nil
}

func (s *traceAwareSession) History() (*gollem.History, error) {
	if s.historyFunc != nil {
		return s.historyFunc()
	}
	return &gollem.History{
		Version:  gollem.HistoryVersion,
		Messages: []gollem.Message{{Role: gollem.RoleUser}},
	}, nil
}

func (s *traceAwareSession) AppendHistory(h *gollem.History) error {
	if s.appendHistFunc != nil {
		return s.appendHistFunc(h)
	}
	return nil
}

func (s *traceAwareSession) CountToken(_ context.Context, _ ...gollem.Input) (int, error) {
	return 0, nil
}

func TestBluebellChat_PlannerTraceNotOverwritten(t *testing.T) {
	// Verify that executePlannerAgent creates child spans (via AsChildAgent)
	// rather than overwriting the root trace. When tasks are executed, the task
	// spans should persist across replan calls.
	ctx := setupTestContext(t)
	repo := repository.NewMemory()
	testTicket := setupTicketAndAlert(t, ctx, repo)
	knowledgeSvc := svcknowledge.New(repo, newMockEmbeddingClient())

	var mu sync.Mutex
	sessionCount := 0

	planJSON := `{"message": "Analyzing.", "tasks": [{"id": "t1", "title": "Check IP", "description": "Look up IP", "tools": [], "llm_id": "test"}]}`

	mockLLM := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			mu.Lock()
			sessionCount++
			sc := sessionCount
			mu.Unlock()

			ssn := &traceAwareSession{
				generateFunc: func(ctx context.Context, input []gollem.Input, opts ...gollem.GenerateOption) (*gollem.Response, error) {
					switch {
					// Session 1: intent resolver → return selector JSON
					case sc == 1:
						return &gollem.Response{
							Texts: []string{`{"prompt_id":"default","intent":"Investigate the alert."}`},
						}, nil
					// Session 2: planSession (created by executeBluebell, not used in agent mode)
					// Session 3: planner agent's internal session → return plan with a task
					case sc <= 3:
						return &gollem.Response{
							Texts: []string{planJSON},
						}, nil
					// Session 4: task agent's session → return task result
					case sc == 4:
						return &gollem.Response{
							Texts: []string{"IP is benign."},
						}, nil
					// Session 5+: replan agent / final response → no more tasks
					default:
						return &gollem.Response{
							Texts: []string{`{"tasks": []}`},
						}, nil
					}
				},
			}
			return ssn, nil
		},
	}

	traceRepo := &mockTraceRepo{}
	chatUC, err := bluebell.New(repo, llm.SingleClientRegistryForTest(mockLLM),
		bluebell.WithKnowledgeService(knowledgeSvc),
		bluebell.WithTraceRepository(traceRepo),
	)
	gt.NoError(t, err)

	err = chatUC.Execute(ctx, &chat.RunContext{
		Session: newDummySession(testTicket.ID),
		Message: "Analyze this alert",
		ChatCtx: &chatModel.ChatContext{Ticket: testTicket},
	})
	gt.NoError(t, err)

	// Verify trace was saved
	trace := traceRepo.last()
	gt.V(t, trace).NotNil()
	gt.V(t, trace.RootSpan).NotNil()

	// The root span should be "agent_execute" and should have children.
	// Before the fix, executePlannerAgent would overwrite the root trace,
	// leaving only the last planner call's llm_call as the sole child.
	gt.V(t, string(trace.RootSpan.Kind)).Equal("agent_execute")
	gt.N(t, len(trace.RootSpan.Children)).Greater(1)

	// Check that a "planner" agent_execute span exists in the tree
	// (created via AsChildAgent, not overwriting the root).
	gt.V(t, findSpan(trace.RootSpan, "planner", gollemtrace.SpanKindAgentExecute)).NotNil()

	// Check that task agent spans also exist (not wiped by replan's StartAgentExecute).
	allAgentSpans := collectSpans(trace.RootSpan, gollemtrace.SpanKindAgentExecute)
	agentNames := make(map[string]bool)
	for _, s := range allAgentSpans {
		agentNames[s.Name] = true
	}
	gt.V(t, agentNames["Check IP"]).Equal(true)

	// Verify "replan-phase-1" child span also exists alongside "planning" and task spans.
	gt.V(t, findSpan(trace.RootSpan, "replan-phase-1", gollemtrace.SpanKindAgentExecute)).NotNil()

	// Verify LLM call spans are recorded (plan + task + replan + final = at least 4).
	llmCalls := collectSpans(trace.RootSpan, gollemtrace.SpanKindLLMCall)
	gt.N(t, len(llmCalls)).GreaterOrEqual(3)

	// Verify LLM call data has token counts and response content recorded.
	hasContentRecorded := false
	for _, lc := range llmCalls {
		if lc.LLMCall == nil {
			continue
		}
		gt.N(t, lc.LLMCall.InputTokens).Greater(0)
		if lc.LLMCall.Response != nil && len(lc.LLMCall.Response.Texts) > 0 {
			hasContentRecorded = true
		}
	}
	gt.V(t, hasContentRecorded).Equal(true)

	// Verify that the planner span's LLM call contains the expected plan response.
	plannerSpan := findSpan(trace.RootSpan, "planner", gollemtrace.SpanKindAgentExecute)
	gt.V(t, plannerSpan).NotNil()
	plannerLLMCalls := collectSpans(plannerSpan, gollemtrace.SpanKindLLMCall)
	gt.N(t, len(plannerLLMCalls)).GreaterOrEqual(1)
	plannerLLM := plannerLLMCalls[0]
	gt.V(t, plannerLLM.LLMCall).NotNil()
	gt.V(t, plannerLLM.LLMCall.Response).NotNil()
	gt.N(t, len(plannerLLM.LLMCall.Response.Texts)).GreaterOrEqual(1)
	gt.V(t, strings.Contains(plannerLLM.LLMCall.Response.Texts[0], "Check IP")).Equal(true)
}

// findSpan recursively searches the span tree for a span matching name and kind.
func findSpan(root *gollemtrace.Span, name string, kind gollemtrace.SpanKind) *gollemtrace.Span {
	if root == nil {
		return nil
	}
	if root.Name == name && root.Kind == kind {
		return root
	}
	for _, child := range root.Children {
		if found := findSpan(child, name, kind); found != nil {
			return found
		}
	}
	return nil
}

// collectSpans recursively collects all spans matching the given kind.
func collectSpans(root *gollemtrace.Span, kind gollemtrace.SpanKind) []*gollemtrace.Span {
	if root == nil {
		return nil
	}
	var result []*gollemtrace.Span
	if root.Kind == kind {
		result = append(result, root)
	}
	for _, child := range root.Children {
		result = append(result, collectSpans(child, kind)...)
	}
	return result
}
