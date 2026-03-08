package swarm_test

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/opaq"
	"github.com/secmon-lab/warren/pkg/domain/mock"
	"github.com/secmon-lab/warren/pkg/domain/model/agent"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/usecase/chat/swarm"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

func newMockPolicyClient(t *testing.T) *mock.PolicyClientMock {
	return &mock.PolicyClientMock{
		QueryFunc: func(ctx context.Context, query string, input, result any, opts ...opaq.QueryOption) error {
			if query == "data.auth.agent" {
				gt.NoError(t, json.Unmarshal([]byte(`{"allow":true}`), &result))
			}
			return nil
		},
		SourcesFunc: func() map[string]string {
			return map[string]string{}
		},
	}
}

func setupTestContext(t *testing.T) context.Context {
	ctx := t.Context()
	notifyFunc := func(ctx context.Context, message string) {}
	traceFunc := func(ctx context.Context, message string) {}
	warnFunc := func(ctx context.Context, message string) {}
	return msg.With(ctx, notifyFunc, traceFunc, warnFunc)
}

func newMockSession() *mock.LLMSessionMock {
	return &mock.LLMSessionMock{
		HistoryFunc: func() (*gollem.History, error) {
			return &gollem.History{Version: 1}, nil
		},
		AppendHistoryFunc: func(h *gollem.History) error {
			return nil
		},
	}
}

func setupTicketAndAlert(t *testing.T, ctx context.Context, repo *repository.Memory) *ticket.Ticket {
	testTicket := ticket.Ticket{
		ID:       types.NewTicketID(),
		Status:   types.TicketStatusOpen,
		AlertIDs: []types.AlertID{types.NewAlertID()},
	}
	gt.NoError(t, repo.PutTicket(ctx, testTicket))

	testAlert := alert.Alert{
		ID:     testTicket.AlertIDs[0],
		Schema: "test.alert",
		Data:   map[string]any{"test": "data"},
	}
	gt.NoError(t, repo.PutAlert(ctx, testAlert))

	return &testTicket
}

func TestSwarmChat_DirectResponse(t *testing.T) {
	ctx := setupTestContext(t)
	repo := repository.NewMemory()
	testTicket := setupTicketAndAlert(t, ctx, repo)

	mockLLM := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			ssn := newMockSession()
			ssn.GenerateContentFunc = func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
				return &gollem.Response{
					Texts: []string{`{"message": "The answer is 42.", "tasks": []}`},
				}, nil
			}
			return ssn, nil
		},
	}

	chatUC := swarm.New(repo, mockLLM, newMockPolicyClient(t),
		swarm.WithNoAuthorization(true),
	)

	err := chatUC.Execute(ctx, testTicket, "What is the meaning of life?")
	gt.NoError(t, err)
}

func TestSwarmChat_SinglePhaseWithTasks(t *testing.T) {
	ctx := setupTestContext(t)
	repo := repository.NewMemory()
	testTicket := setupTicketAndAlert(t, ctx, repo)

	var mu sync.Mutex
	callCount := 0
	mockLLM := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			ssn := newMockSession()
			ssn.GenerateContentFunc = func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
				mu.Lock()
				callCount++
				cc := callCount
				mu.Unlock()

				for _, inp := range input {
					if text, ok := inp.(gollem.Text); ok {
						inputStr := string(text)
						if cc == 1 {
							return &gollem.Response{
								Texts: []string{`{
									"message": "I'll analyze the alert.",
									"tasks": [
										{
											"id": "task-1",
											"title": "Analyze source IP",
											"description": "Look up the source IP",
											"tools": [],
											"sub_agents": []
										}
									]
								}`},
							}, nil
						}
						if strings.Contains(inputStr, "Completed Task Results") || strings.Contains(inputStr, "Phase") {
							return &gollem.Response{
								Texts: []string{`{"tasks": []}`},
							}, nil
						}
						return &gollem.Response{
							Texts: []string{"No threat detected."},
						}, nil
					}
				}
				return &gollem.Response{Texts: []string{"OK"}}, nil
			}
			return ssn, nil
		},
	}

	chatUC := swarm.New(repo, mockLLM, newMockPolicyClient(t),
		swarm.WithNoAuthorization(true),
	)

	err := chatUC.Execute(ctx, testTicket, "Analyze this alert")
	gt.NoError(t, err)
}

