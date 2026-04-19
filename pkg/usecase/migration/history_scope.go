package migration

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	sessModel "github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/service/storage"
)

// HistoryScopeJob copies legacy Ticket-scoped gollem.History files at
// `{prefix}/{schema}/ticket/{tid}/latest.json` into Session-scoped
// destinations at `{prefix}/{schema}/sessions/{sid}/history.json`.
//
// Source mapping: for each Session with Source=slack and a resolved
// Ticket, the ticket's latest.json is copied into the Session's history
// slot. Web/CLI Sessions did not exist pre-redesign and are skipped.
//
// The job is idempotent: when the Session already has a history slot
// populated, it is left untouched. Legacy files are never deleted here;
// cleanup-legacy handles removal after this job succeeds.
type HistoryScopeJob struct {
	storageSvc *storage.Service
	listAll    func(ctx context.Context) ([]*sessModel.Session, error)
}

// NewHistoryScopeJob constructs the job. listAll enumerates every
// Session; the CLI wrapper provides a Firestore-backed implementation
// (see pkg/cli/migrate_chat_session.go:listAllSessions).
func NewHistoryScopeJob(svc *storage.Service, listAll func(ctx context.Context) ([]*sessModel.Session, error)) *HistoryScopeJob {
	return &HistoryScopeJob{storageSvc: svc, listAll: listAll}
}

func (j *HistoryScopeJob) Name() string { return "history-scope" }

func (j *HistoryScopeJob) Description() string {
	return "Copy legacy Ticket-scoped gollem.History latest.json files into Session-scoped sessions/{sid}/history.json destinations. Skips destinations that already exist; leaves legacy files in place for cleanup-legacy."
}

func (j *HistoryScopeJob) Run(ctx context.Context, opts Options) (*Result, error) {
	if j.storageSvc == nil || j.listAll == nil {
		return nil, goerr.New("history-scope: dependencies not wired")
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

		// Only Slack sessions had a legacy ticket-scoped history;
		// Web/CLI are new and start empty.
		if s.Source != sessModel.SessionSourceSlack {
			result.Skipped++
			continue
		}
		tid := s.TicketIDOrNil()
		if tid == nil {
			result.Skipped++
			continue
		}

		// Skip if the destination already has history.
		existing, err := j.storageSvc.GetSessionHistory(ctx, s.ID)
		if err == nil && existing != nil && existing.ToCount() > 0 {
			result.Skipped++
			continue
		}

		// Read legacy source.
		src, err := j.storageSvc.GetLatestHistory(ctx, *tid)
		if err != nil || src == nil || src.ToCount() == 0 {
			// No legacy data to copy — not an error, just nothing to
			// migrate for this session.
			result.Skipped++
			continue
		}

		if opts.DryRun {
			result.Migrated++
			continue
		}

		if err := j.storageSvc.PutSessionHistory(ctx, s.ID, src); err != nil {
			result.Errors++
			continue
		}
		result.Migrated++
		result.MergeDetails(map[string]any{string(s.ID): string(*tid)})
	}
	return result, nil
}
