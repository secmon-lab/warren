package chat_test

import (
	"testing"

	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gt"
	adapter "github.com/secmon-lab/warren/pkg/adapter/storage"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/service/storage"
	"github.com/secmon-lab/warren/pkg/usecase/chat"
)

func TestLoadHistory_LatestFirst(t *testing.T) {
	ctx := t.Context()
	mockStorage := adapter.NewMock()
	repo := repository.NewMemory()
	svc := storage.New(mockStorage)
	ticketID := types.TicketID("ticket-latest")

	// Save latest.json
	latestHistory := &gollem.History{
		Version:  gollem.HistoryVersion,
		Messages: []gollem.Message{{Role: gollem.RoleUser}, {Role: gollem.RoleAssistant}},
	}
	gt.NoError(t, svc.PutLatestHistory(ctx, ticketID, latestHistory))

	// Also save history/{id}.json with different content
	historyID := types.HistoryID("hist-1")
	oldHistory := &gollem.History{
		Version:  gollem.HistoryVersion,
		Messages: []gollem.Message{{Role: gollem.RoleUser}},
	}
	gt.NoError(t, svc.PutHistory(ctx, ticketID, historyID, oldHistory))
	record := ticket.History{ID: historyID, TicketID: ticketID}
	gt.NoError(t, repo.PutHistory(ctx, ticketID, &record))

	// LoadHistory should return latest.json (2 messages) not history/{id}.json (1 message)
	loaded, err := chat.LoadHistory(ctx, repo, ticketID, svc)
	gt.NoError(t, err)
	gt.V(t, loaded).NotNil()
	gt.V(t, loaded.ToCount()).Equal(2)
}

func TestLoadHistory_FallbackToHistoryRecord(t *testing.T) {
	ctx := t.Context()
	mockStorage := adapter.NewMock()
	repo := repository.NewMemory()
	svc := storage.New(mockStorage)
	ticketID := types.TicketID("ticket-fallback")

	// Only save history/{id}.json — no latest.json
	historyID := types.HistoryID("hist-2")
	oldHistory := &gollem.History{
		Version:  gollem.HistoryVersion,
		Messages: []gollem.Message{{Role: gollem.RoleUser}},
	}
	gt.NoError(t, svc.PutHistory(ctx, ticketID, historyID, oldHistory))
	record := ticket.History{ID: historyID, TicketID: ticketID}
	gt.NoError(t, repo.PutHistory(ctx, ticketID, &record))

	// LoadHistory should fallback to history/{id}.json
	loaded, err := chat.LoadHistory(ctx, repo, ticketID, svc)
	gt.NoError(t, err)
	gt.V(t, loaded).NotNil()
	gt.V(t, loaded.ToCount()).Equal(1)
}

func TestLoadHistory_NoHistoryReturnsNil(t *testing.T) {
	ctx := t.Context()
	mockStorage := adapter.NewMock()
	repo := repository.NewMemory()
	svc := storage.New(mockStorage)
	ticketID := types.TicketID("ticket-none")

	// No latest.json, no history record
	loaded, err := chat.LoadHistory(ctx, repo, ticketID, svc)
	gt.NoError(t, err)
	gt.V(t, loaded).Nil()
}