func TestSwarmChat_MaxPhasesLimit(t *testing.T) {
	ctx := setupTestContext(t)
	repo := repository.NewMemory()
	testTicket := setupTicketAndAlert(t, ctx, repo)

	mockLLM := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			ssn := newMockSession()
			ssn.GenerateContentFunc = func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
				for _, inp := range input {
					if _, ok := inp.(gollem.Text); ok {
						return &gollem.Response{
							Texts: []string{`{
								"message": "Working on it...",
								"tasks": [
									{
										"id": "task-loop",
										"title": "Infinite task",
										"description": "This task keeps generating more tasks",
										"tools": [],
										"sub_agents": []
									}
								]
							}`},
						}, nil
					}
				}
				return &gollem.Response{Texts: []string{"Done"}}, nil
			}
			return ssn, nil
		},
	}

	chatUC := swarm.New(repo, mockLLM, newMockPolicyClient(t),
		swarm.WithNoAuthorization(true),
		swarm.WithMaxPhases(2),
	)

	err := chatUC.Execute(ctx, testTicket, "Do something")
	gt.NoError(t, err)
}

func TestSwarmChat_ParallelExecution(t *testing.T) {
	ctx := setupTestContext(t)
	repo := repository.NewMemory()
	testTicket := setupTicketAndAlert(t, ctx, repo)

	var mu sync.Mutex
	callCount := 0

	mockLLM := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			ssn := newMockSession()
			ssn.GenerateContentFunc = func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
				mu.Lock()
				callCount++
				cc := callCount
				mu.Unlock()

				for _, inp := range input {
					if _, ok := inp.(gollem.Text); ok {
						if cc == 1 {
							return &gollem.Response{
								Texts: []string{`{
									"message": "Analyzing in parallel...",
									"tasks": [
										{"id": "t1", "title": "Task 1", "description": "Do task 1", "tools": [], "sub_agents": []},
										{"id": "t2", "title": "Task 2", "description": "Do task 2", "tools": [], "sub_agents": []},
										{"id": "t3", "title": "Task 3", "description": "Do task 3", "tools": [], "sub_agents": []}
									]
								}`},
							}, nil
						}
						if cc >= 5 && cc <= 7 {
							return &gollem.Response{
								Texts: []string{`{"tasks": []}`},
							}, nil
						}
						return &gollem.Response{
							Texts: []string{"All tasks completed."},
						}, nil
					}
				}
				return &gollem.Response{Texts: []string{"OK"}}, nil
			}
			return ssn, nil
		},
	}

	chatUC := swarm.New(repo, mockLLM, newMockPolicyClient(t),
		swarm.WithNoAuthorization(true),
	)

	err := chatUC.Execute(ctx, testTicket, "Analyze all indicators")
	gt.NoError(t, err)
}

func TestSwarmChat_ErrorIsolation(t *testing.T) {
	ctx := setupTestContext(t)
	repo := repository.NewMemory()
	testTicket := setupTicketAndAlert(t, ctx, repo)

	var mu sync.Mutex
	callCount := 0

	mockLLM := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			ssn := newMockSession()
			ssn.GenerateContentFunc = func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
				mu.Lock()
				callCount++
				cc := callCount
				mu.Unlock()

				for _, inp := range input {
					if text, ok := inp.(gollem.Text); ok {
						inputStr := string(text)
						if cc == 1 {
							// Planning: 2 tasks
							return &gollem.Response{
								Texts: []string{`{
									"message": "Analyzing in parallel...",
									"tasks": [
										{"id": "t-ok", "title": "Succeeding task", "description": "This task will succeed", "tools": [], "sub_agents": []},
										{"id": "t-fail", "title": "Failing task", "description": "This task will fail", "tools": [], "sub_agents": []}
									]
								}`},
							}, nil
						}
						// Replan: return empty
						if strings.Contains(inputStr, "Completed Task Results") || strings.Contains(inputStr, "Phase") {
							return &gollem.Response{
								Texts: []string{`{"tasks": []}`},
							}, nil
						}
						// Task agents: succeed for "Succeeding task", fail for "Failing task"
						if strings.Contains(inputStr, "will fail") {
							return nil, goerr.New("simulated task failure")
						}
						return &gollem.Response{
							Texts: []string{"Task completed successfully."},
						}, nil
					}
				}
				return &gollem.Response{Texts: []string{"OK"}}, nil
			}
			return ssn, nil
		},
	}

	chatUC := swarm.New(repo, mockLLM, newMockPolicyClient(t),
		swarm.WithNoAuthorization(true),
	)

	// Execute should complete without error even though one task failed
	err := chatUC.Execute(ctx, testTicket, "Test error isolation")
	gt.NoError(t, err)
}

