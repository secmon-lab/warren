package migration_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/usecase/migration"
)

func TestResult_MergeDetails_InitializesMap(t *testing.T) {
	r := &migration.Result{JobName: "test"}
	r.MergeDetails(map[string]any{"a": 1})
	gt.V(t, r.Details["a"]).Equal(1)
}

func TestResult_MergeDetails_Appends(t *testing.T) {
	r := &migration.Result{JobName: "test", Details: map[string]any{"keep": "original"}}
	r.MergeDetails(map[string]any{"new": "value"})
	gt.V(t, r.Details["keep"]).Equal("original")
	gt.V(t, r.Details["new"]).Equal("value")
}
