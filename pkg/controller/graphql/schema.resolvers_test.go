package graphql

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
)

func TestUpdateTicketConclusion(t *testing.T) {
	repo := repository.NewMemory()
	resolver := &Resolver{repo: repo}

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