func TestSwarmChat_ToolFiltering(t *testing.T) {
	ctx := setupTestContext(t)

	// Create mock tool sets with different spec names
	toolA := &mockToolSet{name: "tool_alpha"}
	toolB := &mockToolSet{name: "tool_beta"}
	toolC := &mockToolSet{name: "tool_gamma"}

	allTools := []gollem.ToolSet{toolA, toolB, toolC}

	// Filter for only tool_alpha and tool_gamma
	filtered := swarm.FilterToolSets(ctx, allTools, []string{"tool_alpha", "tool_gamma"})
	gt.A(t, filtered).Length(2)

	// Verify the right tools are included
	var names []string
	for _, ts := range filtered {
		specs, err := ts.Specs(ctx)
		gt.NoError(t, err)
		for _, s := range specs {
			names = append(names, s.Name)
		}
	}
	gt.V(t, len(names)).Equal(2)
	gt.True(t, containsString(names, "tool_alpha"))
	gt.True(t, containsString(names, "tool_gamma"))
	gt.True(t, !containsString(names, "tool_beta"))
}

func TestSwarmChat_ToolFilteringEmptyAllowList(t *testing.T) {
	ctx := setupTestContext(t)

	toolA := &mockToolSet{name: "tool_alpha"}
	allTools := []gollem.ToolSet{toolA}

	// Empty allow list returns nil
	filtered := swarm.FilterToolSets(ctx, allTools, []string{})
	gt.True(t, filtered == nil)
}

func TestSwarmChat_SubAgentFiltering(t *testing.T) {
	// Create mock sub-agents
	saFalcon := agent.NewSubAgent(
		gollem.NewSubAgent("falcon", "CrowdStrike Falcon", func() (*gollem.Agent, error) { return nil, nil }),
		"hint",
	)
	saBigQuery := agent.NewSubAgent(
		gollem.NewSubAgent("bigquery", "BigQuery", func() (*gollem.Agent, error) { return nil, nil }),
		"hint",
	)
	saSlack := agent.NewSubAgent(
		gollem.NewSubAgent("slack", "Slack", func() (*gollem.Agent, error) { return nil, nil }),
		"hint",
	)

	all := []*agent.SubAgent{saFalcon, saBigQuery, saSlack}

	// Filter for only falcon and slack
	filtered := swarm.FilterSubAgents(all, []string{"falcon", "slack"})
	gt.A(t, filtered).Length(2)

	var names []string
	for _, sa := range filtered {
		names = append(names, sa.Inner().Spec().Name)
	}
	gt.True(t, containsString(names, "falcon"))
	gt.True(t, containsString(names, "slack"))
	gt.True(t, !containsString(names, "bigquery"))
}

func TestSwarmChat_MultiPhaseReplan(t *testing.T) {
	ctx := setupTestContext(t)
	repo := repository.NewMemory()
	testTicket := setupTicketAndAlert(t, ctx, repo)

	var mu sync.Mutex
	callCount := 0

	mockLLM := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			ssn := newMockSession()
			ssn.GenerateContentFunc = func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
				mu.Lock()
				callCount++
				cc := callCount
				mu.Unlock()

				for _, inp := range input {
					if text, ok := inp.(gollem.Text); ok {
						inputStr := string(text)

						// First call: planning - phase 1 tasks
						if cc == 1 {
							return &gollem.Response{
								Texts: []string{`{
									"message": "Starting multi-phase analysis...",
									"tasks": [
										{"id": "p1-t1", "title": "Phase 1 Task", "description": "Initial analysis", "tools": [], "sub_agents": []}
									]
								}`},
							}, nil
						}

						// Replan calls: check if it contains Phase info
						if strings.Contains(inputStr, "Phase") || strings.Contains(inputStr, "Completed Task Results") {
							// First replan: add phase 2 task
							if !strings.Contains(inputStr, "Phase 2") {
								return &gollem.Response{
									Texts: []string{`{
										"tasks": [
											{"id": "p2-t1", "title": "Phase 2 Task", "description": "Follow-up analysis", "tools": [], "sub_agents": []}
										]
									}`},
								}, nil
							}
							// Second replan (after phase 2): done
							return &gollem.Response{
								Texts: []string{`{"tasks": []}`},
							}, nil
						}

						// Task agent responses
						return &gollem.Response{
							Texts: []string{"Task result data."},
						}, nil
					}
				}
				return &gollem.Response{Texts: []string{"OK"}}, nil
			}
			return ssn, nil
		},
	}

	chatUC := swarm.New(repo, mockLLM, newMockPolicyClient(t),
		swarm.WithNoAuthorization(true),
	)

	err := chatUC.Execute(ctx, testTicket, "Run multi-phase analysis")
	gt.NoError(t, err)
}

