package migration

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
)

// CleanupLegacyJob removes pre-redesign data structures after the other
// migration jobs have successfully populated their new equivalents.
// Specifically: ticket_comments documents, legacy gollem.History files
// under ticket/{tid}/, and the unused legacy Session fields (Query /
// SlackURL / Status / RequestID / UpdatedAt) on migrated Sessions.
//
// This job is DESTRUCTIVE and MUST only be executed after dry-run
// verification of comment-to-message, session-source-backfill,
// history-scope, and turn-synthesis outputs on a production snapshot.
type CleanupLegacyJob struct {
	repo interfaces.Repository
}

func NewCleanupLegacyJob(repo interfaces.Repository) *CleanupLegacyJob {
	return &CleanupLegacyJob{repo: repo}
}

func (j *CleanupLegacyJob) Name() string { return "cleanup-legacy" }

func (j *CleanupLegacyJob) Description() string {
	return "Remove legacy ticket_comments, legacy gollem.History files, and deprecated Session fields after other migrations have completed. DESTRUCTIVE; always run --dry-run first. Skeleton; Phase 7.4 full wiring pending."
}

func (j *CleanupLegacyJob) Run(ctx context.Context, opts Options) (*Result, error) {
	return nil, goerr.New("cleanup-legacy: not yet implemented",
		goerr.V("note", "Phase 7.4 wiring pending; requires Firestore batch delete and GCS listing"))
}
