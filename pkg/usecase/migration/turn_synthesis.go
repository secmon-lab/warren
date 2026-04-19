package migration

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	sessModel "github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/utils/clock"
)

// TurnSynthesisJob converts each legacy Session (pre-redesign: one
// req/res cycle) into a Turn attached to the same Session, and stamps
// every Message on that Session with the synthesized TurnID so
// downstream queries can bucket by Turn without losing per-invocation
// granularity.
//
// Idempotence: the job skips Sessions that already have at least one
// Turn persisted, so re-runs are safe.
type TurnSynthesisJob struct {
	repo    interfaces.Repository
	listAll func(ctx context.Context) ([]*sessModel.Session, error)
}

// NewTurnSynthesisJob constructs the job. listAll enumerates every
// Session; the CLI wrapper provides a Firestore-backed implementation
// (see pkg/cli/migrate_chat_session.go:listAllSessions).
func NewTurnSynthesisJob(repo interfaces.Repository, listAll func(ctx context.Context) ([]*sessModel.Session, error)) *TurnSynthesisJob {
	return &TurnSynthesisJob{repo: repo, listAll: listAll}
}

func (j *TurnSynthesisJob) Name() string { return "turn-synthesis" }

func (j *TurnSynthesisJob) Description() string {
	return "Synthesize one Turn per legacy Session and stamp TurnID onto each attached Message. Idempotent: skips Sessions that already carry a Turn."
}

func (j *TurnSynthesisJob) Run(ctx context.Context, opts Options) (*Result, error) {
	if j.repo == nil || j.listAll == nil {
		return nil, goerr.New("turn-synthesis: dependencies not wired")
	}

	sessions, err := j.listAll(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to list sessions")
	}

	result := &Result{JobName: j.Name()}
	for _, s := range sessions {
		if s == nil {
			continue
		}
		result.Scanned++

		existingTurns, err := j.repo.GetTurnsBySession(ctx, s.ID)
		if err != nil {
			result.Errors++
			continue
		}
		if len(existingTurns) > 0 {
			result.Skipped++
			continue
		}

		msgs, err := j.repo.GetSessionMessages(ctx, s.ID)
		if err != nil {
			result.Errors++
			continue
		}
		if len(msgs) == 0 {
			result.Skipped++
			continue
		}

		if opts.DryRun {
			result.Migrated++
			continue
		}

		turn := sessModel.NewTurn(ctx, s.ID)
		// Close immediately: legacy Sessions are already completed.
		closedAt := clock.Now(ctx)
		if s.LastActiveAt != (s.LastActiveAt) && !s.LastActiveAt.IsZero() {
			closedAt = s.LastActiveAt
		}
		turn.Status = sessModel.TurnStatusCompleted
		turn.EndedAt = &closedAt

		if err := j.repo.PutTurn(ctx, turn); err != nil {
			result.Errors++
			continue
		}

		// Stamp every Message with the new TurnID.
		stampErrs := 0
		for _, m := range msgs {
			if m == nil {
				continue
			}
			tid := turn.ID
			m.TurnID = &tid
			if err := j.repo.PutSessionMessage(ctx, m); err != nil {
				stampErrs++
			}
		}
		if stampErrs > 0 {
			result.Errors += stampErrs
			continue
		}
		result.Migrated++
	}
	return result, nil
}
