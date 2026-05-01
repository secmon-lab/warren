package migration_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	sessModel "github.com/secmon-lab/warren/pkg/domain/model/session"
	slackModel "github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	sessService "github.com/secmon-lab/warren/pkg/service/session"
	"github.com/secmon-lab/warren/pkg/usecase/migration"
)

func TestSessionConsolidate_MaterializesCanonicalSlackSession(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	resolver := sessService.NewResolver(repo)

	tid := types.TicketID("tid_consolidate")
	thread := slackModel.Thread{ChannelID: "C1", ThreadID: "1700000000.000300"}
	gt.NoError(t, repo.PutTicket(ctx, ticket.Ticket{ID: tid, SlackThread: &thread}))

	job := migration.NewSessionConsolidateJob(repo, resolver)

	res, err := job.Run(ctx, migration.Options{})
	gt.NoError(t, err)
	gt.V(t, res.Scanned).Equal(1)
	gt.V(t, res.Migrated).Equal(1)
	gt.V(t, res.Skipped).Equal(0)
	gt.V(t, res.Errors).Equal(0)

	// The canonical Session must now exist and be returned by Lookup.
	got, ok, err := resolver.LookupSlackSession(ctx, &tid, thread)
	gt.NoError(t, err)
	gt.V(t, ok).Equal(true)
	gt.V(t, got).NotNil()
	gt.V(t, got.Source).Equal(sessModel.SessionSourceSlack)
	gt.V(t, got.TicketIDPtr != nil && *got.TicketIDPtr == tid).Equal(true)

	// Re-run is idempotent: existing canonical Session is reused.
	res2, err := job.Run(ctx, migration.Options{})
	gt.NoError(t, err)
	gt.V(t, res2.Scanned).Equal(1)
	gt.V(t, res2.Migrated).Equal(0)
	gt.V(t, res2.Skipped).Equal(1)
}

func TestSessionConsolidate_DryRunWritesNothing(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	resolver := sessService.NewResolver(repo)

	tid := types.TicketID("tid_dry_consolidate")
	thread := slackModel.Thread{ChannelID: "C1", ThreadID: "1700000000.000400"}
	gt.NoError(t, repo.PutTicket(ctx, ticket.Ticket{ID: tid, SlackThread: &thread}))

	job := migration.NewSessionConsolidateJob(repo, resolver)
	res, err := job.Run(ctx, migration.Options{DryRun: true})
	gt.NoError(t, err)
	gt.V(t, res.Scanned).Equal(1)
	gt.V(t, res.Migrated).Equal(1)

	// DryRun must not have created the Session.
	_, ok, err := resolver.LookupSlackSession(ctx, &tid, thread)
	gt.NoError(t, err)
	gt.V(t, ok).Equal(false)
}

func TestSessionConsolidate_SkipsTicketsWithoutSlackThread(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	resolver := sessService.NewResolver(repo)

	tidNoThread := types.TicketID("tid_no_thread")
	tidWithThread := types.TicketID("tid_with_thread")
	thread := slackModel.Thread{ChannelID: "C2", ThreadID: "1700000000.000500"}
	gt.NoError(t, repo.PutTicket(ctx, ticket.Ticket{ID: tidNoThread}))
	gt.NoError(t, repo.PutTicket(ctx, ticket.Ticket{ID: tidWithThread, SlackThread: &thread}))

	job := migration.NewSessionConsolidateJob(repo, resolver)
	res, err := job.Run(ctx, migration.Options{})
	gt.NoError(t, err)
	// Only the SlackThread-bearing ticket is scanned.
	gt.V(t, res.Scanned).Equal(1)
	gt.V(t, res.Migrated).Equal(1)
}

func TestSessionConsolidate_RequiresDependencies(t *testing.T) {
	job := migration.NewSessionConsolidateJob(nil, nil)
	_, err := job.Run(context.Background(), migration.Options{})
	gt.Error(t, err)
}
