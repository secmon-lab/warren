package migration

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	sessModel "github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

// SessionSourceBackfillJob walks every Session currently persisted and
// sets Source / TicketIDPtr on rows that lack them. Pre-redesign Sessions
// were all Slack (the only code path that created Sessions), so the
// default inferred Source is `slack`. Rows whose SlackURL is empty are
// treated as Web sessions.
//
// The job is idempotent: rows that already carry a valid Source are
// counted as Skipped.
type SessionSourceBackfillJob struct {
	repo     interfaces.Repository
	listAll  func(ctx context.Context) ([]*sessModel.Session, error)
}

// NewSessionSourceBackfillJob constructs the job. listAll lets tests
// stub out the "list every Session in the system" step; production
// wiring passes a function that iterates `sessions` via Firestore.
func NewSessionSourceBackfillJob(repo interfaces.Repository, listAll func(ctx context.Context) ([]*sessModel.Session, error)) *SessionSourceBackfillJob {
	return &SessionSourceBackfillJob{repo: repo, listAll: listAll}
}

func (j *SessionSourceBackfillJob) Name() string { return "session-source-backfill" }

func (j *SessionSourceBackfillJob) Description() string {
	return "Backfill SessionSource and TicketIDPtr on legacy Session rows. Sessions with a non-empty SlackURL become source=slack; others become source=web. Idempotent: rows already carrying a valid Source are left untouched."
}

func (j *SessionSourceBackfillJob) Run(ctx context.Context, opts Options) (*Result, error) {
	if j.listAll == nil {
		return nil, goerr.New("session-source-backfill: listAll dependency is not wired")
	}
	sessions, err := j.listAll(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to list sessions for backfill")
	}

	result := &Result{JobName: j.Name(), Scanned: len(sessions)}
	var slackCount, webCount int

	for _, s := range sessions {
		if s.Source.Valid() && s.TicketIDPtr != nil {
			result.Skipped++
			continue
		}
		inferred := inferSource(s)
		ticketPtr := inferTicketIDPtr(s)

		if opts.DryRun {
			result.Migrated++
			if inferred == sessModel.SessionSourceSlack {
				slackCount++
			} else if inferred == sessModel.SessionSourceWeb {
				webCount++
			}
			continue
		}

		// Apply the inferred values. TicketIDPtr propagation uses the
		// existing PromoteSessionToTicket method so we do not duplicate
		// transactional logic here.
		updated := *s
		updated.Source = inferred
		if ticketPtr != nil {
			updated.TicketIDPtr = ticketPtr
		}
		if err := j.repo.PutSession(ctx, &updated); err != nil {
			result.Errors++
			continue
		}
		result.Migrated++
		if inferred == sessModel.SessionSourceSlack {
			slackCount++
		} else if inferred == sessModel.SessionSourceWeb {
			webCount++
		}
	}

	result.MergeDetails(map[string]any{
		"inferred_slack": slackCount,
		"inferred_web":   webCount,
	})
	return result, nil
}

// inferSource deduces the Source for a legacy Session row. SlackURL
// presence is the most reliable signal: it was only set by
// ChatFromSlack -> createSession.
func inferSource(s *sessModel.Session) sessModel.SessionSource {
	if s.Source.Valid() {
		return s.Source
	}
	if s.SlackURL != "" {
		return sessModel.SessionSourceSlack
	}
	return sessModel.SessionSourceWeb
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
