package chat_test

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/opaq"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/mock"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	chatModel "github.com/secmon-lab/warren/pkg/domain/model/chat"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	svcknowledge "github.com/secmon-lab/warren/pkg/service/knowledge"
	chatuc "github.com/secmon-lab/warren/pkg/usecase/chat"
	"github.com/secmon-lab/warren/pkg/usecase/chat/aster"
	"github.com/secmon-lab/warren/pkg/usecase/chat/bluebell"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

// --- shared test helpers ---

func newAllowPolicyClient(t *testing.T) *mock.PolicyClientMock {
	t.Helper()
	return &mock.PolicyClientMock{
		QueryFunc: func(ctx context.Context, query string, input, result any, opts ...opaq.QueryOption) error {
			if query == "data.auth.agent" {
				gt.NoError(t, json.Unmarshal([]byte(`{"allow":true}`), &result))
			}
			return nil
		},
		SourcesFunc: func() map[string]string { return map[string]string{} },
	}
}

func newDenyPolicyClient(t *testing.T) *mock.PolicyClientMock {
	t.Helper()
	return &mock.PolicyClientMock{
		QueryFunc: func(ctx context.Context, query string, input, result any, opts ...opaq.QueryOption) error {
			if query == "data.auth.agent" {
				gt.NoError(t, json.Unmarshal([]byte(`{"allow":false}`), &result))
			}
			return nil
		},
		SourcesFunc: func() map[string]string { return map[string]string{} },
	}
}

func newUndefinedPolicyClient() *mock.PolicyClientMock {
	return &mock.PolicyClientMock{
		QueryFunc: func(ctx context.Context, query string, input, result any, opts ...opaq.QueryOption) error {
			if query == "data.auth.agent" {
				return opaq.ErrNoEvalResult
			}
			return nil
		},
		SourcesFunc: func() map[string]string { return map[string]string{} },
	}
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

func newDirectResponseLLM() *mock.LLMClientMock {
	return &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			ssn := &mock.LLMSessionMock{
				HistoryFunc: func() (*gollem.History, error) {
					return &gollem.History{
						Version:  gollem.HistoryVersion,
						Messages: []gollem.Message{{Role: gollem.RoleUser}},
					}, nil
				},
				AppendHistoryFunc: func(h *gollem.History) error { return nil },
				GenerateFunc: func(ctx context.Context, input []gollem.Input, opts ...gollem.GenerateOption) (*gollem.Response, error) {
					return &gollem.Response{
						Texts: []string{`{"message": "Direct response.", "tasks": []}`},
					}, nil
				},
			}
			return ssn, nil
		},
	}
}

func setupTestContext(t *testing.T) context.Context {
	t.Helper()
	return msg.With(t.Context(),
		func(ctx context.Context, message string) {},
		func(ctx context.Context, message string) {},
		func(ctx context.Context, message string) {},
	)
}

func setupRepo(t *testing.T, ctx context.Context) (*repository.Memory, *ticket.Ticket) {
	t.Helper()
	repo := repository.NewMemory()
	tk := ticket.Ticket{
		ID:       types.NewTicketID(),
		Status:   types.TicketStatusOpen,
		AlertIDs: []types.AlertID{types.NewAlertID()},
	}
	gt.NoError(t, repo.PutTicket(ctx, tk))
	gt.NoError(t, repo.PutAlert(ctx, alert.Alert{
		ID:     tk.AlertIDs[0],
		Schema: "test.alert",
		Data:   map[string]any{"test": "data"},
	}))
	return repo, &tk
}

// chatUCFactory builds a ChatUseCase for a given strategy name.
type chatUCFactory struct {
	repo         *repository.Memory
	llm          *mock.LLMClientMock
	policy       *mock.PolicyClientMock
	knowledgeSvc *svcknowledge.Service
}

func (f *chatUCFactory) buildStrategy(t *testing.T, strategy string) chatuc.Strategy {
	t.Helper()
	switch strategy {
	case "aster":
		return aster.New(f.repo, f.llm)
	case "bluebell":
		if f.knowledgeSvc == nil {
			f.knowledgeSvc = svcknowledge.New(f.repo, newMockEmbeddingClient())
		}
		s, err := bluebell.New(f.repo, f.llm,
			bluebell.WithKnowledgeService(f.knowledgeSvc),
		)
		gt.NoError(t, err)
		return s
	default:
		t.Fatalf("unknown strategy: %s", strategy)
		return nil
	}
}

