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

func TestTurnSynthesis_CreatesOneTurnPerSession(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()

	sid := types.SessionID("sid_ts_1")
	gt.NoError(t, repo.PutSession(ctx, &sessModel.Session{ID: sid}))
	m1 := sessModel.NewMessageV2(ctx, sid, nil, nil, sessModel.MessageTypeUser, "hi", nil)
	m2 := sessModel.NewMessageV2(ctx, sid, nil, nil, sessModel.MessageTypeResponse, "ack", nil)
	gt.NoError(t, repo.PutSessionMessage(ctx, m1))
	gt.NoError(t, repo.PutSessionMessage(ctx, m2))

	job := migration.NewTurnSynthesisJob(repo, func(_ context.Context) ([]*sessModel.Session, error) {
		return []*sessModel.Session{{ID: sid}}, nil
	})
	res, err := job.Run(ctx, migration.Options{})
	gt.NoError(t, err)
	gt.V(t, res.Migrated).Equal(1)
	gt.V(t, res.Errors).Equal(0)

	turns, err := repo.GetTurnsBySession(ctx, sid)
	gt.NoError(t, err)
	gt.V(t, len(turns)).Equal(1)
	gt.V(t, turns[0].Status).Equal(sessModel.TurnStatusCompleted)

	// Messages should now reference the turn.
	msgs, err := repo.GetSessionMessages(ctx, sid)
	gt.NoError(t, err)
	for _, m := range msgs {
		gt.V(t, m.TurnID != nil && *m.TurnID == turns[0].ID).Equal(true)
	}
}

func TestTurnSynthesis_SkipsSessionWithTurnAlready(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	sid := types.SessionID("sid_ts_skip")
	gt.NoError(t, repo.PutSession(ctx, &sessModel.Session{ID: sid}))
	existingTurn := sessModel.NewTurn(ctx, sid)
	gt.NoError(t, repo.PutTurn(ctx, existingTurn))

	job := migration.NewTurnSynthesisJob(repo, func(_ context.Context) ([]*sessModel.Session, error) {
		return []*sessModel.Session{{ID: sid}}, nil
	})
	res, err := job.Run(ctx, migration.Options{})
	gt.NoError(t, err)
	gt.V(t, res.Skipped).Equal(1)
	gt.V(t, res.Migrated).Equal(0)
}

func TestTurnSynthesis_SkipsSessionWithoutMessages(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	sid := types.SessionID("sid_ts_empty")
	gt.NoError(t, repo.PutSession(ctx, &sessModel.Session{ID: sid}))

	job := migration.NewTurnSynthesisJob(repo, func(_ context.Context) ([]*sessModel.Session, error) {
		return []*sessModel.Session{{ID: sid}}, nil
	})
	res, err := job.Run(ctx, migration.Options{})
	gt.NoError(t, err)
	gt.V(t, res.Skipped).Equal(1)
	gt.V(t, res.Migrated).Equal(0)
}

func TestTurnSynthesis_DryRunNoWrites(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	sid := types.SessionID("sid_ts_dry")
	gt.NoError(t, repo.PutSession(ctx, &sessModel.Session{ID: sid}))
	m := sessModel.NewMessageV2(ctx, sid, nil, nil, sessModel.MessageTypeUser, "hi", nil)
	gt.NoError(t, repo.PutSessionMessage(ctx, m))

	job := migration.NewTurnSynthesisJob(repo, func(_ context.Context) ([]*sessModel.Session, error) {
		return []*sessModel.Session{{ID: sid}}, nil
	})
	res, err := job.Run(ctx, migration.Options{DryRun: true})
	gt.NoError(t, err)
	gt.V(t, res.Migrated).Equal(1)

	turns, err := repo.GetTurnsBySession(ctx, sid)
	gt.NoError(t, err)
	gt.V(t, len(turns)).Equal(0)
}

func TestTurnSynthesis_RequiresDependencies(t *testing.T) {
	ctx := context.Background()
	job := migration.NewTurnSynthesisJob(nil, nil)
	_, err := job.Run(ctx, migration.Options{})
	gt.Error(t, err)
}
