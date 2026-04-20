package migration_test

import (
	"context"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	sessModel "github.com/secmon-lab/warren/pkg/domain/model/session"
	slackModel "github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	sessService "github.com/secmon-lab/warren/pkg/service/session"
	"github.com/secmon-lab/warren/pkg/usecase/migration"
)

// stubCommentSource serves a pre-populated list of tickets + per-ticket
// comments to the migration job.
type stubCommentSource struct {
	tickets  []*ticket.Ticket
	comments map[types.TicketID][]ticket.Comment
}

func (s *stubCommentSource) ListTicketsWithComments(_ context.Context) ([]*ticket.Ticket, error) {
	return s.tickets, nil
}

func (s *stubCommentSource) GetTicketComments(_ context.Context, id types.TicketID) ([]ticket.Comment, error) {
	return s.comments[id], nil
}

func TestCommentToMessage_MigratesCommentsAsUserMessages(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	resolver := sessService.NewResolver(repo)

	tid := types.TicketID("tid_ccm")
	thread := slackModel.Thread{ChannelID: "C1", ThreadID: "1700000000.000100"}
	t0 := time.Unix(1700000000, 0).UTC()

	tk := &ticket.Ticket{
		ID:          tid,
		SlackThread: &thread,
	}
	comments := []ticket.Comment{
		{
			ID:        types.NewCommentID(),
			TicketID:  tid,
			Comment:   "first thought",
			CreatedAt: t0,
			User:      &slackModel.User{ID: "U_alice", Name: "alice"},
		},
		{
			ID:        types.NewCommentID(),
			TicketID:  tid,
			Comment:   "second thought",
			CreatedAt: t0.Add(1 * time.Minute),
			User:      &slackModel.User{ID: "U_bob", Name: "bob"},
		},
	}
	src := &stubCommentSource{
		tickets:  []*ticket.Ticket{tk},
		comments: map[types.TicketID][]ticket.Comment{tid: comments},
	}

	job := migration.NewCommentToMessageJob(src, resolver, repo, repo)
	res, err := job.Run(ctx, migration.Options{})
	gt.NoError(t, err)
	gt.V(t, res.Scanned).Equal(2)
	gt.V(t, res.Migrated).Equal(2)
	gt.V(t, res.Errors).Equal(0)

	// Locate the resolved session.
	sess, _, err := resolver.ResolveSlackSession(ctx, &tid, thread, "")
	gt.NoError(t, err)
	msgs, err := repo.GetSessionMessages(ctx, sess.ID)
	gt.NoError(t, err)
	gt.V(t, len(msgs)).Equal(2)
	byContent := map[string]*sessModel.Message{}
	for _, m := range msgs {
		byContent[m.Content] = m
	}
	gt.V(t, byContent["first thought"] != nil).Equal(true)
	gt.V(t, byContent["first thought"].Type).Equal(sessModel.MessageTypeUser)
	gt.V(t, byContent["first thought"].Author != nil).Equal(true)
	gt.V(t, byContent["first thought"].Author.DisplayName).Equal("alice")
	gt.V(t, byContent["second thought"].Author.DisplayName).Equal("bob")

	// Rerun → idempotent, no new messages.
	res2, err := job.Run(ctx, migration.Options{})
	gt.NoError(t, err)
	gt.V(t, res2.Scanned).Equal(2)
	gt.V(t, res2.Skipped).Equal(2)
	gt.V(t, res2.Migrated).Equal(0)
}

func TestCommentToMessage_SkipsTicketWithoutThread(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	resolver := sessService.NewResolver(repo)

	tid := types.TicketID("tid_no_thread")
	tk := &ticket.Ticket{ID: tid}
	comments := []ticket.Comment{{
		ID: types.NewCommentID(), TicketID: tid, Comment: "orphan", CreatedAt: time.Now().UTC(),
	}}
	src := &stubCommentSource{
		tickets:  []*ticket.Ticket{tk},
		comments: map[types.TicketID][]ticket.Comment{tid: comments},
	}

	job := migration.NewCommentToMessageJob(src, resolver, repo, repo)
	res, err := job.Run(ctx, migration.Options{})
	gt.NoError(t, err)
	gt.V(t, res.Scanned).Equal(1)
	gt.V(t, res.Skipped).Equal(1)
	gt.V(t, res.Migrated).Equal(0)
}