func (f *chatUCFactory) build(t *testing.T, strategy string) interfaces.ChatUseCase {
	t.Helper()
	return chatuc.NewUseCase(f.buildStrategy(t, strategy),
		chatuc.WithRepository(f.repo),
		chatuc.WithPolicyClient(f.policy),
		chatuc.WithNoAuthorization(false),
	)
}

func (f *chatUCFactory) buildNoAuthz(t *testing.T, strategy string) interfaces.ChatUseCase {
	t.Helper()
	return chatuc.NewUseCase(f.buildStrategy(t, strategy),
		chatuc.WithRepository(f.repo),
		chatuc.WithPolicyClient(f.policy),
		chatuc.WithNoAuthorization(true),
	)
}

// --- characterization tests ---

// TestSessionCreated verifies that a Warren session is created and persisted
// in the repository when Execute is called.
func TestSessionCreated(t *testing.T) {
	strategies := []string{"aster", "bluebell"}
	for _, s := range strategies {
		t.Run(s, func(t *testing.T) {
			ctx := setupTestContext(t)
			repo, tk := setupRepo(t, ctx)

			factory := &chatUCFactory{
				repo:   repo,
				llm:    newDirectResponseLLM(),
				policy: newAllowPolicyClient(t),
			}
			uc := factory.buildNoAuthz(t, s)

			err := uc.Execute(ctx, "hello", chatModel.ChatContext{Ticket: tk})
			gt.NoError(t, err)

			sessions, err := repo.GetSessionsByTicket(ctx, tk.ID)
			gt.NoError(t, err)
			gt.N(t, len(sessions)).GreaterOrEqual(1)
		})
	}
}

// TestSessionStatusCompleted verifies that the Warren session status is set to
// "completed" after successful execution.
func TestSessionStatusCompleted(t *testing.T) {
	strategies := []string{"aster", "bluebell"}
	for _, s := range strategies {
		t.Run(s, func(t *testing.T) {
			ctx := setupTestContext(t)
			repo, tk := setupRepo(t, ctx)

			factory := &chatUCFactory{
				repo:   repo,
				llm:    newDirectResponseLLM(),
				policy: newAllowPolicyClient(t),
			}
			uc := factory.buildNoAuthz(t, s)

			err := uc.Execute(ctx, "hello", chatModel.ChatContext{Ticket: tk})
			gt.NoError(t, err)

			sessions, err := repo.GetSessionsByTicket(ctx, tk.ID)
			gt.NoError(t, err)
			gt.N(t, len(sessions)).GreaterOrEqual(1)
			gt.V(t, sessions[0].Status).Equal(types.SessionStatusCompleted)
		})
	}
}

// TestAuthorizationDenied verifies that when auth policy denies the request,
// the LLM is never called and a notification is sent.
func TestAuthorizationDenied(t *testing.T) {
	strategies := []string{"aster", "bluebell"}
	for _, s := range strategies {
		t.Run(s, func(t *testing.T) {
			ctx := setupTestContext(t)
			repo, tk := setupRepo(t, ctx)

			var mu sync.Mutex
			var notifications []string
			ctx = msg.With(ctx,
				func(ctx context.Context, message string) {
					mu.Lock()
					notifications = append(notifications, message)
					mu.Unlock()
				},
				func(ctx context.Context, message string) {},
				func(ctx context.Context, message string) {},
			)

			llmCalled := false
			mockLLM := &mock.LLMClientMock{
				NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
					llmCalled = true
					return nil, nil
				},
			}

			factory := &chatUCFactory{
				repo:   repo,
				llm:    mockLLM,
				policy: newDenyPolicyClient(t),
			}
			uc := factory.build(t, s)

			err := uc.Execute(ctx, "analyze this", chatModel.ChatContext{Ticket: tk})
			gt.NoError(t, err)

			gt.V(t, llmCalled).Equal(false)

			mu.Lock()
			defer mu.Unlock()
			found := false
			for _, n := range notifications {
				if strings.Contains(n, "Authorization Failed") {
					found = true
					break
				}
			}
			gt.V(t, found).Equal(true)
		})
	}
}

