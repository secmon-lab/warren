package graphql

import (
	"context"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/mock"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/auth"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/usecase"
)

func TestTicketResolver(t *testing.T) {
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
	ctx := context.Background()

	now := time.Now()
	testTicket := &ticket.Ticket{
		ID:        types.TicketID("ticket-1"),
		Metadata:  ticket.Metadata{Title: "Test Ticket", Description: "desc"},
		Status:    types.TicketStatus("open"),
		AlertIDs:  []types.AlertID{"alert-1"},
		CreatedAt: now.Add(-time.Hour), // Created 1 hour ago
		UpdatedAt: now,                 // Updated at current time
	}
	_ = repo.PutTicket(ctx, *testTicket)

	t.Run("GetTicket", func(t *testing.T) {
		got, err := resolver.Query().Ticket(ctx, string(testTicket.ID))
		gt.NoError(t, err)
		gt.Value(t, got.ID).Equal(testTicket.ID)
		gt.Value(t, got.Metadata.Title).Equal(testTicket.Metadata.Title)
	})

	t.Run("TicketTimestampResolvers", func(t *testing.T) {
		got, err := resolver.Query().Ticket(ctx, string(testTicket.ID))
		gt.NoError(t, err)

		// Test CreatedAt resolver
		createdAtStr, err := resolver.Ticket().CreatedAt(ctx, got)
		gt.NoError(t, err)
		expectedCreatedAt := testTicket.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
		gt.Value(t, createdAtStr).Equal(expectedCreatedAt)

		// Test UpdatedAt resolver
		updatedAtStr, err := resolver.Ticket().UpdatedAt(ctx, got)
		gt.NoError(t, err)
		expectedUpdatedAt := testTicket.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")
		gt.Value(t, updatedAtStr).Equal(expectedUpdatedAt)

		// Verify that UpdatedAt is newer than CreatedAt
		gt.Value(t, testTicket.UpdatedAt.After(testTicket.CreatedAt)).Equal(true)
	})

	t.Run("GetTickets", func(t *testing.T) {
		status := "open"
		got, err := resolver.Query().Tickets(ctx, []string{status}, nil, nil)
		gt.NoError(t, err)
		gt.Array(t, got.Tickets).Length(1)
		gt.Value(t, got.Tickets[0].ID).Equal(testTicket.ID)
		gt.Value(t, got.TotalCount).Equal(1)
	})

	t.Run("GetTicketsWithPagination", func(t *testing.T) {
		// Create additional tickets
		tickets := []*ticket.Ticket{
			{
				ID:        types.TicketID("ticket-2"),
				Metadata:  ticket.Metadata{Title: "Test Ticket 2", Description: "desc"},
				Status:    types.TicketStatus("open"),
				CreatedAt: time.Now().Add(time.Hour),
				UpdatedAt: time.Now().Add(time.Hour + time.Minute),
			},
			{
				ID:        types.TicketID("ticket-3"),
				Metadata:  ticket.Metadata{Title: "Test Ticket 3", Description: "desc"},
				Status:    types.TicketStatus("resolved"),
				CreatedAt: time.Now().Add(2 * time.Hour),
				UpdatedAt: time.Now().Add(2*time.Hour + time.Minute),
			},
		}
		for _, t := range tickets {
			_ = repo.PutTicket(ctx, *t)
		}

		t.Run("with limit", func(t *testing.T) {
			limit := 2
			got, err := resolver.Query().Tickets(ctx, nil, nil, &limit)
			gt.NoError(t, err)
			gt.Array(t, got.Tickets).Length(2)
			gt.Value(t, got.TotalCount).Equal(3)
		})

		t.Run("with offset", func(t *testing.T) {
			offset := 1
			got, err := resolver.Query().Tickets(ctx, nil, &offset, nil)
			gt.NoError(t, err)
			gt.Array(t, got.Tickets).Length(2)
			gt.Value(t, got.TotalCount).Equal(3)
		})

		t.Run("with offset and limit", func(t *testing.T) {
			offset := 1
			limit := 1
			got, err := resolver.Query().Tickets(ctx, nil, &offset, &limit)
			gt.NoError(t, err)
			gt.Array(t, got.Tickets).Length(1)
			gt.Value(t, got.TotalCount).Equal(3)
		})

		t.Run("with multiple statuses", func(t *testing.T) {
			got, err := resolver.Query().Tickets(ctx, []string{"open", "pending"}, nil, nil)
			gt.NoError(t, err)
			gt.Array(t, got.Tickets).Length(2)
			gt.Value(t, got.TotalCount).Equal(2)
		})
	})

	t.Run("UpdateTicketStatus", func(t *testing.T) {
		newStatus := types.TicketStatus("resolved")
		got, err := resolver.Mutation().UpdateTicketStatus(ctx, string(testTicket.ID), string(newStatus))
		gt.NoError(t, err)
		gt.Value(t, got.Status).Equal(newStatus)
	})
}

