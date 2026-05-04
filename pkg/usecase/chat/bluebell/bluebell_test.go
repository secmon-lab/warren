package bluebell_test

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/cli/config"
	"github.com/secmon-lab/warren/pkg/cli/config/llm"
	"github.com/secmon-lab/warren/pkg/domain/mock"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	chatModel "github.com/secmon-lab/warren/pkg/domain/model/chat"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	slackModel "github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	svcknowledge "github.com/secmon-lab/warren/pkg/service/knowledge"
	slackService "github.com/secmon-lab/warren/pkg/service/slack"
	"github.com/secmon-lab/warren/pkg/usecase/chat"
	"github.com/secmon-lab/warren/pkg/usecase/chat/bluebell"
	"github.com/secmon-lab/warren/pkg/utils/msg"

	slack "github.com/slack-go/slack"
)

func newDummySession(ticketID types.TicketID) *session.Session {
	return &session.Session{
		ID:       types.NewSessionID(),
		TicketID: ticketID,
		Status:   types.SessionStatusRunning,
	}
}

func setupTestContext(t *testing.T) context.Context {
	t.Helper()
	ctx := t.Context()
	return msg.With(ctx,
		func(ctx context.Context, message string) {},
		func(ctx context.Context, message string) {},
		func(ctx context.Context, message string) {},
	)
}

func newMockEmbeddingClient() *mock.EmbeddingClientMock {
	return &mock.EmbeddingClientMock{
		EmbeddingsFunc: func(ctx context.Context, texts []string, dim int) ([][]float32, error) {
			result := make([][]float32, len(texts))
			for i := range texts {
				result[i] = make([]float32, dim)
			}
			return result, nil
		},
	}
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
	t.Helper()
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

func TestBluebellChat_NewRequiresKnowledgeService(t *testing.T) {
	repo := repository.NewMemory()
	mockLLM := &mock.LLMClientMock{}

	_, err := bluebell.New(repo, llm.SingleClientRegistryForTest(mockLLM))
	gt.V(t, err).NotNil()
	gt.True(t, strings.Contains(err.Error(), "requires knowledge service"))
}

func TestBluebellChat_DirectResponse(t *testing.T) {
	ctx := setupTestContext(t)
	repo := repository.NewMemory()
	testTicket := setupTicketAndAlert(t, ctx, repo)
	knowledgeSvc := svcknowledge.New(repo, newMockEmbeddingClient())

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

	chatUC, err := bluebell.New(repo, llm.SingleClientRegistryForTest(mockLLM),
		bluebell.WithKnowledgeService(knowledgeSvc),
	)
	gt.NoError(t, err)

	err = chatUC.Execute(ctx, &chat.RunContext{
		Session: newDummySession(testTicket.ID),
		Message: "What is the meaning of life?",
		ChatCtx: &chatModel.ChatContext{Ticket: testTicket},
	})
	gt.NoError(t, err)
}

func TestBluebellChat_SinglePhaseWithTasks(t *testing.T) {
	ctx := setupTestContext(t)
	repo := repository.NewMemory()
	testTicket := setupTicketAndAlert(t, ctx, repo)
	knowledgeSvc := svcknowledge.New(repo, newMockEmbeddingClient())

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
											"tools": [], "llm_id": "test"
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

	chatUC, err := bluebell.New(repo, llm.SingleClientRegistryForTest(mockLLM),
		bluebell.WithKnowledgeService(knowledgeSvc),
	)
	gt.NoError(t, err)

	err = chatUC.Execute(ctx, &chat.RunContext{
		Session: newDummySession(testTicket.ID),
		Message: "Analyze this alert",
		ChatCtx: &chatModel.ChatContext{Ticket: testTicket},
	})
	gt.NoError(t, err)
}

func TestBluebellChat_WithPromptEntries_IntentInjected(t *testing.T) {
	ctx := setupTestContext(t)
	repo := repository.NewMemory()
	testTicket := setupTicketAndAlert(t, ctx, repo)
	knowledgeSvc := svcknowledge.New(repo, newMockEmbeddingClient())

	entries := []config.PromptEntry{
		{ID: "infra", Description: "Infrastructure incident investigation"},
	}

	var mu sync.Mutex
	callCount := 0

	mockLLM := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			mu.Lock()
			callCount++
			cc := callCount
			mu.Unlock()

			// First session is the selector
			if cc == 1 {
				ssn := newMockSession()
				ssn.GenerateFunc = func(ctx context.Context, input []gollem.Input, opts ...gollem.GenerateOption) (*gollem.Response, error) {
					return &gollem.Response{
						Texts: []string{`{"prompt_id":"infra","intent":"Investigate deployment-related availability issue."}`},
					}, nil
				}
				return ssn, nil
			}

			// Subsequent sessions
			ssn := newMockSession()
			ssn.GenerateFunc = func(ctx context.Context, input []gollem.Input, opts ...gollem.GenerateOption) (*gollem.Response, error) {
				return &gollem.Response{
					Texts: []string{`{"message": "Direct response.", "tasks": []}`},
				}, nil
			}
			return ssn, nil
		},
	}

	chatUC, err := bluebell.New(repo, llm.SingleClientRegistryForTest(mockLLM),
		bluebell.WithKnowledgeService(knowledgeSvc),
		bluebell.WithPromptEntries(entries),
	)
	gt.NoError(t, err)

	err = chatUC.Execute(ctx, &chat.RunContext{
		Session: newDummySession(testTicket.ID),
		Message: "Check this alert",
		ChatCtx: &chatModel.ChatContext{Ticket: testTicket},
	})
	gt.NoError(t, err)
}

