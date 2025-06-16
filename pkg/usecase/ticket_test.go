package usecase

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/mock"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/repository"
)

func TestCreateManualTicket(t *testing.T) {
	type testCase struct {
		title       string
		description string
		user        *slack.User
		expectError bool
	}

	runTest := func(tc testCase) func(t *testing.T) {
		return func(t *testing.T) {
			ctx := context.Background()
			repo := repository.NewMemory()

			// Create LLM client mock for embedding generation
			llmMock := &mock.LLMClientMock{
				GenerateEmbeddingFunc: func(ctx context.Context, dimension int, input []string) ([][]float64, error) {
					// Return mock embedding vector
					return [][]float64{{0.1, 0.2, 0.3}}, nil
				},
			}

			uc := New(WithRepository(repo), WithLLMClient(llmMock))

			ticket, err := uc.CreateManualTicket(ctx, tc.title, tc.description, tc.user)

			if tc.expectError {
				gt.Error(t, err)
				return
			}

			gt.NoError(t, err)
			gt.Value(t, ticket).NotNil()
			gt.Value(t, ticket.Metadata.Title).Equal(tc.title)
			gt.Value(t, ticket.Metadata.Description).Equal(tc.description)
			gt.Value(t, ticket.Assignee).Equal(tc.user)
			gt.Array(t, ticket.AlertIDs).Length(0)         // Should be empty array
			gt.Value(t, ticket.Metadata.Summary).Equal("") // Should be empty per requirements

			// Verify embedding was generated
			if tc.title != "" || tc.description != "" {
				gt.Array(t, ticket.Embedding).Length(3) // Mock returns 3-dimensional vector
			}

			// Verify ticket was saved to repository
			savedTicket, err := repo.GetTicket(ctx, ticket.ID)
			gt.NoError(t, err)
			gt.Value(t, savedTicket.ID).Equal(ticket.ID)
			gt.Value(t, savedTicket.Metadata.Title).Equal(tc.title)

			// Verify LLM was called for embedding generation
			gt.Array(t, llmMock.GenerateEmbeddingCalls()).Length(1)
		}
	}

	t.Run("success case", runTest(testCase{
		title:       "Test Manual Ticket",
		description: "This is a test description",
		user: &slack.User{
			ID:   "U123456",
			Name: "Test User",
		},
		expectError: false,
	}))

	t.Run("success with empty user", runTest(testCase{
		title:       "Test Manual Ticket",
		description: "This is a test description",
		user:        nil,
		expectError: false,
	}))

	t.Run("success with empty description", runTest(testCase{
		title:       "Test Manual Ticket",
		description: "",
		user: &slack.User{
			ID:   "U123456",
			Name: "Test User",
		},
		expectError: false,
	}))

	t.Run("error with empty title", runTest(testCase{
		title:       "",
		description: "This is a test description",
		user: &slack.User{
			ID:   "U123456",
			Name: "Test User",
		},
		expectError: true,
	}))
}
