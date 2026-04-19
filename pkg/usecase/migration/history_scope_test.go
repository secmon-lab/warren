package migration_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gt"
	adapterStorage "github.com/secmon-lab/warren/pkg/adapter/storage"
	sessModel "github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/storage"
	"github.com/secmon-lab/warren/pkg/usecase/migration"
)

// historyOf returns a valid gollem.History whose Version matches the
// library's current expectation so service writes/reads round-trip.
func historyOf() *gollem.History {
	return &gollem.History{
		Version:  gollem.HistoryVersion,
		Messages: []gollem.Message{{Role: gollem.RoleUser}},
	}
}

func TestHistoryScope_CopiesLegacyTicketHistoryToSession(t *testing.T) {
	ctx := context.Background()
	client := adapterStorage.NewMemoryClient()
	svc := storage.New(client)

	tid := types.TicketID("tid_hs_1")
	sid := types.SessionID("sid_hs_1")
	// Seed legacy ticket-scoped history.
	gt.NoError(t, svc.PutLatestHistory(ctx, tid, historyOf()))

	sessions := []*sessModel.Session{{
		ID:          sid,
		Source:      sessModel.SessionSourceSlack,
		TicketIDPtr: &tid,
	}}
	listAll := func(_ context.Context) ([]*sessModel.Session, error) {
		return sessions, nil
	}

	job := migration.NewHistoryScopeJob(svc, listAll)
	res, err := job.Run(ctx, migration.Options{})
	gt.NoError(t, err)
	gt.V(t, res.Scanned).Equal(1)
	gt.V(t, res.Migrated).Equal(1)
	gt.V(t, res.Errors).Equal(0)

	// The session slot now has data.
	got, err := svc.GetSessionHistory(ctx, sid)
	gt.NoError(t, err)
	gt.V(t, got == nil).Equal(false)
}

func TestHistoryScope_SkipsSessionWithExistingHistory(t *testing.T) {
	ctx := context.Background()
	client := adapterStorage.NewMemoryClient()
	svc := storage.New(client)

	tid := types.TicketID("tid_hs_2")
	sid := types.SessionID("sid_hs_2")
	gt.NoError(t, svc.PutLatestHistory(ctx, tid, historyOf()))
	gt.NoError(t, svc.PutSessionHistory(ctx, sid, historyOf()))

	sessions := []*sessModel.Session{{
		ID: sid, Source: sessModel.SessionSourceSlack, TicketIDPtr: &tid,
	}}
	job := migration.NewHistoryScopeJob(svc, func(_ context.Context) ([]*sessModel.Session, error) {
		return sessions, nil
	})
	res, err := job.Run(ctx, migration.Options{})
	gt.NoError(t, err)
	gt.V(t, res.Migrated).Equal(0)
	gt.V(t, res.Skipped).Equal(1)
}

func TestHistoryScope_SkipsWebAndCLISessions(t *testing.T) {
	ctx := context.Background()
	client := adapterStorage.NewMemoryClient()
	svc := storage.New(client)

	tid := types.TicketID("tid_hs_3")
	gt.NoError(t, svc.PutLatestHistory(ctx, tid, historyOf()))

	sessions := []*sessModel.Session{
		{ID: "sid_web", Source: sessModel.SessionSourceWeb, TicketIDPtr: &tid},
		{ID: "sid_cli", Source: sessModel.SessionSourceCLI, TicketIDPtr: &tid},
	}
	job := migration.NewHistoryScopeJob(svc, func(_ context.Context) ([]*sessModel.Session, error) {
		return sessions, nil
	})
	res, err := job.Run(ctx, migration.Options{})
	gt.NoError(t, err)
	gt.V(t, res.Scanned).Equal(2)
	gt.V(t, res.Skipped).Equal(2)
	gt.V(t, res.Migrated).Equal(0)
}

func TestHistoryScope_SkipsSessionsWithoutTicket(t *testing.T) {
	ctx := context.Background()
	client := adapterStorage.NewMemoryClient()
	svc := storage.New(client)
	sessions := []*sessModel.Session{{ID: "sid_orphan", Source: sessModel.SessionSourceSlack}}
	job := migration.NewHistoryScopeJob(svc, func(_ context.Context) ([]*sessModel.Session, error) {
		return sessions, nil
	})
	res, err := job.Run(ctx, migration.Options{})
	gt.NoError(t, err)
	gt.V(t, res.Skipped).Equal(1)
	gt.V(t, res.Migrated).Equal(0)
}

func TestHistoryScope_DryRunDoesNotWrite(t *testing.T) {
	ctx := context.Background()
	client := adapterStorage.NewMemoryClient()
	svc := storage.New(client)
	tid := types.TicketID("tid_hs_dry")
	sid := types.SessionID("sid_hs_dry")
	gt.NoError(t, svc.PutLatestHistory(ctx, tid, historyOf()))

	sessions := []*sessModel.Session{{
		ID: sid, Source: sessModel.SessionSourceSlack, TicketIDPtr: &tid,
	}}
	job := migration.NewHistoryScopeJob(svc, func(_ context.Context) ([]*sessModel.Session, error) {
		return sessions, nil
	})
	res, err := job.Run(ctx, migration.Options{DryRun: true})
	gt.NoError(t, err)
	gt.V(t, res.Migrated).Equal(1)

	got, err := svc.GetSessionHistory(ctx, sid)
	gt.NoError(t, err)
	gt.V(t, got == nil).Equal(true)
}

func TestHistoryScope_RequiresDependencies(t *testing.T) {
	ctx := context.Background()
	job := migration.NewHistoryScopeJob(nil, nil)
	_, err := job.Run(ctx, migration.Options{})
	gt.Error(t, err)
}
