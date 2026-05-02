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
	slackModel "github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	sessService "github.com/secmon-lab/warren/pkg/service/session"
	"github.com/secmon-lab/warren/pkg/service/storage"
	"github.com/secmon-lab/warren/pkg/usecase/migration"
)

// inMemoryCommentStore is a test-only LegacyCommentStore. Inlined here
// because the only other test that needed it (cleanup_legacy_test.go)
// was dropped on main when the destructive cleanup-legacy job was
// removed; the bundle test still needs to seed legacy ticket.Comment
// rows for the comment-to-message step.
type inMemoryCommentStore struct {
	mu       sync.Mutex
	tickets  []*ticket.Ticket
	comments map[types.TicketID][]ticket.Comment
}

func newInMemoryCommentStore() *inMemoryCommentStore {
	return &inMemoryCommentStore{comments: map[types.TicketID][]ticket.Comment{}}
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

func (s *inMemoryCommentStore) GetTicketComments(_ context.Context, id types.TicketID) ([]ticket.Comment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]ticket.Comment, len(s.comments[id]))
	copy(out, s.comments[id])
	return out, nil
}

// TestV0_16_0Bundle_EndToEnd seeds a Memory repository + Memory storage
// with v0.15-shaped data (a Ticket with a Slack thread, a legacy
// UUID-id Session, two ticket.Comment rows, and a latest.json gollem
// history) and verifies that running the v0.16.0 bundle:
//
//   - executes the five non-destructive jobs in the documented order,
//   - leaves the legacy data untouched (rollback safety), and
//   - materializes the new schema: Source-tagged Sessions, canonical
//     slack_<hash> Session, mirrored user Messages, a Turn with the
//     Messages stamped, and a Session-scoped history file.
func TestV0_16_0Bundle_EndToEnd(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	resolver := sessService.NewResolver(repo)
	storageClient := adapterStorage.NewMemoryClient()
	storageSvc := storage.New(storageClient)

	tid := types.TicketID("tid_bundle")
	thread := slackModel.Thread{ChannelID: "C9", ThreadID: "1700000000.000900"}
	t0 := time.Unix(1700000000, 0).UTC()

	// Seed: v0.15-shaped Ticket with Slack thread.
	gt.NoError(t, repo.PutTicket(ctx, ticket.Ticket{ID: tid, SlackThread: &thread}))

	// Seed: a legacy UUID Session bound to the same Ticket. Source
	// is intentionally empty to simulate pre-redesign data that the
	// session-source-backfill job needs to repair.
	legacySID := types.SessionID("legacy_uuid_session")
	gt.NoError(t, repo.PutSession(ctx, &sessModel.Session{
		ID:           legacySID,
		TicketID:     tid,
		CreatedAt:    t0,
		LastActiveAt: t0.Add(2 * time.Minute),
	}))

	// Seed: legacy gollem.History at the ticket-scoped path.
	legacyHistory := &gollem.History{
		Version:  gollem.HistoryVersion,
		Messages: []gollem.Message{{Role: gollem.RoleUser}},
	}
	gt.NoError(t, storageSvc.PutLatestHistory(ctx, tid, legacyHistory))

	// Seed: legacy ticket.Comments. The bundle's comment-to-message
	// step rewrites these into session.Message rows.
	commentStore := newInMemoryCommentStore()
	commentStore.addTicket(&ticket.Ticket{ID: tid, SlackThread: &thread})
	commentStore.addComment(ticket.Comment{
		ID: types.NewCommentID(), TicketID: tid,
		Comment:   "first thought",
		CreatedAt: t0,
		User:      &slackModel.User{ID: "U_alice", Name: "alice"},
	})
	commentStore.addComment(ticket.Comment{
		ID: types.NewCommentID(), TicketID: tid,
		Comment:   "second thought",
		CreatedAt: t0.Add(time.Minute),
		User:      &slackModel.User{ID: "U_bob", Name: "bob"},
	})

	// Wire the five non-destructive jobs. Order matches
	// runChatSessionRedesignBundle in pkg/cli/migrate.go and the
	// documented Step 3 sequence.
	bundle := []migration.Job{
		migration.NewSessionSourceBackfillJob(repo, sessionsForEach(repo.AllSessions())),
		migration.NewSessionConsolidateJob(repo, resolver),
		migration.NewCommentToMessageJob(commentStore, resolver, repo, repo),
		migration.NewTurnSynthesisJob(repo, func(ctx context.Context, h func(*sessModel.Session) error) error {
			// Re-snapshot at execution time so any Sessions created by
			// earlier bundle steps (consolidate, comment-to-message)
			// are picked up.
			for _, s := range repo.AllSessions() {
				if err := h(s); err != nil {
					return err
				}
			}
			return nil
		}),
		migration.NewHistoryScopeJob(storageSvc, func(ctx context.Context, h func(*sessModel.Session) error) error {
			for _, s := range repo.AllSessions() {
				if err := h(s); err != nil {
					return err
				}
			}
			return nil
		}),
	}

	results, err := migration.RunBundle(ctx, migration.Options{}, bundle...)
	gt.NoError(t, err)
	gt.V(t, len(results)).Equal(5)

	// Order check: the bundle must run jobs in the documented sequence.
	wantOrder := []string{
		"session-source-backfill",
		"session-consolidate",
		"comment-to-message",
		"turn-synthesis",
		"history-scope",
	}
	for i, want := range wantOrder {
		gt.V(t, results[i].JobName).Equal(want)
	}

	// --- Rollback safety: legacy data is untouched. ---

	// Legacy UUID Session row still exists.
	legacy, err := repo.GetSession(ctx, legacySID)
	gt.NoError(t, err)
	gt.V(t, legacy).NotNil()
	// session-source-backfill stamped Source=slack onto the legacy row,
	// but did NOT remove its legacy fields.
	gt.V(t, legacy.Source).Equal(sessModel.SessionSourceSlack)
	gt.V(t, legacy.TicketIDPtr != nil && *legacy.TicketIDPtr == tid).Equal(true)

	// Legacy ticket-scoped history is still readable.
	stillThere, err := storageSvc.GetLatestHistory(ctx, tid)
	gt.NoError(t, err)
	gt.V(t, stillThere).NotNil()

	// Legacy ticket_comments are still present (cleanup-legacy not run).
	remainingComments, err := commentStore.GetTicketComments(ctx, tid)
	gt.NoError(t, err)
	gt.V(t, len(remainingComments)).Equal(2)

	// --- Forward correctness: new schema is materialized. ---

	// session-consolidate created the canonical slack_<hash> Session.
	canonical, ok, err := resolver.LookupSlackSession(ctx, &tid, thread)
	gt.NoError(t, err)
	gt.V(t, ok).Equal(true)
	gt.V(t, canonical).NotNil()
	gt.V(t, canonical.Source).Equal(sessModel.SessionSourceSlack)
	gt.V(t, canonical.ID == legacySID).Equal(false)

	// comment-to-message wrote both Comments as user Messages on the
	// canonical Session.
	msgs, err := repo.GetSessionMessages(ctx, canonical.ID)
	gt.NoError(t, err)
	byContent := map[string]*sessModel.Message{}
	for _, m := range msgs {
		byContent[m.Content] = m
	}
	gt.V(t, byContent["first thought"] != nil).Equal(true)
	gt.V(t, byContent["first thought"].Type).Equal(sessModel.MessageTypeUser)
	gt.V(t, byContent["first thought"].Author != nil &&
		byContent["first thought"].Author.DisplayName == "alice").Equal(true)
	gt.V(t, byContent["second thought"] != nil).Equal(true)
	gt.V(t, byContent["second thought"].Author != nil &&
		byContent["second thought"].Author.DisplayName == "bob").Equal(true)

	// turn-synthesis created exactly one completed Turn for the
	// canonical Session and stamped TurnID onto every Message it
	// already had.
	turns, err := repo.GetTurnsBySession(ctx, canonical.ID)
	gt.NoError(t, err)
	gt.V(t, len(turns)).Equal(1)
	gt.V(t, turns[0].Status).Equal(sessModel.TurnStatusCompleted)
	for _, m := range msgs {
		gt.V(t, m.TurnID != nil && *m.TurnID == turns[0].ID).Equal(true)
	}

	// history-scope copied the legacy history into the Session-scoped
	// destination.
	sessHistory, err := storageSvc.GetSessionHistory(ctx, canonical.ID)
	gt.NoError(t, err)
	gt.V(t, sessHistory).NotNil()
	gt.V(t, sessHistory.Version).Equal(gollem.HistoryVersion)

	// --- Idempotence: re-running the bundle does not duplicate. ---
	results2, err := migration.RunBundle(ctx, migration.Options{}, bundle...)
	gt.NoError(t, err)
	gt.V(t, len(results2)).Equal(5)

	msgs2, err := repo.GetSessionMessages(ctx, canonical.ID)
	gt.NoError(t, err)
	gt.V(t, len(msgs2)).Equal(len(msgs))

	turns2, err := repo.GetTurnsBySession(ctx, canonical.ID)
	gt.NoError(t, err)
	gt.V(t, len(turns2)).Equal(1)
}

