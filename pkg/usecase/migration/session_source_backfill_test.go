package migration_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	sessModel "github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	"github.com/secmon-lab/warren/pkg/usecase/migration"
)

// listAllFrom builds a closure that returns every session currently in
// the in-memory repository. Used to wire production-equivalent
// enumeration for tests without coupling to Firestore iteration details.
func listAllFrom(repo *repository.Memory) migration.SessionForEach {
	return sessionsForEach(repo.AllSessions())
}

// sessionsForEach wraps a []*Session slice into a SessionForEach so
// migration tests can feed a pre-populated list into jobs without
// spinning up an iterator.
func sessionsForEach(sessions []*sessModel.Session) migration.SessionForEach {
	return func(_ context.Context, handle func(*sessModel.Session) error) error {
		for _, s := range sessions {
			if err := handle(s); err != nil {
				return err
			}
		}
		return nil
	}
}

func TestSessionSourceBackfill_InfersFromSlackURL(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()

	// Legacy Session with SlackURL present → slack
	slackLike := &sessModel.Session{
		ID:       types.SessionID("sid_slack"),
		TicketID: types.TicketID("tid_1"),
		SlackURL: "https://slack.com/archives/C1/p123",
	}
	gt.NoError(t, repo.PutSession(ctx, slackLike))

	// Legacy Session without SlackURL → web
	webLike := &sessModel.Session{
		ID:       types.SessionID("sid_web"),
		TicketID: types.TicketID("tid_2"),
	}
	gt.NoError(t, repo.PutSession(ctx, webLike))

	job := migration.NewSessionSourceBackfillJob(repo, listAllFrom(repo))
	res, err := job.Run(ctx, migration.Options{})
	gt.NoError(t, err)
	gt.V(t, res.Scanned).Equal(2)
	gt.V(t, res.Migrated).Equal(2)
	gt.V(t, res.Errors).Equal(0)

	after, err := repo.GetSession(ctx, "sid_slack")
	gt.NoError(t, err)
	gt.V(t, after.Source).Equal(sessModel.SessionSourceSlack)
	gt.V(t, after.TicketIDPtr != nil && *after.TicketIDPtr == "tid_1").Equal(true)

	after2, err := repo.GetSession(ctx, "sid_web")
	gt.NoError(t, err)
	gt.V(t, after2.Source).Equal(sessModel.SessionSourceWeb)
}

func TestSessionSourceBackfill_DryRunWritesNothing(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()

	orig := &sessModel.Session{
		ID:       types.SessionID("sid_d"),
		TicketID: types.TicketID("tid_d"),
		SlackURL: "x",
	}
	gt.NoError(t, repo.PutSession(ctx, orig))

	job := migration.NewSessionSourceBackfillJob(repo, listAllFrom(repo))
	res, err := job.Run(ctx, migration.Options{DryRun: true})
	gt.NoError(t, err)
	gt.V(t, res.Migrated).Equal(1)

	after, err := repo.GetSession(ctx, "sid_d")
	gt.NoError(t, err)
	// Source not populated in the store; dry-run only reports.
	gt.V(t, after.Source).Equal(sessModel.SessionSource(""))
}

func TestSessionSourceBackfill_IdempotentOnAlreadyMigrated(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()

	tid := types.TicketID("tid_i")
	migrated := &sessModel.Session{
		ID:          types.SessionID("sid_i"),
		TicketID:    tid,
		TicketIDPtr: &tid,
		Source:      sessModel.SessionSourceSlack,
	}
	gt.NoError(t, repo.PutSession(ctx, migrated))

	job := migration.NewSessionSourceBackfillJob(repo, listAllFrom(repo))
	res, err := job.Run(ctx, migration.Options{})
	gt.NoError(t, err)
	gt.V(t, res.Scanned).Equal(1)
	gt.V(t, res.Skipped).Equal(1)
	gt.V(t, res.Migrated).Equal(0)
}

func TestSessionSourceBackfill_RequiresListAll(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	job := migration.NewSessionSourceBackfillJob(repo, nil)
	_, err := job.Run(ctx, migration.Options{})
	gt.Error(t, err)
}
