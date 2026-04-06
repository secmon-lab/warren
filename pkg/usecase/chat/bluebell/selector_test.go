package bluebell_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/cli/config"
	"github.com/secmon-lab/warren/pkg/domain/mock"
	chatModel "github.com/secmon-lab/warren/pkg/domain/model/chat"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	svcknowledge "github.com/secmon-lab/warren/pkg/service/knowledge"
	"github.com/secmon-lab/warren/pkg/usecase/chat/bluebell"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

func setupSelectorTestContext(t *testing.T) context.Context {
	t.Helper()
	ctx := t.Context()
	return msg.With(ctx,
		func(ctx context.Context, message string) {},
		func(ctx context.Context, message string) {},
		func(ctx context.Context, message string) {},
	)
}

func newSelectorMockLLM(responseJSON string) *mock.LLMClientMock {
	return &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			return &mock.LLMSessionMock{
				GenerateFunc: func(ctx context.Context, input []gollem.Input, opts ...gollem.GenerateOption) (*gollem.Response, error) {
					return &gollem.Response{Texts: []string{responseJSON}}, nil
				},
				HistoryFunc: func() (*gollem.History, error) {
					return nil, nil
				},
				AppendHistoryFunc: func(h *gollem.History) error {
					return nil
				},
			}, nil
		},
	}
}

func TestResolveIntent_ZeroCandidates(t *testing.T) {
	ctx := setupSelectorTestContext(t)
	repo := repository.NewMemory()

	embMock := &mock.EmbeddingClientMock{
		EmbeddingsFunc: func(ctx context.Context, texts []string, dim int) ([][]float32, error) {
			result := make([][]float32, len(texts))
			for i := range texts {
				result[i] = make([]float32, dim)
			}
			return result, nil
		},
	}
	knowledgeSvc := svcknowledge.New(repo, embMock)

	mockLLM := newSelectorMockLLM(`{"prompt_id":"default","intent":"Investigate the root cause of the alert."}`)

	chat, err := bluebell.New(repo, mockLLM,
		bluebell.WithKnowledgeService(knowledgeSvc),
		// No WithPromptEntries — zero candidates
	)
	gt.NoError(t, err)

	chatCtx := &chatModel.ChatContext{
		Ticket: &ticket.Ticket{ID: types.NewTicketID()},
	}

	// With 0 entries, resolver still runs (XY Detection + Intent Resolution)
	resolved := bluebell.ExportResolveIntent(ctx, chat, "test message", chatCtx)
	gt.V(t, resolved).NotNil()
	gt.V(t, resolved.PromptID).Equal("default")
	gt.V(t, resolved.Intent).Equal("Investigate the root cause of the alert.")
}

func TestResolveIntent_SingleCandidate(t *testing.T) {
	ctx := setupSelectorTestContext(t)
	repo := repository.NewMemory()

	embMock := &mock.EmbeddingClientMock{
		EmbeddingsFunc: func(ctx context.Context, texts []string, dim int) ([][]float32, error) {
			result := make([][]float32, len(texts))
			for i := range texts {
				result[i] = make([]float32, dim)
			}
			return result, nil
		},
	}
	knowledgeSvc := svcknowledge.New(repo, embMock)

	mockLLM := newSelectorMockLLM(`{"prompt_id":"infra","intent":"Investigate infrastructure availability issue related to deployment."}`)

	entries := []config.PromptEntry{
		{ID: "infra", Description: "Infrastructure incident investigation"},
	}

	chat, err := bluebell.New(repo, mockLLM,
		bluebell.WithKnowledgeService(knowledgeSvc),
		bluebell.WithPromptEntries(entries),
	)
	gt.NoError(t, err)

	chatCtx := &chatModel.ChatContext{
		Ticket: &ticket.Ticket{ID: types.NewTicketID()},
	}

	resolved := bluebell.ExportResolveIntent(ctx, chat, "check this alert", chatCtx)
	gt.V(t, resolved).NotNil()
	gt.V(t, resolved.PromptID).Equal("infra")
	gt.V(t, resolved.Intent != "").Equal(true)
}

