package aster_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gt"
	adapter "github.com/secmon-lab/warren/pkg/adapter/storage"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/mock"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	chatModel "github.com/secmon-lab/warren/pkg/domain/model/chat"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	storage_svc "github.com/secmon-lab/warren/pkg/service/storage"
	"github.com/secmon-lab/warren/pkg/usecase/chat"
	"github.com/secmon-lab/warren/pkg/usecase/chat/aster"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

func newDummySession(ticketID types.TicketID) *session.Session {
	return &session.Session{
		ID:       types.NewSessionID(),
		TicketID: ticketID,
		Status:   types.SessionStatusRunning,
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
			return &gollem.History{
				Version:  gollem.HistoryVersion,
				Messages: []gollem.Message{{Role: gollem.RoleUser}},
			}, nil
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

func TestAsterChat_DirectResponse(t *testing.T) {
	ctx := setupTestContext(t)
	repo := repository.NewMemory()
	testTicket := setupTicketAndAlert(t, ctx, repo)

	mockLLM := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			ssn := newMockSession()
			ssn.GenerateFunc = func(ctx context.Context, input []gollem.Input, opts ...gollem.GenerateOption) (*gollem.Response, error) {
				return &gollem.Response{
					Texts: []string{`{"message": "The answer is 42.", "tasks": []}`},
				}, nil
			}
			return ssn, nil
		},
	}

	chatUC := aster.New(repo, mockLLM)
	ssn := newDummySession(testTicket.ID)

	err := chatUC.Execute(ctx, &chat.RunContext{Session: ssn, Message: "What is the meaning of life?", ChatCtx: &chatModel.ChatContext{Ticket: testTicket}})
	gt.NoError(t, err)
}

