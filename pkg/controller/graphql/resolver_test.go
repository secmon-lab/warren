package graphql

import (
	"context"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
)

func TestTicketResolver(t *testing.T) {
	repo := repository.NewMemory()
	resolver := NewResolver(repo)
	ctx := context.Background()

	testTicket := &ticket.Ticket{
		ID:        types.TicketID("ticket-1"),
		Metadata:  ticket.Metadata{Title: "Test Ticket", Description: "desc"},
		Status:    types.TicketStatus("open"),
		AlertIDs:  []types.AlertID{"alert-1"},
		CreatedAt: time.Now(),
	}
	_ = repo.PutTicket(ctx, *testTicket)

	t.Run("GetTicket", func(t *testing.T) {
		got, err := resolver.Query().Ticket(ctx, string(testTicket.ID))
		gt.NoError(t, err)
		gt.Value(t, got.ID).Equal(testTicket.ID)
		gt.Value(t, got.Metadata.Title).Equal(testTicket.Metadata.Title)
	})

	t.Run("UpdateTicketStatus", func(t *testing.T) {
		newStatus := types.TicketStatus("closed")
		got, err := resolver.Mutation().UpdateTicketStatus(ctx, string(testTicket.ID), string(newStatus))
		gt.NoError(t, err)
		gt.Value(t, got.Status).Equal(newStatus)
	})
}

func TestAlertResolver(t *testing.T) {
	repo := repository.NewMemory()
	resolver := NewResolver(repo)
	ctx := context.Background()

	testAlert := &alert.Alert{
		ID:        types.AlertID("alert-1"),
		Metadata:  alert.Metadata{Title: "Test Alert", Description: "desc"},
		CreatedAt: time.Now(),
		Finding:   &alert.Finding{Severity: types.AlertSeverity("high")},
	}
	_ = repo.PutAlert(ctx, *testAlert)

	t.Run("GetAlert", func(t *testing.T) {
		got, err := resolver.Query().Alert(ctx, string(testAlert.ID))
		gt.NoError(t, err)
		gt.Value(t, got.ID).Equal(testAlert.ID)
		gt.Value(t, got.Metadata.Title).Equal(testAlert.Metadata.Title)
		gt.Value(t, got.Finding.Severity).Equal(types.AlertSeverity("high"))
	})
}

func TestCrossReference(t *testing.T) {
	repo := repository.NewMemory()
	resolver := NewResolver(repo)
	ctx := context.Background()

	ticketID := types.TicketID("ticket-1")
	alertID := types.AlertID("alert-1")

	testTicket := &ticket.Ticket{
		ID:        ticketID,
		Metadata:  ticket.Metadata{Title: "Test Ticket"},
		Status:    types.TicketStatus("open"),
		AlertIDs:  []types.AlertID{alertID},
		CreatedAt: time.Now(),
	}
	testAlert := &alert.Alert{
		ID:        alertID,
		Metadata:  alert.Metadata{Title: "Test Alert"},
		CreatedAt: time.Now(),
		TicketID:  ticketID,
		Finding:   &alert.Finding{Severity: types.AlertSeverity("high")},
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
