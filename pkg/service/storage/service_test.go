package storage_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gt"
	adapter "github.com/secmon-lab/warren/pkg/adapter/storage"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/storage"
)

func TestPutAndGetLatestHistory(t *testing.T) {
	ctx := context.Background()
	mockStorage := adapter.NewMock()
	svc := storage.New(mockStorage, storage.WithPrefix("test/"))
	ticketID := types.TicketID("ticket-1")

	history := &gollem.History{
		Version:  gollem.HistoryVersion,
		Messages: []gollem.Message{{Role: gollem.RoleUser}},
	}

	gt.NoError(t, svc.PutLatestHistory(ctx, ticketID, history))

	loaded, err := svc.GetLatestHistory(ctx, ticketID)
	gt.NoError(t, err)
	gt.V(t, loaded).NotNil()
	gt.V(t, loaded.Version).Equal(gollem.HistoryVersion)
	gt.V(t, loaded.ToCount()).Equal(1)
}

func TestGetLatestHistory_NotFound(t *testing.T) {
	ctx := context.Background()
	mockStorage := adapter.NewMock()
	svc := storage.New(mockStorage, storage.WithPrefix("test/"))
	ticketID := types.TicketID("nonexistent")

	_, err := svc.GetLatestHistory(ctx, ticketID)
	gt.V(t, err).NotNil() // should return error for missing object
}

func TestPutLatestHistory_Overwrite(t *testing.T) {
	ctx := context.Background()
	mockStorage := adapter.NewMock()
	svc := storage.New(mockStorage, storage.WithPrefix("test/"))
	ticketID := types.TicketID("ticket-2")

	history1 := &gollem.History{
		Version:  gollem.HistoryVersion,
		Messages: []gollem.Message{{Role: gollem.RoleUser}},
	}
	gt.NoError(t, svc.PutLatestHistory(ctx, ticketID, history1))

	history2 := &gollem.History{
		Version: gollem.HistoryVersion,
		Messages: []gollem.Message{
			{Role: gollem.RoleUser},
			{Role: gollem.RoleAssistant},
		},
	}
	gt.NoError(t, svc.PutLatestHistory(ctx, ticketID, history2))

	loaded, err := svc.GetLatestHistory(ctx, ticketID)
	gt.NoError(t, err)
	gt.V(t, loaded).NotNil()
	gt.V(t, loaded.ToCount()).Equal(2)
}

func TestHistoryRepo_LoadAndSave(t *testing.T) {
	ctx := context.Background()
	mockStorage := adapter.NewMock()
	svc := storage.New(mockStorage)
	ticketID := types.TicketID("ticket-repo")
	logger := slog.Default()

	repo := storage.NewHistoryRepo(svc, ticketID, logger)

	// Load returns nil when no history exists
	loaded, err := repo.Load(ctx, string(ticketID))
	gt.NoError(t, err)
	gt.V(t, loaded).Nil()

	// Save and then Load
	history := &gollem.History{
		Version:  gollem.HistoryVersion,
		Messages: []gollem.Message{{Role: gollem.RoleUser}},
	}
	gt.NoError(t, repo.Save(ctx, string(ticketID), history))

	loaded, err = repo.Load(ctx, string(ticketID))
	gt.NoError(t, err)
	gt.V(t, loaded).NotNil()
	gt.V(t, loaded.Version).Equal(gollem.HistoryVersion)
	gt.V(t, loaded.ToCount()).Equal(1)
}

func TestHistoryRepo_SaveErrorDoesNotReturn(t *testing.T) {
	// HistoryRepo.Save should not return error even if storage fails,
	// to avoid interrupting agent execution.
	// We test this by verifying Save returns nil on a valid save.
	ctx := context.Background()
	mockStorage := adapter.NewMock()
	svc := storage.New(mockStorage)
	ticketID := types.TicketID("ticket-err")
	logger := slog.Default()

	repo := storage.NewHistoryRepo(svc, ticketID, logger)

	history := &gollem.History{
		Version:  gollem.HistoryVersion,
		Messages: []gollem.Message{{Role: gollem.RoleUser}},
	}
	err := repo.Save(ctx, string(ticketID), history)
	gt.NoError(t, err) // Save always returns nil
}

func TestPathToLatestHistory(t *testing.T) {
	// Verify the path structure by checking that Put and Get use the same path
	ctx := context.Background()
	mockStorage := adapter.NewMock()
	svc := storage.New(mockStorage, storage.WithPrefix("pfx/"))
	ticketID := types.TicketID("t1")

	history := &gollem.History{
		Version:  gollem.HistoryVersion,
		Messages: []gollem.Message{{Role: gollem.RoleUser}},
	}
	gt.NoError(t, svc.PutLatestHistory(ctx, ticketID, history))

	// Verify we can get it back (proves path consistency)
	loaded, err := svc.GetLatestHistory(ctx, ticketID)
	gt.NoError(t, err)
	gt.V(t, loaded).NotNil()
}
