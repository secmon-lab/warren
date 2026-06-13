package aster_test

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/gollem-dev/gollem"
	gollemtrace "github.com/gollem-dev/gollem/trace"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/mock"
	chatModel "github.com/secmon-lab/warren/pkg/domain/model/chat"
	"github.com/secmon-lab/warren/pkg/repository"
	svcknowledge "github.com/secmon-lab/warren/pkg/service/knowledge"
	"github.com/secmon-lab/warren/pkg/usecase/chat"
	"github.com/secmon-lab/warren/pkg/usecase/chat/aster"
)

func TestAsterChat_PlanWithKnowledgeService(t *testing.T) {
	ctx := setupTestContext(t)
	repo := repository.NewMemory()
	testTicket := setupTicketAndAlert(t, ctx, repo)

	embeddingMock := &mock.EmbeddingClientMock{
		EmbeddingsFunc: func(ctx context.Context, texts []string, dimensionality int) ([][]float32, error) {
			result := make([][]float32, len(texts))
			for i := range texts {
				result[i] = make([]float32, dimensionality)
			}
			return result, nil
		},
	}
	knowledgeSvc := svcknowledge.New(repo, embeddingMock)

	// The mock LLM must handle multiple NewSession calls:
	// 1. The planner agent creates its own session internally (for plan)
	// 2. The replan/final phases also create sessions
	// The planner agent may call knowledge_search, then return a plan JSON.
	var mu sync.Mutex
	sessionCount := 0

	mockLLM := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			mu.Lock()
			sessionCount++
			sc := sessionCount
			mu.Unlock()

			ssn := newMockSession()
			ssn.GenerateFunc = func(ctx context.Context, input []gollem.Input, opts ...gollem.GenerateOption) (*gollem.Response, error) {
				// The agent's first session (plan) should return a direct plan
				// (without function calls, since no knowledge entries exist).
				if sc == 1 {
					// This is the planSession created by executeAster (not used in agent mode)
					return &gollem.Response{
						Texts: []string{`{"message": "Direct response.", "tasks": []}`},
					}, nil
				}
				// Agent-created session
				return &gollem.Response{
					Texts: []string{`{"message": "Analyzed with knowledge.", "tasks": []}`},
				}, nil
			}
			return ssn, nil
		},
	}

	chatUC := aster.New(repo, mockLLM,
		aster.WithKnowledgeService(knowledgeSvc),
	)
	ssn := newDummySession(testTicket.ID)

	err := chatUC.Execute(ctx, &chat.RunContext{Session: ssn, Message: "Analyze this alert", ChatCtx: &chatModel.ChatContext{Ticket: testTicket}})
	gt.NoError(t, err)

	// Verify that multiple sessions were created (planSession + agent's internal session)
	mu.Lock()
	gt.N(t, sessionCount).Greater(1)
	mu.Unlock()
}

func TestAsterChat_PlanWithoutKnowledgeService(t *testing.T) {
	// This tests the fallback path (no knowledge service) still works.
	// Existing TestAsterChat_DirectResponse already covers this, but this
	// explicitly verifies only one session is created.
	ctx := setupTestContext(t)
	repo := repository.NewMemory()
	testTicket := setupTicketAndAlert(t, ctx, repo)

	var mu sync.Mutex
	sessionCount := 0

	mockLLM := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			mu.Lock()
			sessionCount++
			mu.Unlock()

			ssn := newMockSession()
			ssn.GenerateFunc = func(ctx context.Context, input []gollem.Input, opts ...gollem.GenerateOption) (*gollem.Response, error) {
				return &gollem.Response{
					Texts: []string{`{"message": "Direct response.", "tasks": []}`},
				}, nil
			}
			return ssn, nil
		},
	}

	chatUC := aster.New(repo, mockLLM)
	ssn := newDummySession(testTicket.ID)

	err := chatUC.Execute(ctx, &chat.RunContext{Session: ssn, Message: "Analyze this alert", ChatCtx: &chatModel.ChatContext{Ticket: testTicket}})
	gt.NoError(t, err)

	// Without knowledge service, only one session should be created (planSession).
	mu.Lock()
	gt.V(t, sessionCount).Equal(1)
	mu.Unlock()
}

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

