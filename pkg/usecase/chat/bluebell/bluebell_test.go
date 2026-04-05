package bluebell_test

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
	"github.com/secmon-lab/warren/pkg/cli/config"
	"github.com/secmon-lab/warren/pkg/domain/mock"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	chatModel "github.com/secmon-lab/warren/pkg/domain/model/chat"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	svcknowledge "github.com/secmon-lab/warren/pkg/service/knowledge"
	"github.com/secmon-lab/warren/pkg/usecase/chat/bluebell"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

func newMockPolicyClient(t *testing.T) *mock.PolicyClientMock {
	t.Helper()
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
	mockPolicy := newMockPolicyClient(t)

	_, err := bluebell.New(repo, mockLLM, mockPolicy)
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

	chatUC, err := bluebell.New(repo, mockLLM, newMockPolicyClient(t),
		bluebell.WithNoAuthorization(true),
		bluebell.WithKnowledgeService(knowledgeSvc),
	)
	gt.NoError(t, err)

	err = chatUC.Execute(ctx, "What is the meaning of life?", chatModel.ChatContext{Ticket: testTicket})
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
											"tools": []
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

	chatUC, err := bluebell.New(repo, mockLLM, newMockPolicyClient(t),
		bluebell.WithNoAuthorization(true),
		bluebell.WithKnowledgeService(knowledgeSvc),
	)
	gt.NoError(t, err)

	err = chatUC.Execute(ctx, "Analyze this alert", chatModel.ChatContext{Ticket: testTicket})
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

	chatUC, err := bluebell.New(repo, mockLLM, newMockPolicyClient(t),
		bluebell.WithNoAuthorization(true),
		bluebell.WithKnowledgeService(knowledgeSvc),
		bluebell.WithPromptEntries(entries),
	)
	gt.NoError(t, err)

	err = chatUC.Execute(ctx, "Check this alert", chatModel.ChatContext{Ticket: testTicket})
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
										"tools": []
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

	chatUC, err := bluebell.New(repo, mockLLM, newMockPolicyClient(t),
		bluebell.WithNoAuthorization(true),
		bluebell.WithKnowledgeService(knowledgeSvc),
		bluebell.WithMaxPhases(2),
	)
	gt.NoError(t, err)

	err = chatUC.Execute(ctx, "Do something", chatModel.ChatContext{Ticket: testTicket})
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
										{"id": "t-ok", "title": "Succeeding task", "description": "This will succeed", "tools": []},
										{"id": "t-fail", "title": "Failing task", "description": "This will fail", "tools": []}
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

	chatUC, err := bluebell.New(repo, mockLLM, newMockPolicyClient(t),
		bluebell.WithNoAuthorization(true),
		bluebell.WithKnowledgeService(knowledgeSvc),
	)
	gt.NoError(t, err)

	err = chatUC.Execute(ctx, "Test error isolation", chatModel.ChatContext{Ticket: testTicket})
	gt.NoError(t, err)
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

	chatUC, err := bluebell.New(repo, mockLLM, newMockPolicyClient(t),
		bluebell.WithNoAuthorization(true),
		bluebell.WithKnowledgeService(knowledgeSvc),
	)
	gt.NoError(t, err)

	// Ticketless: empty ticket
	err = chatUC.Execute(ctx, "General question", chatModel.ChatContext{
		Ticket: &ticket.Ticket{},
	})
	gt.NoError(t, err)
}
