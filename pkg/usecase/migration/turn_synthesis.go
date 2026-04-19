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
	forEach SessionForEach
}

// NewTurnSynthesisJob constructs the job. `forEach` streams every
// Session through the handle callback; the CLI wrapper wires a
// Firestore iterator (see pkg/cli/migrate_chat_session.go:
// forEachSession).
func NewTurnSynthesisJob(repo interfaces.Repository, forEach SessionForEach) *TurnSynthesisJob {
	return &TurnSynthesisJob{repo: repo, forEach: forEach}
}

func (j *TurnSynthesisJob) Name() string { return "turn-synthesis" }

func (j *TurnSynthesisJob) Description() string {
	return "Synthesize one Turn per legacy Session and stamp TurnID onto each attached Message. Idempotent: skips Sessions that already carry a Turn."
}

func (j *TurnSynthesisJob) Run(ctx context.Context, opts Options) (*Result, error) {
	if j.repo == nil || j.forEach == nil {
		return nil, goerr.New("turn-synthesis: dependencies not wired")
	}

	result := &Result{JobName: j.Name()}
	if err := j.forEach(ctx, func(s *sessModel.Session) error {
		if s == nil {
			return nil
		}
		result.Scanned++

		existingTurns, err := j.repo.GetTurnsBySession(ctx, s.ID)
		if err != nil {
			result.Errors++
			return nil
		}
		if len(existingTurns) > 0 {
			result.Skipped++
			return nil
		}

		msgs, err := j.repo.GetSessionMessages(ctx, s.ID)
		if err != nil {
			result.Errors++
			return nil
		}
		if len(msgs) == 0 {
			result.Skipped++
			return nil
		}

		if opts.DryRun {
			result.Migrated++
			return nil
		}

		turn := sessModel.NewTurn(ctx, s.ID)
		// Close immediately: legacy Sessions are already completed.
		closedAt := clock.Now(ctx)
		if !s.LastActiveAt.IsZero() {
			closedAt = s.LastActiveAt
		}
		turn.Status = sessModel.TurnStatusCompleted
		turn.EndedAt = &closedAt

		if err := j.repo.PutTurn(ctx, turn); err != nil {
			result.Errors++
			return nil
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
			return nil
		}
		result.Migrated++
		return nil
	}); err != nil {
		return nil, goerr.Wrap(err, "failed to iterate sessions for turn-synthesis")
	}
	return result, nil
}