func TestResolveIntent_MultipleCandidates(t *testing.T) {
	ctx := setupSelectorTestContext(t)
	repo := repository.NewMemory()

	embMock := &mock.EmbeddingClientMock{
		EmbeddingsFunc: func(ctx context.Context, texts []string, dim int) ([][]float32, error) {
			result := make([][]float32, len(texts))
			for i := range texts {
				result[i] = make([]float32, dim)
			}
			return result, nil
		},
	}
	knowledgeSvc := svcknowledge.New(repo, embMock)

	mockLLM := newSelectorMockLLM(`{"prompt_id":"security","intent":"Investigate potential credential compromise from unusual login location."}`)

	entries := []config.PromptEntry{
		{ID: "security", Description: "Security threat investigation"},
		{ID: "infra", Description: "Infrastructure incident investigation"},
	}

	chat, err := bluebell.New(repo, mockLLM,
		bluebell.WithKnowledgeService(knowledgeSvc),
		bluebell.WithPromptEntries(entries),
	)
	gt.NoError(t, err)

	chatCtx := &chatModel.ChatContext{
		Ticket: &ticket.Ticket{ID: types.NewTicketID()},
	}

	resolved := bluebell.ExportResolveIntent(ctx, chat, "suspicious login detected", chatCtx)
	gt.V(t, resolved).NotNil()
	gt.V(t, resolved.PromptID).Equal("security")
	gt.V(t, resolved.Intent != "").Equal(true)
}

func TestResolveIntent_UnknownPromptID_Fallback(t *testing.T) {
	ctx := setupSelectorTestContext(t)
	repo := repository.NewMemory()

	embMock := &mock.EmbeddingClientMock{
		EmbeddingsFunc: func(ctx context.Context, texts []string, dim int) ([][]float32, error) {
			result := make([][]float32, len(texts))
			for i := range texts {
				result[i] = make([]float32, dim)
			}
			return result, nil
		},
	}
	knowledgeSvc := svcknowledge.New(repo, embMock)

	// LLM returns a prompt_id that doesn't exist
	mockLLM := newSelectorMockLLM(`{"prompt_id":"nonexistent","intent":"something"}`)

	entries := []config.PromptEntry{
		{ID: "security", Description: "Security threat investigation"},
		{ID: "infra", Description: "Infrastructure incident investigation"},
	}

	chat, err := bluebell.New(repo, mockLLM,
		bluebell.WithKnowledgeService(knowledgeSvc),
		bluebell.WithPromptEntries(entries),
	)
	gt.NoError(t, err)

	chatCtx := &chatModel.ChatContext{
		Ticket: &ticket.Ticket{ID: types.NewTicketID()},
	}

	resolved := bluebell.ExportResolveIntent(ctx, chat, "test", chatCtx)
	gt.V(t, resolved == nil).Equal(true) // fallback to nil
}

func TestResolveIntent_InvalidJSON_Fallback(t *testing.T) {
	ctx := setupSelectorTestContext(t)
	repo := repository.NewMemory()

	embMock := &mock.EmbeddingClientMock{
		EmbeddingsFunc: func(ctx context.Context, texts []string, dim int) ([][]float32, error) {
			result := make([][]float32, len(texts))
			for i := range texts {
				result[i] = make([]float32, dim)
			}
			return result, nil
		},
	}
	knowledgeSvc := svcknowledge.New(repo, embMock)

	mockLLM := newSelectorMockLLM(`not valid json`)

	entries := []config.PromptEntry{
		{ID: "security", Description: "Security threat investigation"},
		{ID: "infra", Description: "Infrastructure incident investigation"},
	}

	chat, err := bluebell.New(repo, mockLLM,
		bluebell.WithKnowledgeService(knowledgeSvc),
		bluebell.WithPromptEntries(entries),
	)
	gt.NoError(t, err)

	chatCtx := &chatModel.ChatContext{
		Ticket: &ticket.Ticket{ID: types.NewTicketID()},
	}

	resolved := bluebell.ExportResolveIntent(ctx, chat, "test", chatCtx)
	gt.V(t, resolved == nil).Equal(true) // fallback to nil
}

func TestResolveIntent_DefaultPromptID_Accepted(t *testing.T) {
	ctx := setupSelectorTestContext(t)
	repo := repository.NewMemory()

	embMock := &mock.EmbeddingClientMock{
		EmbeddingsFunc: func(ctx context.Context, texts []string, dim int) ([][]float32, error) {
			result := make([][]float32, len(texts))
			for i := range texts {
				result[i] = make([]float32, dim)
			}
			return result, nil
		},
	}
	knowledgeSvc := svcknowledge.New(repo, embMock)

	mockLLM := newSelectorMockLLM(`{"prompt_id":"default","intent":"Use standard security investigation approach."}`)

	entries := []config.PromptEntry{
		{ID: "security", Description: "Security threat investigation"},
	}

	chat, err := bluebell.New(repo, mockLLM,
		bluebell.WithKnowledgeService(knowledgeSvc),
		bluebell.WithPromptEntries(entries),
	)
	gt.NoError(t, err)

	chatCtx := &chatModel.ChatContext{
		Ticket: &ticket.Ticket{ID: types.NewTicketID()},
	}

	// For single candidate, prompt_id is forced to the entry's ID
	resolved := bluebell.ExportResolveIntent(ctx, chat, "test", chatCtx)
	gt.V(t, resolved).NotNil()
	gt.V(t, resolved.PromptID).Equal("security") // forced to single entry's ID
}
