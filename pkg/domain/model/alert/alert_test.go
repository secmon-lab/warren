package alert_test

import (
	"encoding/json"
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

func TestAttribute_UnmarshalJSON(t *testing.T) {
	t.Run("handles string values", func(t *testing.T) {
		data := `{"key": "severity", "value": "high", "link": "http://example.com", "auto": true}`
		var attr alert.Attribute
		gt.NoError(t, json.Unmarshal([]byte(data), &attr))
		gt.Equal(t, attr.Key, "severity")
		gt.Equal(t, attr.Value, "high")
		gt.Equal(t, attr.Link, "http://example.com")
		gt.True(t, attr.Auto)
	})

	t.Run("handles numeric values", func(t *testing.T) {
		data := `{"key": "count", "value": 42, "link": "", "auto": false}`
		var attr alert.Attribute
		gt.NoError(t, json.Unmarshal([]byte(data), &attr))
		gt.Equal(t, attr.Key, "count")
		gt.Equal(t, attr.Value, "42")
		gt.Equal(t, attr.Link, "")
		gt.False(t, attr.Auto)
	})

	t.Run("handles float values", func(t *testing.T) {
		data := `{"key": "score", "value": 3.14, "link": "", "auto": false}`
		var attr alert.Attribute
		gt.NoError(t, json.Unmarshal([]byte(data), &attr))
		gt.Equal(t, attr.Key, "score")
		gt.Equal(t, attr.Value, "3.14")
	})

	t.Run("handles boolean values", func(t *testing.T) {
		data := `{"key": "is_critical", "value": true, "link": "", "auto": false}`
		var attr alert.Attribute
		gt.NoError(t, json.Unmarshal([]byte(data), &attr))
		gt.Equal(t, attr.Key, "is_critical")
		gt.Equal(t, attr.Value, "true")
	})

	t.Run("handles null values", func(t *testing.T) {
		data := `{"key": "empty", "value": null, "link": "", "auto": false}`
		var attr alert.Attribute
		gt.NoError(t, json.Unmarshal([]byte(data), &attr))
		gt.Equal(t, attr.Key, "empty")
		gt.Equal(t, attr.Value, "")
	})
}
