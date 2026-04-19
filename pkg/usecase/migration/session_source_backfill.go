package migration

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	sessModel "github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

// SessionSourceBackfillJob walks every Session currently persisted and
// sets Source=slack / TicketIDPtr on rows that lack them.
//
// Pre-redesign Sessions were all Slack — there was no Web or CLI
// Session code path before chat-session-redesign, so every row whose
// Source is empty is, by construction, a Slack Session. No inference
// from SlackURL or ChannelRef is required (and doing so
// misclassifies rows where the legacy constructor forgot to populate
// SlackURL).
//
// The job is idempotent: rows that already carry a valid Source are
// counted as Skipped.
type SessionSourceBackfillJob struct {
	repo    interfaces.Repository
	forEach SessionForEach
}

// NewSessionSourceBackfillJob constructs the job. `forEach` streams
// every Session through the handle callback; tests pass an in-memory
// closure and the CLI wires a Firestore iterator. Streaming avoids
// loading the entire session collection into memory.
func NewSessionSourceBackfillJob(repo interfaces.Repository, forEach SessionForEach) *SessionSourceBackfillJob {
	return &SessionSourceBackfillJob{repo: repo, forEach: forEach}
}

func (j *SessionSourceBackfillJob) Name() string { return "session-source-backfill" }

func (j *SessionSourceBackfillJob) Description() string {
	return "Backfill SessionSource and TicketIDPtr on legacy Session rows. Every pre-redesign Session was created by Slack mentions, so rows with an empty Source become source=slack unconditionally. Idempotent: rows already carrying a valid Source are left untouched."
}

func (j *SessionSourceBackfillJob) Run(ctx context.Context, opts Options) (*Result, error) {
	if j.forEach == nil {
		return nil, goerr.New("session-source-backfill: forEach dependency is not wired")
	}

	result := &Result{JobName: j.Name()}

	if err := j.forEach(ctx, func(s *sessModel.Session) error {
		result.Scanned++
		if s.Source.Valid() && s.TicketIDPtr != nil {
			result.Skipped++
			return nil
		}
		ticketPtr := inferTicketIDPtr(s)

		if opts.DryRun {
			result.Migrated++
			return nil
		}

		updated := *s
		if !updated.Source.Valid() {
			updated.Source = sessModel.SessionSourceSlack
		}
		if ticketPtr != nil {
			updated.TicketIDPtr = ticketPtr
		}
		if err := j.repo.PutSession(ctx, &updated); err != nil {
			result.Errors++
			return nil
		}
		result.Migrated++
		return nil
	}); err != nil {
		return nil, goerr.Wrap(err, "failed to iterate sessions for backfill")
	}

	return result, nil
}


// inferTicketIDPtr returns a pointer to the Session's existing TicketID
// when the legacy column is populated, or the existing TicketIDPtr
// verbatim. Returns nil only when neither field carries a ticket.
func inferTicketIDPtr(s *sessModel.Session) *types.TicketID {
	if s.TicketIDPtr != nil {
		return s.TicketIDPtr
	}
	if s.TicketID != "" {
		tid := s.TicketID
		return &tid
	}
	return nil
}
