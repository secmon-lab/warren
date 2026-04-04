package aster_test

import (
	"context"
	"sync"
	"testing"

	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/mock"
	chatModel "github.com/secmon-lab/warren/pkg/domain/model/chat"
	"github.com/secmon-lab/warren/pkg/repository"
	svcknowledge "github.com/secmon-lab/warren/pkg/service/knowledge"
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

	chatUC := aster.New(repo, mockLLM, newMockPolicyClient(t),
		aster.WithKnowledgeService(knowledgeSvc),
		aster.WithNoAuthorization(true),
	)

	err := chatUC.Execute(ctx, "Analyze this alert", chatModel.ChatContext{Ticket: testTicket})
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

	chatUC := aster.New(repo, mockLLM, newMockPolicyClient(t),
		aster.WithNoAuthorization(true),
	)

	err := chatUC.Execute(ctx, "Analyze this alert", chatModel.ChatContext{Ticket: testTicket})
	gt.NoError(t, err)

	// Without knowledge service, only one session should be created (planSession).
	mu.Lock()
	gt.V(t, sessionCount).Equal(1)
	mu.Unlock()
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
