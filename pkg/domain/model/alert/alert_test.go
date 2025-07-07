package alert_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/utils/embedding"
	"github.com/secmon-lab/warren/pkg/utils/test"
)

func TestAlert_FillMetadata(t *testing.T) {
	client := test.NewGeminiClient(t)

	a := alert.New(t.Context(), "test.alert.v1", map[string]any{
		"foo": "bar",
		"baz": 123,
	}, alert.Metadata{})

	gt.NoError(t, a.FillMetadata(t.Context(), client))
	gt.NotEqual(t, a.Metadata.Title, "")
	gt.NotEqual(t, a.Metadata.Description, "")
	gt.Equal(t, len(a.Embedding), embedding.EmbeddingDimension)
	t.Logf("metadata: %+v", a.Metadata)
}
