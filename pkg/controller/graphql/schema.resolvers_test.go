package graphql

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/mock"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/usecase"
)

func TestUpdateTicketConclusion(t *testing.T) {
	repo := repository.NewMemory()

	// Create LLM client mock for embedding generation
	llmMock := &mock.LLMClientMock{
		GenerateEmbeddingFunc: func(ctx context.Context, dimension int, input []string) ([][]float64, error) {
			embedding := make([]float64, dimension)
			for i := range embedding {
				embedding[i] = 0.1 + float64(i)*0.01
			}
			return [][]float64{embedding}, nil
		},
	}

	uc := usecase.New(usecase.WithRepository(repo), usecase.WithLLMClient(llmMock))
	resolver := NewResolver(repo, nil, uc)

	// Create a resolved ticket
	testTicket := &ticket.Ticket{
		ID:     types.NewTicketID(),
		Status: types.TicketStatusResolved,
		Metadata: ticket.Metadata{
			Title:       "Test Ticket",
			Description: "Test Description",
		},
	}
	gt.NoError(t, repo.PutTicket(context.Background(), *testTicket))

	runTest := func(tc struct {
		name           string
		ticketID       string
		conclusion     string
		reason         string
		expectedError  bool
		expectedStatus types.TicketStatus
	}) func(t *testing.T) {
		return func(t *testing.T) {
			ctx := context.Background()
			result, err := resolver.Mutation().UpdateTicketConclusion(ctx, tc.ticketID, tc.conclusion, tc.reason)

			if tc.expectedError {
				gt.Error(t, err)
				return
			}

			gt.NoError(t, err)
			gt.NotNil(t, result)
			gt.Equal(t, tc.conclusion, string(result.Conclusion))
			gt.Equal(t, tc.reason, result.Reason)
		}
	}

	t.Run("success case", runTest(struct {
		name           string
		ticketID       string
		conclusion     string
		reason         string
		expectedError  bool
		expectedStatus types.TicketStatus
	}{
		name:           "success",
		ticketID:       string(testTicket.ID),
		conclusion:     string(types.AlertConclusionTruePositive),
		reason:         "This is a test reason",
		expectedError:  false,
		expectedStatus: types.TicketStatusResolved,
	}))

	t.Run("invalid conclusion", runTest(struct {
		name           string
		ticketID       string
		conclusion     string
		reason         string
		expectedError  bool
		expectedStatus types.TicketStatus
	}{
		name:           "invalid conclusion",
		ticketID:       string(testTicket.ID),
		conclusion:     "invalid_conclusion",
		reason:         "This is a test reason",
		expectedError:  true,
		expectedStatus: types.TicketStatusResolved,
	}))

	t.Run("non-resolved ticket", runTest(struct {
		name           string
		ticketID       string
		conclusion     string
		reason         string
		expectedError  bool
		expectedStatus types.TicketStatus
	}{
		name: "non-resolved ticket",
		ticketID: func() string {
			openTicket := &ticket.Ticket{
				ID:     types.NewTicketID(),
				Status: types.TicketStatusOpen,
				Metadata: ticket.Metadata{
					Title:       "Open Ticket",
					Description: "Open Description",
				},
			}
			gt.NoError(t, repo.PutTicket(context.Background(), *openTicket))
			return string(openTicket.ID)
		}(),
		conclusion:     string(types.AlertConclusionTruePositive),
		reason:         "This is a test reason",
		expectedError:  true,
		expectedStatus: types.TicketStatusOpen,
	}))

	t.Run("ticket not found", runTest(struct {
		name           string
		ticketID       string
		conclusion     string
		reason         string
		expectedError  bool
		expectedStatus types.TicketStatus
	}{
		name:           "ticket not found",
		ticketID:       string(types.NewTicketID()),
		conclusion:     string(types.AlertConclusionTruePositive),
		reason:         "This is a test reason",
		expectedError:  true,
		expectedStatus: types.TicketStatusResolved,
	}))
}

func TestAlertsPaginated(t *testing.T) {
	repo := repository.NewMemory()
	resolver := NewResolver(repo, nil, nil)

	// Create a test ticket with multiple alerts
	ticketID := types.NewTicketID()
	alertIDs := make([]types.AlertID, 7) // Create 7 alerts for pagination testing

	// Create alerts
	for i := 0; i < 7; i++ {
		alertID := types.NewAlertID()
		alertIDs[i] = alertID
		testAlert := &alert.Alert{
			ID:       alertID,
			Schema:   types.AlertSchema("test"),
			TicketID: ticketID,
			Data:     map[string]interface{}{"test": "data"},
		}
		gt.NoError(t, repo.PutAlert(context.Background(), *testAlert))
	}

	// Create ticket with alerts
	testTicket := &ticket.Ticket{
		ID:       ticketID,
		Status:   types.TicketStatusOpen,
		AlertIDs: alertIDs,
		Metadata: ticket.Metadata{
			Title:       "Test Ticket",
			Description: "Test ticket with multiple alerts",
		},
	}
	gt.NoError(t, repo.PutTicket(context.Background(), *testTicket))

	ctx := context.Background()

	t.Run("default pagination", func(t *testing.T) {
		response, err := resolver.Ticket().AlertsPaginated(ctx, testTicket, nil, nil)
		gt.NoError(t, err)
		gt.V(t, response).NotEqual(nil)
		gt.V(t, response.TotalCount).Equal(7)
		gt.V(t, len(response.Alerts)).Equal(5) // Default limit is 5
	})

	t.Run("custom offset and limit", func(t *testing.T) {
		offset := 2
		limit := 3
		response, err := resolver.Ticket().AlertsPaginated(ctx, testTicket, &offset, &limit)
		gt.NoError(t, err)
		gt.V(t, response).NotEqual(nil)
		gt.V(t, response.TotalCount).Equal(7)
		gt.V(t, len(response.Alerts)).Equal(3)
	})

	t.Run("offset beyond range", func(t *testing.T) {
		offset := 10
		limit := 5
		response, err := resolver.Ticket().AlertsPaginated(ctx, testTicket, &offset, &limit)
		gt.NoError(t, err)
		gt.V(t, response).NotEqual(nil)
		gt.V(t, response.TotalCount).Equal(7)
		gt.V(t, len(response.Alerts)).Equal(0) // No alerts returned
	})

	t.Run("partial last page", func(t *testing.T) {
		offset := 5
		limit := 5
		response, err := resolver.Ticket().AlertsPaginated(ctx, testTicket, &offset, &limit)
		gt.NoError(t, err)
		gt.V(t, response).NotEqual(nil)
		gt.V(t, response.TotalCount).Equal(7)
		gt.V(t, len(response.Alerts)).Equal(2) // Only 2 alerts remaining
	})
}