// TestV0_16_0Bundle_DryRunWritesNothing confirms the bundle honors
// DryRun across every step: no Sessions, Messages, Turns, or copied
// history files are produced.
func TestV0_16_0Bundle_DryRunWritesNothing(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	resolver := sessService.NewResolver(repo)
	storageClient := adapterStorage.NewMemoryClient()
	storageSvc := storage.New(storageClient)

	tid := types.TicketID("tid_bundle_dry")
	thread := slackModel.Thread{ChannelID: "C9", ThreadID: "1700000000.001000"}
	t0 := time.Unix(1700000000, 0).UTC()

	gt.NoError(t, repo.PutTicket(ctx, ticket.Ticket{ID: tid, SlackThread: &thread}))
	legacySID := types.SessionID("legacy_uuid_dry")
	gt.NoError(t, repo.PutSession(ctx, &sessModel.Session{
		ID: legacySID, TicketID: tid, CreatedAt: t0,
	}))
	gt.NoError(t, storageSvc.PutLatestHistory(ctx, tid, &gollem.History{
		Version: gollem.HistoryVersion, Messages: []gollem.Message{{Role: gollem.RoleUser}},
	}))

	commentStore := newInMemoryCommentStore()
	commentStore.addTicket(&ticket.Ticket{ID: tid, SlackThread: &thread})
	commentStore.addComment(ticket.Comment{
		ID: types.NewCommentID(), TicketID: tid, Comment: "x", CreatedAt: t0,
		User: &slackModel.User{ID: "U_x", Name: "x"},
	})

	bundle := []migration.Job{
		migration.NewSessionSourceBackfillJob(repo, sessionsForEach(repo.AllSessions())),
		migration.NewSessionConsolidateJob(repo, resolver),
		migration.NewCommentToMessageJob(commentStore, resolver, repo, repo),
		migration.NewTurnSynthesisJob(repo, func(ctx context.Context, h func(*sessModel.Session) error) error {
			for _, s := range repo.AllSessions() {
				if err := h(s); err != nil {
					return err
				}
			}
			return nil
		}),
		migration.NewHistoryScopeJob(storageSvc, func(ctx context.Context, h func(*sessModel.Session) error) error {
			for _, s := range repo.AllSessions() {
				if err := h(s); err != nil {
					return err
				}
			}
			return nil
		}),
	}

	_, err := migration.RunBundle(ctx, migration.Options{DryRun: true}, bundle...)
	gt.NoError(t, err)

	// Canonical Session must NOT have been created.
	_, ok, err := resolver.LookupSlackSession(ctx, &tid, thread)
	gt.NoError(t, err)
	gt.V(t, ok).Equal(false)

	// Legacy Session row's Source is left untouched (DryRun
	// session-source-backfill does not write).
	legacy, err := repo.GetSession(ctx, legacySID)
	gt.NoError(t, err)
	gt.V(t, legacy.Source).Equal(sessModel.SessionSource(""))

	// No Session-scoped history written.
	got, err := storageSvc.GetSessionHistory(ctx, legacySID)
	gt.V(t, err != nil || got == nil).Equal(true)
}
