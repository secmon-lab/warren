package usecase

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/mock"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/utils/clock"
)

func TestGenerateResolveMessage(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx = clock.With(ctx, func() time.Time { return now })

	runTest := func(tc struct {
		name           string
		ticket         *ticket.Ticket
		llmResponse    string
		llmError       error
		expectedPrefix string
	}) func(t *testing.T) {
		return func(t *testing.T) {
			// Setup LLM mock
			llmMock := &mock.LLMClientMock{
				NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
					return &mock.LLMSessionMock{
						GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
							if tc.llmError != nil {
								return nil, tc.llmError
							}
							return &gollem.Response{
								Texts: []string{tc.llmResponse},
							}, nil
						},
					}, nil
				},
				GenerateEmbeddingFunc: func(ctx context.Context, dimension int, inputs []string) ([][]float64, error) {
					// Return mock embedding data with correct dimension
					embedding := make([]float64, dimension)
					for i := range embedding {
						embedding[i] = 0.1 + float64(i)*0.01 // Generate some test values
					}
					return [][]float64{embedding}, nil
				},
			}

			// Create usecase
			uc := New(
				WithLLMClient(llmMock),
				WithRepository(repository.NewMemory()),
			)

			// Test the generateResolveMessage function
			message := uc.generateResolveMessage(ctx, tc.ticket)

			// Verify the result
			if tc.expectedPrefix != "" {
				// Check if message starts with the expected prefix or contains it
				gt.Value(t, strings.Contains(message, tc.expectedPrefix)).Equal(true)
			} else {
				gt.Value(t, message).Equal(tc.llmResponse)
			}
		}
	}

	t.Run("success with conclusion", runTest(struct {
		name           string
		ticket         *ticket.Ticket
		llmResponse    string
		llmError       error
		expectedPrefix string
	}{
		name: "success with conclusion",
		ticket: &ticket.Ticket{
			ID:         types.NewTicketID(),
			Status:     types.TicketStatusResolved,
			Conclusion: types.AlertConclusionFalsePositive,
			Reason:     "False positive detection",
			Metadata: ticket.Metadata{
				Title: "Test Alert",
			},
		},
		llmResponse: "Great work! It was a false positive, but good job on the thorough investigation 🎯",
		llmError:    nil,
	}))

	t.Run("success without conclusion", runTest(struct {
		name           string
		ticket         *ticket.Ticket
		llmResponse    string
		llmError       error
		expectedPrefix string
	}{
		name: "success without conclusion",
		ticket: &ticket.Ticket{
			ID:     types.NewTicketID(),
			Status: types.TicketStatusResolved,
			Reason: "Response completed successfully",
			Metadata: ticket.Metadata{
				Title: "Network Alert",
			},
		},
		llmResponse: "Resolution complete! Another heroic day protecting the world's peace 🦸‍♂️",
		llmError:    nil,
	}))

	t.Run("llm error fallback", runTest(struct {
		name           string
		ticket         *ticket.Ticket
		llmResponse    string
		llmError       error
		expectedPrefix string
	}{
		name: "llm error fallback",
		ticket: &ticket.Ticket{
			ID:     types.NewTicketID(),
			Status: types.TicketStatusResolved,
			Metadata: ticket.Metadata{
				Title: "Test Alert",
			},
		},
		llmResponse:    "",
		llmError:       goerr.New("LLM error"),
		expectedPrefix: "🎉 Great work! Ticket resolved successfully 🎯",
	}))

	t.Run("empty response fallback", runTest(struct {
		name           string
		ticket         *ticket.Ticket
		llmResponse    string
		llmError       error
		expectedPrefix string
	}{
		name: "empty response fallback",
		ticket: &ticket.Ticket{
			ID:     types.NewTicketID(),
			Status: types.TicketStatusResolved,
			Metadata: ticket.Metadata{
				Title: "Test Alert",
			},
		},
		llmResponse:    "",
		llmError:       nil,
		expectedPrefix: "🎉 Great work! Ticket resolved successfully 🎯",
	}))
}