func TestCommentToMessage_DryRunNoWrites(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	resolver := sessService.NewResolver(repo)

	tid := types.TicketID("tid_dry")
	thread := slackModel.Thread{ChannelID: "C1", ThreadID: "1700000000.000200"}
	tk := &ticket.Ticket{ID: tid, SlackThread: &thread}
	comments := []ticket.Comment{{
		ID: types.NewCommentID(), TicketID: tid, Comment: "dry", CreatedAt: time.Now().UTC(),
	}}
	src := &stubCommentSource{
		tickets:  []*ticket.Ticket{tk},
		comments: map[types.TicketID][]ticket.Comment{tid: comments},
	}

	job := migration.NewCommentToMessageJob(src, resolver, repo, repo)
	res, err := job.Run(ctx, migration.Options{DryRun: true})
	gt.NoError(t, err)
	gt.V(t, res.Migrated).Equal(1)

	sess, _, err := resolver.ResolveSlackSession(ctx, &tid, thread, "")
	gt.NoError(t, err)
	msgs, err := repo.GetSessionMessages(ctx, sess.ID)
	gt.NoError(t, err)
	gt.V(t, len(msgs)).Equal(0)
}

func TestCommentToMessage_RequiresDependencies(t *testing.T) {
	ctx := context.Background()
	job := migration.NewCommentToMessageJob(nil, nil, nil, nil)
	_, err := job.Run(ctx, migration.Options{})
	gt.Error(t, err)
}

// TestCommentToMessage_DryRunDoesNotCreateSession verifies that a
// dry-run does not leave behind a brand-new Session document when the
// thread has never been chatted on before. Regression guard for the
// "dry-run must not mutate state" spec requirement.
func TestCommentToMessage_DryRunDoesNotCreateSession(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	resolver := sessService.NewResolver(repo)

	tid := types.TicketID("tid_dry_create")
	thread := slackModel.Thread{ChannelID: "C1", ThreadID: "1700000000.000300"}
	tk := &ticket.Ticket{ID: tid, SlackThread: &thread}
	comments := []ticket.Comment{{
		ID: types.NewCommentID(), TicketID: tid, Comment: "c", CreatedAt: time.Now().UTC(),
	}}
	src := &stubCommentSource{
		tickets:  []*ticket.Ticket{tk},
		comments: map[types.TicketID][]ticket.Comment{tid: comments},
	}

	job := migration.NewCommentToMessageJob(src, resolver, repo, repo)
	_, err := job.Run(ctx, migration.Options{DryRun: true})
	gt.NoError(t, err)

	// Look up via the resolver: since dry-run ran through
	// LookupSlackSession (read-only), there should be no Session yet.
	found, ok, err := resolver.LookupSlackSession(ctx, &tid, thread)
	gt.NoError(t, err)
	gt.V(t, ok).Equal(false)
	gt.V(t, found == nil).Equal(true)
}

// TestCommentToMessage_AuthorAwareIdempotence: two comments with the
// same second-resolution timestamp and same content from different
// users must NOT collapse into one migrated Message.
func TestCommentToMessage_AuthorAwareIdempotence(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	resolver := sessService.NewResolver(repo)

	tid := types.TicketID("tid_author_key")
	thread := slackModel.Thread{ChannelID: "C1", ThreadID: "1700000000.000400"}
	tk := &ticket.Ticket{ID: tid, SlackThread: &thread}
	at := time.Unix(1700000000, 0).UTC()

	sameText := "+1 to this plan"
	comments := []ticket.Comment{
		{ID: types.NewCommentID(), TicketID: tid, Comment: sameText, CreatedAt: at, User: &slackModel.User{ID: "U_alice", Name: "alice"}},
		{ID: types.NewCommentID(), TicketID: tid, Comment: sameText, CreatedAt: at, User: &slackModel.User{ID: "U_bob", Name: "bob"}},
	}
	src := &stubCommentSource{
		tickets:  []*ticket.Ticket{tk},
		comments: map[types.TicketID][]ticket.Comment{tid: comments},
	}

	job := migration.NewCommentToMessageJob(src, resolver, repo, repo)
	res, err := job.Run(ctx, migration.Options{})
	gt.NoError(t, err)
	gt.V(t, res.Migrated).Equal(2)

	sess, _, err := resolver.ResolveSlackSession(ctx, &tid, thread, "")
	gt.NoError(t, err)
	msgs, err := repo.GetSessionMessages(ctx, sess.ID)
	gt.NoError(t, err)
	gt.V(t, len(msgs)).Equal(2)

	// Rerun is idempotent — same author-content-createdAt tuple dedup.
	res2, err := job.Run(ctx, migration.Options{})
	gt.NoError(t, err)
	gt.V(t, res2.Skipped).Equal(2)
	gt.V(t, res2.Migrated).Equal(0)
}