func TestBluebellChat_MaxPhasesLimit(t *testing.T) {
	ctx := setupTestContext(t)
	repo := repository.NewMemory()
	testTicket := setupTicketAndAlert(t, ctx, repo)
	knowledgeSvc := svcknowledge.New(repo, newMockEmbeddingClient())

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
										"tools": [], "llm_id": "test"
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

	chatUC, err := bluebell.New(repo, llm.SingleClientRegistryForTest(mockLLM),
		bluebell.WithKnowledgeService(knowledgeSvc),
		bluebell.WithMaxPhases(2),
	)
	gt.NoError(t, err)

	err = chatUC.Execute(ctx, &chat.RunContext{
		Session: newDummySession(testTicket.ID),
		Message: "Do something",
		ChatCtx: &chatModel.ChatContext{Ticket: testTicket},
	})
	gt.NoError(t, err)
}

func TestBluebellChat_ErrorIsolation(t *testing.T) {
	ctx := setupTestContext(t)
	repo := repository.NewMemory()
	testTicket := setupTicketAndAlert(t, ctx, repo)
	knowledgeSvc := svcknowledge.New(repo, newMockEmbeddingClient())

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
									"message": "Analyzing...",
									"tasks": [
										{"id": "t-ok", "title": "Succeeding task", "description": "This will succeed", "tools": [], "llm_id": "test"},
										{"id": "t-fail", "title": "Failing task", "description": "This will fail", "tools": [], "llm_id": "test"}
									]
								}`},
							}, nil
						}
						if strings.Contains(inputStr, "Completed Task Results") || strings.Contains(inputStr, "Phase") {
							return &gollem.Response{
								Texts: []string{`{"tasks": []}`},
							}, nil
						}
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

	chatUC, err := bluebell.New(repo, llm.SingleClientRegistryForTest(mockLLM),
		bluebell.WithKnowledgeService(knowledgeSvc),
	)
	gt.NoError(t, err)

	err = chatUC.Execute(ctx, &chat.RunContext{
		Session: newDummySession(testTicket.ID),
		Message: "Test error isolation",
		ChatCtx: &chatModel.ChatContext{Ticket: testTicket},
	})
	gt.NoError(t, err)
}

// newTestSlackService creates a slack.Service with a mock client for testing.
// The returned postedMessages slice captures all PostMessageContext calls.
func newTestSlackService(t *testing.T) (*slackService.Service, *[]slack.MsgOption) {
	t.Helper()
	var mu sync.Mutex
	var postedOptions []slack.MsgOption

	slackMock := &mock.SlackClientMock{
		PostMessageContextFunc: func(ctx context.Context, channelID string, options ...slack.MsgOption) (string, string, error) {
			mu.Lock()
			postedOptions = append(postedOptions, options...)
			mu.Unlock()
			return channelID, "1234567890.123456", nil
		},
		UpdateMessageContextFunc: func(ctx context.Context, channelID, timestamp string, options ...slack.MsgOption) (string, string, string, error) {
			return channelID, timestamp, "", nil
		},
		AuthTestFunc: func() (*slack.AuthTestResponse, error) {
			return &slack.AuthTestResponse{
				UserID:       "U123",
				TeamID:       "T123",
				Team:         "test-team",
				EnterpriseID: "",
				BotID:        "B123",
			}, nil
		},
		GetTeamInfoFunc: func() (*slack.TeamInfo, error) {
			return &slack.TeamInfo{
				Domain: "test-workspace",
			}, nil
		},
	}

	svc, err := slackService.New(slackMock, "C1234567890")
	gt.NoError(t, err)
	return svc, &postedOptions
}

func setupTicketWithSlackThread(t *testing.T, ctx context.Context, repo *repository.Memory) *ticket.Ticket {
	t.Helper()
	testTicket := ticket.Ticket{
		ID:       types.NewTicketID(),
		Status:   types.TicketStatusOpen,
		AlertIDs: []types.AlertID{types.NewAlertID()},
		SlackThread: &slackModel.Thread{
			ChannelID: "C1234567890",
			ThreadID:  "1234567890.000000",
		},
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

func TestBluebellChat_ContextBlock_ZeroEntries(t *testing.T) {
	ctx := setupTestContext(t)
	repo := repository.NewMemory()
	testTicket := setupTicketWithSlackThread(t, ctx, repo)
	knowledgeSvc := svcknowledge.New(repo, newMockEmbeddingClient())
	slackSvc, postedOptions := newTestSlackService(t)

	var mu sync.Mutex
	sessionCount := 0
	mockLLM := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			mu.Lock()
			sessionCount++
			sc := sessionCount
			mu.Unlock()

			ssn := newMockSession()
			if sc == 1 {
				// Resolver session (0 entries: resolver only)
				ssn.GenerateFunc = func(ctx context.Context, input []gollem.Input, opts ...gollem.GenerateOption) (*gollem.Response, error) {
					return &gollem.Response{
						Texts: []string{`{"prompt_id":"default","intent":"Investigate root cause of the alert."}`},
					}, nil
				}
			} else {
				// Planning session
				ssn.GenerateFunc = func(ctx context.Context, input []gollem.Input, opts ...gollem.GenerateOption) (*gollem.Response, error) {
					return &gollem.Response{
						Texts: []string{`{"message": "Direct response.", "tasks": []}`},
					}, nil
				}
			}
			return ssn, nil
		},
	}

	chatUC, err := bluebell.New(repo, llm.SingleClientRegistryForTest(mockLLM),
		bluebell.WithKnowledgeService(knowledgeSvc),
		bluebell.WithSlackService(slackSvc),
		// No WithPromptEntries — 0 entries, resolver still runs
	)
	gt.NoError(t, err)

	err = chatUC.Execute(ctx, &chat.RunContext{
		Session: newDummySession(testTicket.ID),
		Message: "Investigate this alert",
		ChatCtx: &chatModel.ChatContext{Ticket: testTicket},
	})
	gt.NoError(t, err)

	// Verify context block with "Executing as (default)" was posted
	found := false
	for _, opt := range *postedOptions {
		rendered := renderMsgOption(opt)
		if strings.Contains(rendered, " as ") && strings.Contains(rendered, "(default)") {
			found = true
			break
		}
	}
	gt.True(t, found)
}

func TestBluebellChat_ContextBlock_WithPromptEntry(t *testing.T) {
	ctx := setupTestContext(t)
	repo := repository.NewMemory()
	testTicket := setupTicketWithSlackThread(t, ctx, repo)
	knowledgeSvc := svcknowledge.New(repo, newMockEmbeddingClient())
	slackSvc, postedOptions := newTestSlackService(t)

	entries := []config.PromptEntry{
		{ID: "security", Name: "Security Investigation", Description: "Security threat investigation"},
	}

	var mu sync.Mutex
	sessionCount := 0
	mockLLM := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			mu.Lock()
			sessionCount++
			sc := sessionCount
			mu.Unlock()

			ssn := newMockSession()
			if sc == 1 {
				// Selector session
				ssn.GenerateFunc = func(ctx context.Context, input []gollem.Input, opts ...gollem.GenerateOption) (*gollem.Response, error) {
					return &gollem.Response{
						Texts: []string{`{"prompt_id":"security","intent":"Investigate credential compromise."}`},
					}, nil
				}
			} else {
				// Planning session
				ssn.GenerateFunc = func(ctx context.Context, input []gollem.Input, opts ...gollem.GenerateOption) (*gollem.Response, error) {
					return &gollem.Response{
						Texts: []string{`{"message": "Direct response.", "tasks": []}`},
					}, nil
				}
			}
			return ssn, nil
		},
	}

	chatUC, err := bluebell.New(repo, llm.SingleClientRegistryForTest(mockLLM),
		bluebell.WithKnowledgeService(knowledgeSvc),
		bluebell.WithSlackService(slackSvc),
		bluebell.WithPromptEntries(entries),
	)
	gt.NoError(t, err)

	err = chatUC.Execute(ctx, &chat.RunContext{
		Session: newDummySession(testTicket.ID),
		Message: "Check suspicious login",
		ChatCtx: &chatModel.ChatContext{Ticket: testTicket},
	})
	gt.NoError(t, err)

	// Verify context block with prompt name was posted
	found := false
	for _, opt := range *postedOptions {
		rendered := renderMsgOption(opt)
		if strings.Contains(rendered, " as ") && strings.Contains(rendered, "Security Investigation") {
			found = true
			break
		}
	}
	gt.True(t, found)
}

func TestBluebellChat_ContextBlock_NoSlackThread(t *testing.T) {
	ctx := setupTestContext(t)
	repo := repository.NewMemory()
	// Ticket WITHOUT SlackThread
	testTicket := setupTicketAndAlert(t, ctx, repo)
	knowledgeSvc := svcknowledge.New(repo, newMockEmbeddingClient())
	slackSvc, postedOptions := newTestSlackService(t)

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

	chatUC, err := bluebell.New(repo, llm.SingleClientRegistryForTest(mockLLM),
		bluebell.WithKnowledgeService(knowledgeSvc),
		bluebell.WithSlackService(slackSvc),
	)
	gt.NoError(t, err)

	err = chatUC.Execute(ctx, &chat.RunContext{
		Session: newDummySession(testTicket.ID),
		Message: "Test without slack thread",
		ChatCtx: &chatModel.ChatContext{Ticket: testTicket},
	})
	gt.NoError(t, err)

	// No context block should be posted (no SlackThread)
	for _, opt := range *postedOptions {
		rendered := renderMsgOption(opt)
		gt.V(t, strings.Contains(rendered, " as ")).Equal(false)
	}
}

// renderMsgOption extracts the blocks JSON from a slack.MsgOption for test assertions.
func renderMsgOption(opt slack.MsgOption) string {
	_, values, _ := slack.UnsafeApplyMsgOptions("", "", "", opt)
	return values.Get("blocks")
}

func TestBluebellChat_Ticketless(t *testing.T) {
	ctx := setupTestContext(t)
	repo := repository.NewMemory()
	knowledgeSvc := svcknowledge.New(repo, newMockEmbeddingClient())

	mockLLM := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			ssn := newMockSession()
			ssn.GenerateFunc = func(ctx context.Context, input []gollem.Input, opts ...gollem.GenerateOption) (*gollem.Response, error) {
				return &gollem.Response{
					Texts: []string{`{"message": "Here is the answer.", "tasks": []}`},
				}, nil
			}
			return ssn, nil
		},
	}

	chatUC, err := bluebell.New(repo, llm.SingleClientRegistryForTest(mockLLM),
		bluebell.WithKnowledgeService(knowledgeSvc),
	)
	gt.NoError(t, err)

	// Ticketless: empty ticket
	err = chatUC.Execute(ctx, &chat.RunContext{
		Session: newDummySession(""),
		Message: "General question",
		ChatCtx: &chatModel.ChatContext{
			Ticket: &ticket.Ticket{},
		},
	})
	gt.NoError(t, err)
}
