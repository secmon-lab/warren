// Package migration implements the one-shot data migration jobs required
// by the chat-session-redesign spec (Phase 7). Jobs are invoked via the
// `warren migrate --job <name>` CLI; this package holds the
// business-logic implementation while pkg/cli/migrate.go only wires each
// job into the existing migrationJobs slice.
//
// All jobs are idempotent: safe to re-run, stop, and resume. Each
// implementation documents the key used to skip already-migrated rows.
package migration

import (
	"context"
	"maps"

	sessModel "github.com/secmon-lab/warren/pkg/domain/model/session"
)

// SessionForEach streams every Session in the target dataset through
// the supplied handle callback. Migration jobs consume this instead
// of a ([]*Session, error) returner so they process one Session at a
// time — keeping memory use bounded regardless of collection size.
//
// If handle returns a non-nil error, iteration stops and the error
// bubbles up from forEach.
type SessionForEach func(ctx context.Context, handle func(*sessModel.Session) error) error

// Job describes a single migration unit. Name is the CLI flag
// (`--job <name>`); Description is the short blurb printed by
// `warren migrate --list`.
type Job interface {
	Name() string
	Description() string
	Run(ctx context.Context, opts Options) (*Result, error)
}

// Options carries the shared configuration passed down to every Job.
// Concrete job implementations embed the clients they need when
// constructed (see NewCommentToMessageJob etc.), so Options only needs
// the small set of cross-cutting flags.
type Options struct {
	// DryRun instructs the job to compute and report what it would do
	// without mutating any data. Every Job MUST honor this.
	DryRun bool
}

// Result is the structured report returned by Job.Run. Scanned / Migrated
// / Skipped / Errors are counters; Details is for job-specific fields
// (e.g. a list of Session IDs that were merged, file paths copied, etc.).
type Result struct {
	JobName  string         `json:"job_name"`
	Scanned  int            `json:"scanned"`
	Migrated int            `json:"migrated"`
	Skipped  int            `json:"skipped"`
	Errors   int            `json:"errors"`
	Details  map[string]any `json:"details,omitempty"`
}

// MergeDetails is a small helper used by individual jobs to stash extra
// reporting fields without worrying about nil maps.
func (r *Result) MergeDetails(kv map[string]any) {
	if r.Details == nil {
		r.Details = map[string]any{}
	}
	maps.Copy(r.Details, kv)
}

// RunBundle invokes every Job in `jobs` sequentially against `opts`,
// collecting Results. The first error short-circuits the run; preceding
// Results are still returned so callers can see how far the bundle got.
//
// This is the orchestration primitive shared by the v0.16.0 CLI bundle
// and the bundle's tests; keeping it in this package means the same
// ordering logic is exercised by both code paths.
func RunBundle(ctx context.Context, opts Options, jobs ...Job) ([]*Result, error) {
	results := make([]*Result, 0, len(jobs))
	for _, j := range jobs {
		res, err := j.Run(ctx, opts)
		if res != nil {
			results = append(results, res)
		}
		if err != nil {
			return results, err
		}
	}
	return results, nil
}