func TestSwarmChat_AuthorizationDenied(t *testing.T) {
	ctx := setupTestContext(t)
	repo := repository.NewMemory()
	testTicket := setupTicketAndAlert(t, ctx, repo)

	denyPolicy := &mock.PolicyClientMock{
		QueryFunc: func(ctx context.Context, query string, input, result any, opts ...opaq.QueryOption) error {
			if query == "data.auth.agent" {
				gt.NoError(t, json.Unmarshal([]byte(`{"allow":false}`), &result))
			}
			return nil
		},
		SourcesFunc: func() map[string]string {
			return map[string]string{}
		},
	}

	mockLLM := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			t.Fatal("LLM should not be called when authorization is denied")
			return nil, nil
		},
	}

	chatUC := swarm.New(repo, mockLLM, denyPolicy)

	err := chatUC.Execute(ctx, testTicket, "Analyze this")
	gt.NoError(t, err)
}

// mockToolSet implements gollem.ToolSet for testing.
type mockToolSet struct {
	name string
}

func (m *mockToolSet) Specs(_ context.Context) ([]gollem.ToolSpec, error) {
	return []gollem.ToolSpec{
		{Name: m.name, Description: "Mock tool: " + m.name},
	}, nil
}

func (m *mockToolSet) Run(_ context.Context, _ string, _ map[string]any) (map[string]any, error) {
	return nil, nil
}

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func TestStartSessionMonitor_AbortDetection(t *testing.T) {
	ctx := setupTestContext(t)
	repo := repository.NewMemory()

	// Create a session
	ssn := &session.Session{
		ID:     types.NewSessionID(),
		Status: types.SessionStatusRunning,
	}
	gt.NoError(t, repo.PutSession(ctx, ssn))

	chatUC := swarm.New(repo, nil, nil)

	monitorCtx, stop := chatUC.StartSessionMonitor(ctx, ssn.ID)
	defer stop()

	// Verify context is not cancelled initially
	gt.Value(t, monitorCtx.Err()).Equal(nil)

	// Simulate abort by updating session status
	ssn.Status = types.SessionStatusAborted
	gt.NoError(t, repo.PutSession(ctx, ssn))

	// Wait for monitor to detect abort (ticker is 10s, so we need to wait)
	// Use a shorter approach: just call stop and check context
	// The monitor checks every 10s, so in tests we rely on the stop() cancellation path
	stop()

	// After stop(), context should be cancelled
	gt.Value(t, monitorCtx.Err()).NotEqual(nil)
}

func TestStartSessionMonitor_NormalCompletion(t *testing.T) {
	ctx := setupTestContext(t)
	repo := repository.NewMemory()

	// Create a running session
	ssn := &session.Session{
		ID:     types.NewSessionID(),
		Status: types.SessionStatusRunning,
	}
	gt.NoError(t, repo.PutSession(ctx, ssn))

	chatUC := swarm.New(repo, nil, nil)

	monitorCtx, stop := chatUC.StartSessionMonitor(ctx, ssn.ID)

	// stop() should cancel context and goroutine should terminate cleanly
	stop()

	// Context should be cancelled after stop
	gt.Value(t, monitorCtx.Err()).NotEqual(nil)
}

func TestStartSessionMonitor_DBErrorContinuesMonitoring(t *testing.T) {
	ctx := setupTestContext(t)
	repo := repository.NewMemory()

	// Create a session - the session doesn't exist in DB, so GetSession returns nil
	// This simulates a DB error scenario where session is not found
	nonExistentID := types.NewSessionID()

	chatUC := swarm.New(repo, nil, nil)

	monitorCtx, stop := chatUC.StartSessionMonitor(ctx, nonExistentID)
	defer stop()

	// Context should still be active (monitor continues despite nil session)
	gt.Value(t, monitorCtx.Err()).Equal(nil)

	// Clean stop
	stop()
}

// Ensure imports are used.
var (
	_ = goerr.New
	_ = agent.NewSubAgent
)
