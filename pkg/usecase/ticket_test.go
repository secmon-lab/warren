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
					// Return mock embedding vector with correct dimension
					embedding := make([]float64, dimension)
					for i := range embedding {
						embedding[i] = 0.1 + float64(i)*0.01 // Generate some test values
					}
					return [][]float64{embedding}, nil
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

			// Verify embedding was generated for manual tickets (title + description only)
			if tc.title != "" {
				gt.Array(t, ticket.Embedding).Length(256) // Mock returns 256-dimensional vector
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

	t.Run("valid title and description", runTest(testCase{
		title:       "Test Title",
		description: "Test Description",
		user:        &slack.User{ID: "user1", Name: "Test User"},
		expectError: false,
	}))

	t.Run("valid title only", runTest(testCase{
		title:       "Test Title",
		description: "",
		user:        &slack.User{ID: "user1", Name: "Test User"},
		expectError: false,
	}))

	t.Run("empty title", runTest(testCase{
		title:       "",
		description: "Test Description",
		user:        &slack.User{ID: "user1", Name: "Test User"},
		expectError: true,
	}))

	t.Run("nil user", runTest(testCase{
		title:       "Test Title",
		description: "Test Description",
		user:        nil,
		expectError: false,
	}))
}
