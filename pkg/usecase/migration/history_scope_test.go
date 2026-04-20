package migration_test

import (
	"context"
	"io"
	"sync"
	"testing"

	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gt"
	adapterStorage "github.com/secmon-lab/warren/pkg/adapter/storage"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	sessModel "github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/storage"
	"github.com/secmon-lab/warren/pkg/usecase/migration"
)

// copySpyClient wraps a MemoryClient and counts CopyObject /
// PutObject / GetObject calls so tests can assert that history-scope
// uses the server-side copy path rather than downloading and
// re-uploading the payload.
type copySpyClient struct {
	mu     sync.Mutex
	inner  *adapterStorage.MemoryClient
	puts   int
	gets   int
	copies int
}

var _ interfaces.StorageClient = (*copySpyClient)(nil)

func newCopySpyClient() *copySpyClient {
	return &copySpyClient{inner: adapterStorage.NewMemoryClient()}
}

func (c *copySpyClient) PutObject(ctx context.Context, object string) io.WriteCloser {
	c.mu.Lock()
	c.puts++
	c.mu.Unlock()
	return c.inner.PutObject(ctx, object)
}

func (c *copySpyClient) GetObject(ctx context.Context, object string) (io.ReadCloser, error) {
	c.mu.Lock()
	c.gets++
	c.mu.Unlock()
	return c.inner.GetObject(ctx, object)
}

func (c *copySpyClient) DeleteObject(ctx context.Context, object string) error {
	return c.inner.DeleteObject(ctx, object)
}

func (c *copySpyClient) CopyObject(ctx context.Context, src, dst string) error {
	c.mu.Lock()
	c.copies++
	c.mu.Unlock()
	return c.inner.CopyObject(ctx, src, dst)
}

func (c *copySpyClient) Close(ctx context.Context) { c.inner.Close(ctx) }

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
	job := migration.NewHistoryScopeJob(svc, sessionsForEach(sessions))
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
	job := migration.NewHistoryScopeJob(svc, sessionsForEach(sessions))
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
	job := migration.NewHistoryScopeJob(svc, sessionsForEach(sessions))
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
	job := migration.NewHistoryScopeJob(svc, sessionsForEach(sessions))
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
	job := migration.NewHistoryScopeJob(svc, sessionsForEach(sessions))
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

// TestHistoryScope_UsesServerSideCopy guards against regression to the
// download+reupload pattern. GCS Copy runs entirely on Google's side,
// so the migration binary's egress bandwidth stays at zero even for
// production datasets with tens of thousands of Sessions. If a future
// refactor accidentally reintroduces Get/Put on the history payload,
// this assertion will fail.
func TestHistoryScope_UsesServerSideCopy(t *testing.T) {
	ctx := context.Background()
	spy := newCopySpyClient()
	svc := storage.New(spy)

	tid := types.TicketID("tid_hs_spy")
	sid := types.SessionID("sid_hs_spy")
	gt.NoError(t, svc.PutLatestHistory(ctx, tid, historyOf()))

	// Reset counters after the seed write so we only measure the job.
	spy.mu.Lock()
	spy.puts = 0
	spy.gets = 0
	spy.copies = 0
	spy.mu.Unlock()

	sessions := []*sessModel.Session{{
		ID: sid, Source: sessModel.SessionSourceSlack, TicketIDPtr: &tid,
	}}
	job := migration.NewHistoryScopeJob(svc, sessionsForEach(sessions))
	res, err := job.Run(ctx, migration.Options{})
	gt.NoError(t, err)
	gt.V(t, res.Migrated).Equal(1)

	spy.mu.Lock()
	defer spy.mu.Unlock()
	gt.V(t, spy.copies).Equal(1)
	// Only the HasSessionHistory probe reads; payload is never downloaded.
	gt.V(t, spy.gets).Equal(1)
	// No PutObject at all on the payload path.
	gt.V(t, spy.puts).Equal(0)
}
