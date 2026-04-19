package migration_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gt"
	adapterStorage "github.com/secmon-lab/warren/pkg/adapter/storage"
	sessModel "github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/storage"
	"github.com/secmon-lab/warren/pkg/usecase/migration"
)

// inMemoryCommentStore is a test-only LegacyCommentStore implementation.
// It lives here (not in repository/memory) because the migration
// boundary intentionally keeps Comment CRUD out of the main Repository
// interface — only the migration package ever touches legacy comments,
// so the test helper is colocated with the migration tests.
type inMemoryCommentStore struct {
	mu       sync.Mutex
	tickets  []*ticket.Ticket
	comments map[types.TicketID][]ticket.Comment
}

func newInMemoryCommentStore() *inMemoryCommentStore {
	return &inMemoryCommentStore{
		comments: map[types.TicketID][]ticket.Comment{},
	}
}

func (s *inMemoryCommentStore) addTicket(t *ticket.Ticket) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tickets = append(s.tickets, t)
}

func (s *inMemoryCommentStore) addComment(c ticket.Comment) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.comments[c.TicketID] = append(s.comments[c.TicketID], c)
}

func (s *inMemoryCommentStore) ListTicketsWithComments(_ context.Context) ([]*ticket.Ticket, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*ticket.Ticket, len(s.tickets))
	copy(out, s.tickets)
	return out, nil
}

func (s *inMemoryCommentStore) GetTicketComments(_ context.Context, ticketID types.TicketID) ([]ticket.Comment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]ticket.Comment, len(s.comments[ticketID]))
	copy(out, s.comments[ticketID])
	return out, nil
}

func (s *inMemoryCommentStore) DeleteTicketComment(_ context.Context, ticketID types.TicketID, commentID types.CommentID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := s.comments[ticketID]
	out := list[:0]
	for _, c := range list {
		if c.ID != commentID {
			out = append(out, c)
		}
	}
	s.comments[ticketID] = out
	return nil
}

func TestCleanupLegacy_DeletesCommentsAndHistory(t *testing.T) {
	ctx := context.Background()
	store := newInMemoryCommentStore()
	client := adapterStorage.NewMemoryClient()
	svc := storage.New(client)

	tid := types.TicketID("tid_cl_1")
	tk := &ticket.Ticket{ID: tid}
	store.addTicket(tk)
	store.addComment(ticket.Comment{ID: types.NewCommentID(), TicketID: tid, Comment: "a", CreatedAt: time.Now()})
	store.addComment(ticket.Comment{ID: types.NewCommentID(), TicketID: tid, Comment: "b", CreatedAt: time.Now()})
	gt.NoError(t, svc.PutLatestHistory(ctx, tid, &gollem.History{
		Version:  gollem.HistoryVersion,
		Messages: []gollem.Message{{Role: gollem.RoleUser}},
	}))

	job := migration.NewCleanupLegacyJob(store, svc,
		func(_ context.Context) ([]*sessModel.Session, error) { return nil, nil },
	)
	res, err := job.Run(ctx, migration.Options{})
	gt.NoError(t, err)
	// 2 comments + 1 latest.json
	gt.V(t, res.Migrated).Equal(3)
	gt.V(t, res.Errors).Equal(0)

	remaining, err := store.GetTicketComments(ctx, tid)
	gt.NoError(t, err)
	gt.V(t, len(remaining)).Equal(0)

	// After deletion the storage object is gone; the service surfaces
	// this as an error (not nil+nil), which is acceptable cleanup semantics.
	_, err = svc.GetLatestHistory(ctx, tid)
	gt.Error(t, err)
}

func TestCleanupLegacy_DryRunNoDeletes(t *testing.T) {
	ctx := context.Background()
	store := newInMemoryCommentStore()
	client := adapterStorage.NewMemoryClient()
	svc := storage.New(client)

	tid := types.TicketID("tid_cl_dry")
	tk := &ticket.Ticket{ID: tid}
	store.addTicket(tk)
	store.addComment(ticket.Comment{ID: types.NewCommentID(), TicketID: tid, Comment: "x", CreatedAt: time.Now()})
	gt.NoError(t, svc.PutLatestHistory(ctx, tid, &gollem.History{
		Version:  gollem.HistoryVersion,
		Messages: []gollem.Message{{Role: gollem.RoleUser}},
	}))

	job := migration.NewCleanupLegacyJob(store, svc,
		func(_ context.Context) ([]*sessModel.Session, error) { return nil, nil },
	)
	res, err := job.Run(ctx, migration.Options{DryRun: true})
	gt.NoError(t, err)
	gt.V(t, res.Migrated).Equal(2)

	remaining, err := store.GetTicketComments(ctx, tid)
	gt.NoError(t, err)
	gt.V(t, len(remaining)).Equal(1)
}

func TestCleanupLegacy_WithoutStorageSkipsHistory(t *testing.T) {
	ctx := context.Background()
	store := newInMemoryCommentStore()
	tid := types.TicketID("tid_cl_2")
	tk := &ticket.Ticket{ID: tid}
	store.addTicket(tk)
	store.addComment(ticket.Comment{
		ID: types.NewCommentID(), TicketID: tid, Comment: "c", CreatedAt: time.Now(),
	})

	job := migration.NewCleanupLegacyJob(store, nil,
		func(_ context.Context) ([]*sessModel.Session, error) { return nil, nil },
	)
	res, err := job.Run(ctx, migration.Options{})
	gt.NoError(t, err)
	gt.V(t, res.Migrated).Equal(1)
}

func TestCleanupLegacy_RequiresDependencies(t *testing.T) {
	ctx := context.Background()
	job := migration.NewCleanupLegacyJob(nil, nil, nil)
	_, err := job.Run(ctx, migration.Options{})
	gt.Error(t, err)
}
