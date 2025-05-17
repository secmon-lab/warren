package alert

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/utils/test"
)

func TestAlert_FillMetadata(t *testing.T) {
	client := test.NewGeminiClient(t)

	alert := New(t.Context(), "test.alert.v1", map[string]any{
		"foo": "bar",
		"baz": 123,
	}, Metadata{})

	gt.NoError(t, alert.FillMetadata(t.Context(), client))
	gt.NotEqual(t, alert.Metadata.Title, "")
	gt.NotEqual(t, alert.Metadata.Description, "")
	t.Logf("metadata: %+v", alert.Metadata)
}