func TestAlertResolver(t *testing.T) {
	repo := repository.NewMemory()
	resolver := NewResolver(repo, nil, nil)
	ctx := context.Background()

	testAlert := &alert.Alert{
		ID:        types.AlertID("alert-1"),
		Metadata:  alert.Metadata{Title: "Test Alert", Description: "desc"},
		CreatedAt: time.Now(),
	}
	_ = repo.PutAlert(ctx, *testAlert)

	t.Run("GetAlert", func(t *testing.T) {
		got, err := resolver.Query().Alert(ctx, string(testAlert.ID))
		gt.NoError(t, err)
		gt.Value(t, got.ID).Equal(testAlert.ID)
		gt.Value(t, got.Metadata.Title).Equal(testAlert.Metadata.Title)
	})
}

func TestCrossReference(t *testing.T) {
	repo := repository.NewMemory()
	resolver := NewResolver(repo, nil, nil)
	ctx := context.Background()

	ticketID := types.TicketID("ticket-1")
	alertID := types.AlertID("alert-1")

	now := time.Now()
	testTicket := &ticket.Ticket{
		ID:        ticketID,
		Metadata:  ticket.Metadata{Title: "Test Ticket"},
		Status:    types.TicketStatus("open"),
		AlertIDs:  []types.AlertID{alertID},
		CreatedAt: now,
		UpdatedAt: now,
	}
	testAlert := &alert.Alert{
		ID:        alertID,
		Metadata:  alert.Metadata{Title: "Test Alert"},
		CreatedAt: time.Now(),
		TicketID:  ticketID,
	}
	_ = repo.PutTicket(ctx, *testTicket)
	_ = repo.PutAlert(ctx, *testAlert)

	t.Run("Alert references Ticket", func(t *testing.T) {
		got, err := resolver.Alert().Ticket(ctx, testAlert)
		gt.NoError(t, err)
		gt.Value(t, got.ID).Equal(ticketID)
	})

	t.Run("Ticket references Alerts", func(t *testing.T) {
		got, err := resolver.Ticket().Alerts(ctx, testTicket)
		gt.NoError(t, err)
		gt.Array(t, got).Length(1)
		gt.Value(t, got[0].ID).Equal(alertID)
	})
}

func TestCreateTicket(t *testing.T) {
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

	uc := usecase.New(usecase.WithRepository(repo), usecase.WithLLMClient(llmMock))
	resolver := NewResolver(repo, nil, uc)

	// Create a context with authentication token
	token := &auth.Token{
		Sub:  "user123",
		Name: "Test User",
	}
	ctx := auth.ContextWithToken(context.Background(), token)

	t.Run("CreateTicket without Slack service", func(t *testing.T) {
		title := "Test Manual Ticket"
		description := "This is a test ticket created manually"

		// Test creating a ticket without test flag
		got, err := resolver.Mutation().CreateTicket(ctx, title, description, nil)
		gt.NoError(t, err)
		gt.Value(t, got.Metadata.Title).Equal(title)
		gt.Value(t, got.Metadata.Description).Equal(description)
		gt.Value(t, got.Assignee.ID).Equal("user123")
		gt.Value(t, got.Assignee.Name).Equal("Test User")
		gt.Value(t, got.IsTest).Equal(false)

		// Verify ticket was saved to repository
		savedTicket, err := repo.GetTicket(ctx, got.ID)
		gt.NoError(t, err)
		gt.Value(t, savedTicket.Metadata.Title).Equal(title)
	})

	t.Run("CreateTicket with test flag", func(t *testing.T) {
		title := "Test Manual Ticket with Flag"
		description := "This is a test ticket with test flag"
		isTest := true

		got, err := resolver.Mutation().CreateTicket(ctx, title, description, &isTest)
		gt.NoError(t, err)
		gt.Value(t, got.IsTest).Equal(true)
	})

	t.Run("CreateTicket with empty title should fail", func(t *testing.T) {
		_, err := resolver.Mutation().CreateTicket(ctx, "", "description", nil)
		gt.Error(t, err)
	})

	t.Run("CreateTicket without authentication should fail", func(t *testing.T) {
		ctxNoAuth := context.Background()
		_, err := resolver.Mutation().CreateTicket(ctxNoAuth, "title", "description", nil)
		gt.Error(t, err)
	})
}
