package migration

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
)

// TurnSynthesisJob translates each legacy Session (= 1 req/res cycle in
// the pre-redesign model) into a Turn on the newly-reshaped Session
// (= conversation unit). Messages retain their existing SessionID but
// gain a TurnID pointing at the synthesized Turn so downstream queries
// can bucket by Turn without losing per-invocation granularity.
//
// This job depends on decisions about how to reconcile 1:N
// old-Session-to-new-Session mappings that the spec marks as a
// production-data exercise; it is therefore left as a skeleton until
// the data shape is validated against a restored snapshot.
type TurnSynthesisJob struct {
	repo interfaces.Repository
}

func NewTurnSynthesisJob(repo interfaces.Repository) *TurnSynthesisJob {
	return &TurnSynthesisJob{repo: repo}
}

func (j *TurnSynthesisJob) Name() string { return "turn-synthesis" }

func (j *TurnSynthesisJob) Description() string {
	return "Synthesize Turn entities from pre-redesign Sessions and attach Messages to the correct Turn. Skeleton; Phase 7 full wiring pending on production data review."
}

func (j *TurnSynthesisJob) Run(ctx context.Context, opts Options) (*Result, error) {
	return nil, goerr.New("turn-synthesis: not yet implemented",
		goerr.V("note", "See chat-session-redesign spec Phase 7.2b"))
}
