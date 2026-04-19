package migration

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/storage"
)

// CleanupLegacyJob removes pre-redesign data structures after the other
// migration jobs have successfully populated their new equivalents.
// Three classes of data are purged:
//
//   1. ticket.Comment subcollection documents (replaced by type=user
//      session.Messages via comment-to-message).
//   2. Legacy ticket-scoped gollem.History snapshots at
//      `{prefix}/{schema}/ticket/{tid}/latest.json` (replaced by
//      session-scoped history via history-scope).
//   3. Nothing on Session documents themselves — the deprecated fields
//      (Query / SlackURL / Status / RequestID / UpdatedAt) are left in
//      place since reading them costs nothing; they will drop off
//      naturally when the next PutSession rewrite runs against the new
//      schema.
//
// This job is DESTRUCTIVE and MUST only be executed after dry-run
// verification of comment-to-message, session-source-backfill,
// history-scope, and turn-synthesis outputs on a production snapshot.
type CleanupLegacyJob struct {
	store      LegacyCommentStore
	storageSvc *storage.Service
	forEach    SessionForEach
}

// NewCleanupLegacyJob constructs the job. `store` provides the raw
// read/delete surface for the legacy `ticket_comments` subcollection —
// in production this is a thin wrapper over the Firestore client (see
// pkg/cli/migrate_chat_session.go:firestoreCommentSource), in tests
// a Memory-backed implementation is sufficient. storageSvc is required
// to purge legacy latest.json files; pass nil to skip storage cleanup.
// `forEach` streams Sessions; it is currently unused by the cleanup
// body but retained in the signature so callers can pre-wire it
// uniformly across migration jobs.
func NewCleanupLegacyJob(
	store LegacyCommentStore,
	storageSvc *storage.Service,
	forEach SessionForEach,
) *CleanupLegacyJob {
	return &CleanupLegacyJob{
		store:      store,
		storageSvc: storageSvc,
		forEach:    forEach,
	}
}

func (j *CleanupLegacyJob) Name() string { return "cleanup-legacy" }

func (j *CleanupLegacyJob) Description() string {
	return "Remove legacy ticket_comments, legacy gollem.History latest.json files, after comment-to-message / history-scope have run. DESTRUCTIVE; always run --dry-run first."
}

func (j *CleanupLegacyJob) Run(ctx context.Context, opts Options) (*Result, error) {
	if j.store == nil {
		return nil, goerr.New("cleanup-legacy: dependencies not wired")
	}

	result := &Result{JobName: j.Name()}

	// 1. Delete ticket_comments documents.
	tickets, err := j.store.ListTicketsWithComments(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to list tickets")
	}
	for _, t := range tickets {
		if t == nil {
			continue
		}
		comments, err := j.store.GetTicketComments(ctx, t.ID)
		if err != nil {
			result.Errors++
			continue
		}
		for _, c := range comments {
			result.Scanned++
			if opts.DryRun {
				result.Migrated++
				continue
			}
			if err := j.store.DeleteTicketComment(ctx, t.ID, c.ID); err != nil {
				result.Errors++
				continue
			}
			result.Migrated++
		}
	}

	// 2. Delete legacy latest.json files. Skip when storage is not
	//    configured (e.g. Firestore-only test harnesses).
	if j.storageSvc != nil {
		seen := map[types.TicketID]bool{}
		for _, t := range tickets {
			if t == nil || seen[t.ID] {
				continue
			}
			seen[t.ID] = true
			result.Scanned++
			if opts.DryRun {
				result.Migrated++
				continue
			}
			if err := j.storageSvc.DeleteLatestHistory(ctx, t.ID); err != nil {
				result.Errors++
				continue
			}
			result.Migrated++
		}
	}

	return result, nil
}
