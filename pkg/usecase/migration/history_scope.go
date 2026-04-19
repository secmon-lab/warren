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
	forEach    SessionForEach
}

// NewHistoryScopeJob constructs the job. `forEach` streams every
// Session through the handle callback.
func NewHistoryScopeJob(svc *storage.Service, forEach SessionForEach) *HistoryScopeJob {
	return &HistoryScopeJob{storageSvc: svc, forEach: forEach}
}

func (j *HistoryScopeJob) Name() string { return "history-scope" }

func (j *HistoryScopeJob) Description() string {
	return "Copy legacy Ticket-scoped gollem.History latest.json files into Session-scoped sessions/{sid}/history.json destinations. Skips destinations that already exist; leaves legacy files in place for cleanup-legacy."
}

func (j *HistoryScopeJob) Run(ctx context.Context, opts Options) (*Result, error) {
	if j.storageSvc == nil || j.forEach == nil {
		return nil, goerr.New("history-scope: dependencies not wired")
	}

	result := &Result{JobName: j.Name()}
	if err := j.forEach(ctx, func(s *sessModel.Session) error {
		if s == nil {
			return nil
		}
		result.Scanned++

		if s.Source != sessModel.SessionSourceSlack {
			result.Skipped++
			return nil
		}
		tid := s.TicketIDOrNil()
		if tid == nil {
			result.Skipped++
			return nil
		}

		existing, err := j.storageSvc.GetSessionHistory(ctx, s.ID)
		if err == nil && existing != nil && existing.ToCount() > 0 {
			result.Skipped++
			return nil
		}

		src, err := j.storageSvc.GetLatestHistory(ctx, *tid)
		if err != nil || src == nil || src.ToCount() == 0 {
			result.Skipped++
			return nil
		}

		if opts.DryRun {
			result.Migrated++
			return nil
		}

		if err := j.storageSvc.PutSessionHistory(ctx, s.ID, src); err != nil {
			result.Errors++
			return nil
		}
		result.Migrated++
		result.MergeDetails(map[string]any{string(s.ID): string(*tid)})
		return nil
	}); err != nil {
		return nil, goerr.Wrap(err, "failed to iterate sessions for history-scope")
	}
	return result, nil
}