// traceAwareSession wraps a generate function and records LLM call spans via trace handler.
// This simulates what real LLM clients (vertex_client, etc.) do internally.
type traceAwareSession struct {
	generateFunc func(ctx context.Context, input []gollem.Input, opts ...gollem.GenerateOption) (*gollem.Response, error)
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
				Response:     &gollemtrace.LLMResponse{Texts: resp.Texts},
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
	return &gollem.History{
		Version:  gollem.HistoryVersion,
		Messages: []gollem.Message{{Role: gollem.RoleUser}},
	}, nil
}

func (s *traceAwareSession) AppendHistory(_ *gollem.History) error { return nil }

func (s *traceAwareSession) CountToken(_ context.Context, _ ...gollem.Input) (int, error) {
	return 0, nil
}

func TestAsterChat_PlannerTraceNotOverwritten(t *testing.T) {
	// Verify that executePlannerAgent creates child spans (via AsChildAgent)
	// rather than overwriting the root trace. When tasks are executed, the task
	// spans should persist across replan calls.
	ctx := setupTestContext(t)
	repo := repository.NewMemory()
	testTicket := setupTicketAndAlert(t, ctx, repo)

	embeddingMock := &mock.EmbeddingClientMock{
		EmbeddingsFunc: func(ctx context.Context, texts []string, dimensionality int) ([][]float32, error) {
			result := make([][]float32, len(texts))
			for i := range texts {
				result[i] = make([]float32, dimensionality)
			}
			return result, nil
		},
	}
	knowledgeSvc := svcknowledge.New(repo, embeddingMock)

	var mu sync.Mutex
	sessionCount := 0

	mockLLM := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			mu.Lock()
			sessionCount++
			sc := sessionCount
			mu.Unlock()

			return &traceAwareSession{
				generateFunc: func(ctx context.Context, input []gollem.Input, opts ...gollem.GenerateOption) (*gollem.Response, error) {
					switch {
					case sc <= 2:
						return &gollem.Response{
							Texts: []string{`{"message": "Analyzing.", "tasks": [{"id": "t1", "title": "Check IP", "description": "Look up IP", "tools": [], "sub_agents": []}]}`},
						}, nil
					case sc == 3:
						return &gollem.Response{
							Texts: []string{"IP is benign."},
						}, nil
					default:
						return &gollem.Response{
							Texts: []string{`{"tasks": []}`},
						}, nil
					}
				},
			}, nil
		},
	}

	traceRepo := &mockTraceRepo{}
	chatUC := aster.New(repo, mockLLM,
		aster.WithKnowledgeService(knowledgeSvc),
		aster.WithTraceRepository(traceRepo),
	)
	ssn := newDummySession(testTicket.ID)

	err := chatUC.Execute(ctx, &chat.RunContext{
		Session: ssn,
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

	// Verify LLM call spans are recorded (plan + task + replan + final = at least 3).
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

func TestFetchKnowledgeTags_NilService(t *testing.T) {
	ctx := setupTestContext(t)
	tags := aster.FetchKnowledgeTags(ctx, nil)
	gt.V(t, tags).Nil()
}

func TestFetchKnowledgeTags_WithService(t *testing.T) {
	ctx := setupTestContext(t)
	repo := repository.NewMemory()
	embeddingMock := &mock.EmbeddingClientMock{
		EmbeddingsFunc: func(ctx context.Context, texts []string, dimensionality int) ([][]float32, error) {
			return make([][]float32, len(texts)), nil
		},
	}
	knowledgeSvc := svcknowledge.New(repo, embeddingMock)

	// Create a tag
	_, err := knowledgeSvc.CreateTag(ctx, "test-tag", "A test tag")
	gt.NoError(t, err)

	tags := aster.FetchKnowledgeTags(ctx, knowledgeSvc)
	gt.A(t, tags).Length(1)
	gt.V(t, tags[0].Name).Equal("test-tag")
}