func TestAsterChat_SinglePhaseWithTasks(t *testing.T) {
	ctx := setupTestContext(t)
	repo := repository.NewMemory()
	testTicket := setupTicketAndAlert(t, ctx, repo)

	var mu sync.Mutex
	callCount := 0
	mockLLM := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			ssn := newMockSession()
			ssn.GenerateFunc = func(ctx context.Context, input []gollem.Input, opts ...gollem.GenerateOption) (*gollem.Response, error) {
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

	chatUC := aster.New(repo, mockLLM)
	ssn := newDummySession(testTicket.ID)

	err := chatUC.Execute(ctx, &chat.RunContext{Session: ssn, Message: "Analyze this alert", ChatCtx: &chatModel.ChatContext{Ticket: testTicket}})
	gt.NoError(t, err)
}

func TestAsterChat_MaxPhasesLimit(t *testing.T) {
	ctx := setupTestContext(t)
	repo := repository.NewMemory()
	testTicket := setupTicketAndAlert(t, ctx, repo)

	mockLLM := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			ssn := newMockSession()
			ssn.GenerateFunc = func(ctx context.Context, input []gollem.Input, opts ...gollem.GenerateOption) (*gollem.Response, error) {
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

	chatUC := aster.New(repo, mockLLM,
		aster.WithMaxPhases(2),
	)
	ssn := newDummySession(testTicket.ID)

	err := chatUC.Execute(ctx, &chat.RunContext{Session: ssn, Message: "Do something", ChatCtx: &chatModel.ChatContext{Ticket: testTicket}})
	gt.NoError(t, err)
}

func TestAsterChat_ParallelExecution(t *testing.T) {
	ctx := setupTestContext(t)
	repo := repository.NewMemory()
	testTicket := setupTicketAndAlert(t, ctx, repo)

	var mu sync.Mutex
	callCount := 0

	mockLLM := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			ssn := newMockSession()
			ssn.GenerateFunc = func(ctx context.Context, input []gollem.Input, opts ...gollem.GenerateOption) (*gollem.Response, error) {
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

	chatUC := aster.New(repo, mockLLM)
	ssn := newDummySession(testTicket.ID)

	err := chatUC.Execute(ctx, &chat.RunContext{Session: ssn, Message: "Analyze all indicators", ChatCtx: &chatModel.ChatContext{Ticket: testTicket}})
	gt.NoError(t, err)
}

func TestAsterChat_ErrorIsolation(t *testing.T) {
	ctx := setupTestContext(t)
	repo := repository.NewMemory()
	testTicket := setupTicketAndAlert(t, ctx, repo)

	var mu sync.Mutex
	callCount := 0

	mockLLM := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			ssn := newMockSession()
			ssn.GenerateFunc = func(ctx context.Context, input []gollem.Input, opts ...gollem.GenerateOption) (*gollem.Response, error) {
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

	chatUC := aster.New(repo, mockLLM)
	ssn := newDummySession(testTicket.ID)

	// Execute should complete without error even though one task failed
	err := chatUC.Execute(ctx, &chat.RunContext{Session: ssn, Message: "Test error isolation", ChatCtx: &chatModel.ChatContext{Ticket: testTicket}})
	gt.NoError(t, err)
}

func TestAsterChat_ToolFiltering(t *testing.T) {
	// Create mock tool sets with different IDs
	toolA := &mockToolSet{name: "tool_alpha"}
	toolB := &mockToolSet{name: "tool_beta"}
	toolC := &mockToolSet{name: "tool_gamma"}

	allTools := []interfaces.ToolSet{toolA, toolB, toolC}

	// Filter for only tool_alpha and tool_gamma
	filtered := aster.FilterToolSets(allTools, []string{"tool_alpha", "tool_gamma"})
	gt.A(t, filtered).Length(2)

	// Verify the right tools are included
	var ids []string
	for _, ts := range filtered {
		ids = append(ids, ts.ID())
	}
	gt.V(t, len(ids)).Equal(2)
	gt.True(t, containsString(ids, "tool_alpha"))
	gt.True(t, containsString(ids, "tool_gamma"))
	gt.True(t, !containsString(ids, "tool_beta"))
}

func TestAsterChat_ToolFilteringEmptyAllowList(t *testing.T) {
	toolA := &mockToolSet{name: "tool_alpha"}
	allTools := []interfaces.ToolSet{toolA}

	// Empty allow list returns nil
	filtered := aster.FilterToolSets(allTools, []string{})
	gt.True(t, filtered == nil)
}

func TestAsterChat_MultiPhaseReplan(t *testing.T) {
	ctx := setupTestContext(t)
	repo := repository.NewMemory()
	testTicket := setupTicketAndAlert(t, ctx, repo)

	var mu sync.Mutex
	callCount := 0

	mockLLM := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			ssn := newMockSession()
			ssn.GenerateFunc = func(ctx context.Context, input []gollem.Input, opts ...gollem.GenerateOption) (*gollem.Response, error) {
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

	chatUC := aster.New(repo, mockLLM)
	ssn := newDummySession(testTicket.ID)

	err := chatUC.Execute(ctx, &chat.RunContext{Session: ssn, Message: "Run multi-phase analysis", ChatCtx: &chatModel.ChatContext{Ticket: testTicket}})
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

func (m *mockToolSet) ID() string {
	return m.name
}

func (m *mockToolSet) Description() string {
	return "Mock tool: " + m.name
}

func (m *mockToolSet) Prompt(_ context.Context) (string, error) {
	return "", nil
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

	chatUC := aster.New(repo, nil,
		aster.WithMonitorPollInterval(10*time.Millisecond),
	)

	monitorCtx, stop := chatUC.StartSessionMonitor(ctx, ssn.ID)
	defer stop()

	// Verify context is not cancelled initially
	gt.Value(t, monitorCtx.Err()).Equal(nil)

	// Simulate abort by updating session status
	ssn.Status = types.SessionStatusAborted
	gt.NoError(t, repo.PutSession(ctx, ssn))

	// Wait for the monitor goroutine to detect the abort and cancel the context
	select {
	case <-monitorCtx.Done():
		// Success: monitor detected the abort
	case <-time.After(1 * time.Second):
		t.Fatal("context was not canceled by monitor")
	}

	gt.Value(t, monitorCtx.Err()).Equal(context.Canceled)
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

	chatUC := aster.New(repo, nil)

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

	chatUC := aster.New(repo, nil)

	monitorCtx, stop := chatUC.StartSessionMonitor(ctx, nonExistentID)
	defer stop()

	// Context should still be active (monitor continues despite nil session)
	gt.Value(t, monitorCtx.Err()).Equal(nil)

	// Clean stop
	stop()
}

func TestAsterChat_LatestHistorySavedOnDirectResponse(t *testing.T) {
	ctx := setupTestContext(t)
	repo := repository.NewMemory()
	testTicket := setupTicketAndAlert(t, ctx, repo)
	mockStorage := adapter.NewMock()

	mockLLM := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			ssn := newMockSession()
			ssn.GenerateFunc = func(ctx context.Context, input []gollem.Input, opts ...gollem.GenerateOption) (*gollem.Response, error) {
				return &gollem.Response{
					Texts: []string{`{"message": "Direct response.", "tasks": []}`},
				}, nil
			}
			return ssn, nil
		},
	}

	chatUC := aster.New(repo, mockLLM,
		aster.WithStorageClient(mockStorage),
	)
	ssn := newDummySession(testTicket.ID)

	gt.NoError(t, chatUC.Execute(ctx, &chat.RunContext{Session: ssn, Message: "Hello", ChatCtx: &chatModel.ChatContext{Ticket: testTicket}}))

	// Verify latest.json was saved
	storageSvc := storage_svc.New(mockStorage)
	latest, err := storageSvc.GetLatestHistory(ctx, testTicket.ID)
	gt.NoError(t, err)
	gt.V(t, latest).NotNil()
}

func TestAsterChat_LatestHistorySavedAfterReplan(t *testing.T) {
	ctx := setupTestContext(t)
	repo := repository.NewMemory()
	testTicket := setupTicketAndAlert(t, ctx, repo)
	mockStorage := adapter.NewMock()

	var mu sync.Mutex
	callCount := 0
	mockLLM := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			ssn := newMockSession()
			ssn.GenerateFunc = func(ctx context.Context, input []gollem.Input, opts ...gollem.GenerateOption) (*gollem.Response, error) {
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
									"message": "Working...",
									"tasks": [{"id": "t1", "title": "Task", "description": "Do it", "tools": [], "sub_agents": []}]
								}`},
							}, nil
						}
						if strings.Contains(inputStr, "Phase") || strings.Contains(inputStr, "Completed Task Results") {
							return &gollem.Response{
								Texts: []string{`{"tasks": []}`},
							}, nil
						}
						return &gollem.Response{Texts: []string{"Done."}}, nil
					}
				}
				return &gollem.Response{Texts: []string{"OK"}}, nil
			}
			return ssn, nil
		},
	}

	chatUC := aster.New(repo, mockLLM,
		aster.WithStorageClient(mockStorage),
	)
	ssn := newDummySession(testTicket.ID)

	gt.NoError(t, chatUC.Execute(ctx, &chat.RunContext{Session: ssn, Message: "Analyze", ChatCtx: &chatModel.ChatContext{Ticket: testTicket}}))

	// Verify latest.json was saved (saved after plan, replan, and final response)
	storageSvc := storage_svc.New(mockStorage)
	latest, err := storageSvc.GetLatestHistory(ctx, testTicket.ID)
	gt.NoError(t, err)
	gt.V(t, latest).NotNil()
}

func TestAsterChat_LatestHistorySavedOnAbort(t *testing.T) {
	ctx := setupTestContext(t)
	repo := repository.NewMemory()
	testTicket := setupTicketAndAlert(t, ctx, repo)
	mockStorage := adapter.NewMock()

	var mu sync.Mutex
	callCount := 0
	mockLLM := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			ssn := newMockSession()
			ssn.GenerateFunc = func(ctx context.Context, input []gollem.Input, opts ...gollem.GenerateOption) (*gollem.Response, error) {
				mu.Lock()
				callCount++
				cc := callCount
				mu.Unlock()

				// First call: plan with tasks
				if cc == 1 {
					return &gollem.Response{
						Texts: []string{`{
							"message": "Working...",
							"tasks": [{"id": "t1", "title": "Slow Task", "description": "Takes time", "tools": [], "sub_agents": []}]
						}`},
					}, nil
				}

				// Task agent: simulate slow execution, return error for cancelled ctx
				if cc == 2 {
					return &gollem.Response{Texts: []string{"Task done."}}, nil
				}

				// Replan: return error to simulate abort detection
				return nil, context.Canceled
			}
			return ssn, nil
		},
	}

	chatUC := aster.New(repo, mockLLM,
		aster.WithStorageClient(mockStorage),
		aster.WithMonitorPollInterval(10*time.Millisecond),
	)
	ssn := newDummySession(testTicket.ID)

	// Execute — plan succeeds and latest is saved,
	// but replan may fail or be aborted
	_ = chatUC.Execute(ctx, &chat.RunContext{Session: ssn, Message: "Slow analysis", ChatCtx: &chatModel.ChatContext{Ticket: testTicket}})

	// Verify latest.json was saved at least after the planning phase
	storageSvc := storage_svc.New(mockStorage)
	latest, err := storageSvc.GetLatestHistory(ctx, testTicket.ID)
	gt.NoError(t, err)
	gt.V(t, latest).NotNil()
}

func TestAsterChat_LatestHistorySavedOnReplanError(t *testing.T) {
	ctx := setupTestContext(t)
	repo := repository.NewMemory()
	testTicket := setupTicketAndAlert(t, ctx, repo)
	mockStorage := adapter.NewMock()

	var mu sync.Mutex
	callCount := 0
	mockLLM := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			ssn := newMockSession()
			ssn.GenerateFunc = func(ctx context.Context, input []gollem.Input, opts ...gollem.GenerateOption) (*gollem.Response, error) {
				mu.Lock()
				callCount++
				cc := callCount
				mu.Unlock()

				for _, inp := range input {
					if text, ok := inp.(gollem.Text); ok {
						inputStr := string(text)

						// Plan: return tasks
						if cc == 1 {
							return &gollem.Response{
								Texts: []string{`{
									"message": "Working...",
									"tasks": [{"id": "t1", "title": "Task", "description": "Do it", "tools": [], "sub_agents": []}]
								}`},
							}, nil
						}

						// Replan: return error
						if strings.Contains(inputStr, "Phase") || strings.Contains(inputStr, "Completed Task Results") {
							return nil, goerr.New("simulated replan error")
						}

						// Task agent
						return &gollem.Response{Texts: []string{"Done."}}, nil
					}
				}
				return &gollem.Response{Texts: []string{"OK"}}, nil
			}
			return ssn, nil
		},
	}

	chatUC := aster.New(repo, mockLLM,
		aster.WithStorageClient(mockStorage),
	)
	ssn := newDummySession(testTicket.ID)

	// Execute completes (replan error is logged, proceeds to final response)
	_ = chatUC.Execute(ctx, &chat.RunContext{Session: ssn, Message: "Test replan error", ChatCtx: &chatModel.ChatContext{Ticket: testTicket}})

	// Verify latest.json was saved at least after the planning phase
	storageSvc := storage_svc.New(mockStorage)
	latest, err := storageSvc.GetLatestHistory(ctx, testTicket.ID)
	gt.NoError(t, err)
	gt.V(t, latest).NotNil()
}

func TestAsterChat_BudgetMiddlewareNotAccumulated(t *testing.T) {
	// Regression test: verify that budget middleware is NOT accumulated across tasks.
	// Before the fix, each executeTask call would append a new budget middleware
	// to the shared SubAgent instance, causing stale trackers from previous tasks
	// to block tool execution in subsequent tasks.
	//
	// This test runs two phases (each with one task). The budget is set so that
	// each task's tracker is exhausted after a few tool calls. If the bug exists,
	// the second task's tool calls would be blocked by the stale tracker from phase 1.
	ctx := setupTestContext(t)
	repo := repository.NewMemory()
	testTicket := setupTicketAndAlert(t, ctx, repo)

	var mu sync.Mutex
	sessionCount := 0
	toolCallCount := 0

	mockLLM := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			mu.Lock()
			sessionCount++
			ssnNum := sessionCount
			mu.Unlock()

			ssn := newMockSession()
			localCallCount := 0

			ssn.GenerateFunc = func(ctx context.Context, input []gollem.Input, opts ...gollem.GenerateOption) (*gollem.Response, error) {
				localCallCount++

				switch {
				// Session 1: planning session → plan prompt
				case ssnNum == 1 && localCallCount == 1:
					return &gollem.Response{
						Texts: []string{`{
							"message": "Starting analysis",
							"tasks": [
								{"id": "t1", "title": "Task 1", "description": "First task", "tools": ["test_tool"], "sub_agents": []}
							]
						}`},
					}, nil

				// Session 2: task 1 agent → first call produces tool call
				case ssnNum == 2 && localCallCount == 1:
					return &gollem.Response{
						FunctionCalls: []*gollem.FunctionCall{{Name: "test_tool", Arguments: map[string]any{}}},
					}, nil
				case ssnNum == 2 && localCallCount == 2:
					return &gollem.Response{
						Texts: []string{"Task 1 done."},
					}, nil

				// Session 3: replan session → schedule task 2
				case ssnNum == 3 && localCallCount == 1:
					return &gollem.Response{
						Texts: []string{`{
							"tasks": [
								{"id": "t2", "title": "Task 2", "description": "Second task", "tools": ["test_tool"], "sub_agents": []}
							]
						}`},
					}, nil

				// Session 4: task 2 agent → produces tool call
				case ssnNum == 4 && localCallCount == 1:
					return &gollem.Response{
						FunctionCalls: []*gollem.FunctionCall{{Name: "test_tool", Arguments: map[string]any{}}},
					}, nil
				case ssnNum == 4 && localCallCount == 2:
					return &gollem.Response{
						Texts: []string{"Task 2 done."},
					}, nil

				// Session 5: replan session → no more tasks
				case ssnNum == 5 && localCallCount == 1:
					return &gollem.Response{
						Texts: []string{`{"tasks": []}`},
					}, nil

				// Session 6: final response session
				case ssnNum == 6:
					return &gollem.Response{
						Texts: []string{"All tasks completed."},
					}, nil
				}

				return &gollem.Response{Texts: []string{"OK"}}, nil
			}
			return ssn, nil
		},
	}

	// Budget: each tool call costs 30 out of 100 total.
	// Task 1 and Task 2 each make 1 tool call, consuming 30 each.
	// If trackers are independent, each task starts at 100 and ends at 70.
	// If the accumulation bug exists, task 2 would inherit task 1's depleted state.
	budgetStrategy := &recordingBudgetStrategy{
		initialBudget:   100.0,
		beforeCost:      30.0,
		hardLimitMargin: 3,
	}

	// Track tool executions
	testToolSet := &trackingToolSet{
		name: "test_tool",
		runFunc: func() {
			mu.Lock()
			toolCallCount++
			mu.Unlock()
		},
	}

	chatUC := aster.New(repo, mockLLM,
		aster.WithMaxPhases(3),
		aster.WithBudgetStrategy(budgetStrategy),
		aster.WithTools([]interfaces.ToolSet{testToolSet}),
	)
	ssn := newDummySession(testTicket.ID)

	err := chatUC.Execute(ctx, &chat.RunContext{Session: ssn, Message: "Analyze with budget", ChatCtx: &chatModel.ChatContext{Ticket: testTicket}})
	gt.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()

	// Both tasks must have executed their tool calls
	gt.N[int](t, toolCallCount).Greater(1)

	// Verify budget consumption was recorded correctly
	budgetStrategy.mu.Lock()
	calls := budgetStrategy.calls
	budgetStrategy.mu.Unlock()

	// There should be at least 2 calls (1 per task)
	gt.N[int](t, len(calls)).GreaterOrEqual(2)

	// Each call should have CallCount == 1, proving each task got a fresh tracker.
	// If the accumulation bug existed, the second task's call would have CallCount > 1
	// because the stale tracker from task 1 would carry over its state.
	for i, call := range calls {
		gt.N[int](t, call.CallCount).
			Describef("call[%d]: CallCount should be 1 (fresh tracker per task), got %d", i, call.CallCount).
			Equal(1)
	}
}

// recordingBudgetStrategy records each BeforeToolCall invocation for assertions.
type recordingBudgetStrategy struct {
	mu              sync.Mutex
	initialBudget   float64
	beforeCost      float64
	hardLimitMargin int
	calls           []aster.ToolCallContext
}

func (s *recordingBudgetStrategy) InitialBudget() float64 { return s.initialBudget }
func (s *recordingBudgetStrategy) BeforeToolCall(ctx aster.ToolCallContext) float64 {
	s.mu.Lock()
	s.calls = append(s.calls, ctx)
	s.mu.Unlock()
	return s.beforeCost
}
func (s *recordingBudgetStrategy) AfterToolCall(_ aster.ToolCallContext) float64 { return 0 }
func (s *recordingBudgetStrategy) ShouldExit(state aster.BudgetState) bool {
	return state.CallsAfterSoft > s.hardLimitMargin
}

// trackingToolSet implements gollem.ToolSet with a callback to track tool executions.
type trackingToolSet struct {
	name    string
	runFunc func()
}

func (m *trackingToolSet) Specs(_ context.Context) ([]gollem.ToolSpec, error) {
	return []gollem.ToolSpec{
		{Name: m.name, Description: "Tracking tool: " + m.name},
	}, nil
}

func (m *trackingToolSet) Run(_ context.Context, _ string, _ map[string]any) (map[string]any, error) {
	if m.runFunc != nil {
		m.runFunc()
	}
	return map[string]any{"result": "ok"}, nil
}

func (m *trackingToolSet) ID() string {
	return m.name
}

func (m *trackingToolSet) Description() string {
	return "Tracking tool: " + m.name
}

func (m *trackingToolSet) Prompt(_ context.Context) (string, error) {
	return "", nil
}

// Ensure imports are used.
var (
	_ = goerr.New
)
