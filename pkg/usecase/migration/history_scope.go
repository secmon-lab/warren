package migration

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/service/storage"
)

// HistoryScopeJob copies legacy Ticket-scoped gollem.History files at
// `{prefix}/{schema}/ticket/{tid}/latest.json` into Session-scoped
// destinations at `{prefix}/{schema}/sessions/{sid}/history.json`.
//
// The job is idempotent: it skips destinations that already exist and
// does not delete the legacy file (cleanup-legacy handles that).
type HistoryScopeJob struct {
	repo       interfaces.Repository
	storageSvc *storage.Service
}

func NewHistoryScopeJob(repo interfaces.Repository, svc *storage.Service) *HistoryScopeJob {
	return &HistoryScopeJob{repo: repo, storageSvc: svc}
}

func (j *HistoryScopeJob) Name() string { return "history-scope" }

func (j *HistoryScopeJob) Description() string {
	return "Copy legacy Ticket-scoped gollem.History latest.json files into Session-scoped sessions/{sid}/history.json destinations. Skips destinations that already exist; leaves legacy files in place for cleanup-legacy."
}

func (j *HistoryScopeJob) Run(ctx context.Context, opts Options) (*Result, error) {
	return nil, goerr.New("history-scope: not yet implemented",
		goerr.V("note", "Phase 7.3 full wiring pending; requires storage bucket enumerator"))
}