// TestAuthorizationPolicyNotDefined verifies that when auth policy is not defined,
// LLM is not called and a notification about undefined policy is sent.
func TestAuthorizationPolicyNotDefined(t *testing.T) {
	strategies := []string{"aster", "bluebell"}
	for _, s := range strategies {
		t.Run(s, func(t *testing.T) {
			ctx := setupTestContext(t)
			repo, tk := setupRepo(t, ctx)

			var mu sync.Mutex
			var notifications []string
			ctx = msg.With(ctx,
				func(ctx context.Context, message string) {
					mu.Lock()
					notifications = append(notifications, message)
					mu.Unlock()
				},
				func(ctx context.Context, message string) {},
				func(ctx context.Context, message string) {},
			)

			llmCalled := false
			mockLLM := &mock.LLMClientMock{
				NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
					llmCalled = true
					return nil, nil
				},
			}

			factory := &chatUCFactory{
				repo:   repo,
				llm:    mockLLM,
				policy: newUndefinedPolicyClient(),
			}
			uc := factory.build(t, s)

			err := uc.Execute(ctx, "analyze this", chatModel.ChatContext{Ticket: tk})
			gt.NoError(t, err)

			gt.V(t, llmCalled).Equal(false)

			mu.Lock()
			defer mu.Unlock()
			found := false
			for _, n := range notifications {
				if strings.Contains(n, "policy is not defined") {
					found = true
					break
				}
			}
			gt.V(t, found).Equal(true)
		})
	}
}

// TestNoAuthzBypass verifies that WithNoAuthorization(true) bypasses auth checks
// and the strategy executes normally.
func TestNoAuthzBypass(t *testing.T) {
	strategies := []string{"aster", "bluebell"}
	for _, s := range strategies {
		t.Run(s, func(t *testing.T) {
			ctx := setupTestContext(t)
			repo, tk := setupRepo(t, ctx)

			factory := &chatUCFactory{
				repo:   repo,
				llm:    newDirectResponseLLM(),
				policy: newDenyPolicyClient(t),
			}
			// buildNoAuthz sets WithNoAuthorization(true)
			uc := factory.buildNoAuthz(t, s)

			err := uc.Execute(ctx, "hello", chatModel.ChatContext{Ticket: tk})
			gt.NoError(t, err)

			sessions, err := repo.GetSessionsByTicket(ctx, tk.ID)
			gt.NoError(t, err)
			gt.N(t, len(sessions)).GreaterOrEqual(1)
			gt.V(t, sessions[0].Status).Equal(types.SessionStatusCompleted)
		})
	}
}

// TestNotifyMessageDelivered verifies that msg.Notify is called with the LLM
// response message during execution.
func TestNotifyMessageDelivered(t *testing.T) {
	strategies := []string{"aster", "bluebell"}
	for _, s := range strategies {
		t.Run(s, func(t *testing.T) {
			ctx := setupTestContext(t)
			repo, tk := setupRepo(t, ctx)

			var mu sync.Mutex
			var notifications []string
			ctx = msg.With(ctx,
				func(ctx context.Context, message string) {
					mu.Lock()
					notifications = append(notifications, message)
					mu.Unlock()
				},
				func(ctx context.Context, message string) {},
				func(ctx context.Context, message string) {},
			)

			factory := &chatUCFactory{
				repo:   repo,
				llm:    newDirectResponseLLM(),
				policy: newAllowPolicyClient(t),
			}
			uc := factory.buildNoAuthz(t, s)

			err := uc.Execute(ctx, "hello", chatModel.ChatContext{Ticket: tk})
			gt.NoError(t, err)

			mu.Lock()
			defer mu.Unlock()
			found := false
			for _, n := range notifications {
				if strings.Contains(n, "Direct response.") {
					found = true
					break
				}
			}
			gt.V(t, found).Equal(true)
		})
	}
}

// TestSessionCreatedEvenWhenAuthDenied verifies that a Warren session is created
// even when authorization is denied (session creation happens before auth check).
func TestSessionCreatedEvenWhenAuthDenied(t *testing.T) {
	strategies := []string{"aster", "bluebell"}
	for _, s := range strategies {
		t.Run(s, func(t *testing.T) {
			ctx := setupTestContext(t)
			repo, tk := setupRepo(t, ctx)

			factory := &chatUCFactory{
				repo:   repo,
				llm:    newDirectResponseLLM(),
				policy: newDenyPolicyClient(t),
			}
			uc := factory.build(t, s)

			err := uc.Execute(ctx, "denied request", chatModel.ChatContext{Ticket: tk})
			gt.NoError(t, err)

			sessions, err := repo.GetSessionsByTicket(ctx, tk.ID)
			gt.NoError(t, err)
			gt.N(t, len(sessions)).GreaterOrEqual(1)
		})
	}
}
