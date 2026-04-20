package migration

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
)

// SessionConsolidateJob guarantees that every Slack-origin Ticket
// has its canonical `slack_<hash>` Session persisted. Pre-redesign
// Warren created one UUID Session per @warren mention; those legacy
// rows are left in Firestore untouched — the Conversation UI
// filters them out at read time — and a separate cleanup job is
// expected to purge them later.
//
// The job is idempotent: calling ResolveSlackSession on a thread
// whose canonical Session already exists returns the existing row
// without a second write.
type SessionConsolidateJob struct {
	tickets  TicketList
	resolver CommentResolverClient
}

// TicketList is the read-only subset of the repository used to
// enumerate Tickets. Declared as a separate interface so tests can
// pass a trivial fake without wiring the whole Repository.
type TicketList interface {
	GetAllTickets(ctx context.Context) ([]*ticket.Ticket, error)
}

// NewSessionConsolidateJob wires the job's dependencies.
func NewSessionConsolidateJob(tickets TicketList, resolver CommentResolverClient) *SessionConsolidateJob {
	return &SessionConsolidateJob{tickets: tickets, resolver: resolver}
}

func (j *SessionConsolidateJob) Name() string { return "session-consolidate" }

func (j *SessionConsolidateJob) Description() string {
	return "Materialize the canonical slack_<hash> Session for every Slack-origin Ticket so the Conversation sidebar always has a single survivor per thread. Legacy UUID Session rows are left in place; a separate cleanup job is responsible for removing them."
}

func (j *SessionConsolidateJob) Run(ctx context.Context, opts Options) (*Result, error) {
	if j.tickets == nil || j.resolver == nil {
		return nil, goerr.New("session-consolidate: dependencies not wired")
	}
	result := &Result{JobName: j.Name()}
	var created, existed int

	tickets, err := j.tickets.GetAllTickets(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to list tickets")
	}

	for _, t := range tickets {
		if t == nil || t.SlackThread == nil {
			continue
		}
		result.Scanned++
		tid := t.ID
		th := *t.SlackThread

		if opts.DryRun {
			// Look up only; do not write. Counts reflect what
			// Run(DryRun=false) would do.
			_, ok, err := j.resolver.LookupSlackSession(ctx, &tid, th)
			if err != nil {
				errutil.Handle(ctx, goerr.Wrap(err, "session-consolidate: lookup failed",
					goerr.V("ticket_id", tid)))
				result.Errors++
				continue
			}
			if ok {
				existed++
				result.Skipped++
			} else {
				created++
				result.Migrated++
			}
			continue
		}

		// ResolveSlackSession is deterministic and idempotent: if the
		// canonical Session already exists, it returns the existing
		// row; otherwise it creates one. The `created` return flag
		// distinguishes the two outcomes for reporting. UserID is
		// unknown at migration time (we are acting on behalf of the
		// ticket, not any individual user) so we pass the zero value.
		_, wasCreated, err := j.resolver.ResolveSlackSession(ctx, &tid, th, types.UserID(""))
		if err != nil {
			errutil.Handle(ctx, goerr.Wrap(err, "session-consolidate: resolve failed",
				goerr.V("ticket_id", tid)))
			result.Errors++
			continue
		}
		if wasCreated {
			created++
			result.Migrated++
		} else {
			existed++
			result.Skipped++
		}
	}

	result.MergeDetails(map[string]any{
		"created": created,
		"existed": existed,
	})
	return result, nil
}
