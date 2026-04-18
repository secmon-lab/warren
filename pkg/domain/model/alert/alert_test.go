package alert_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/m-mizutani/gollem"
	gollem_mock "github.com/m-mizutani/gollem/mock"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/tag"
	"github.com/secmon-lab/warren/pkg/utils/embedding"
	"github.com/secmon-lab/warren/pkg/utils/test"
)

func TestAlert_FillMetadata(t *testing.T) {
	client := test.NewGeminiClient(t)

	a := alert.New(t.Context(), "test.alert.v1", map[string]any{
		"foo": "bar",
		"baz": 123,
	}, alert.Metadata{})

	gt.NoError(t, a.FillMetadata(t.Context(), client, nil))
	gt.NotEqual(t, a.Title, "")
	gt.NotEqual(t, a.Description, "")
	gt.Equal(t, len(a.Embedding), embedding.EmbeddingDimension)
	t.Logf("metadata: %+v", a.Metadata)
}

func TestAlert_FillMetadata_TagInference(t *testing.T) {
	// Mock LLM that records the prompt it receives and returns a fixed
	// metadata JSON including both known and unknown tag names.
	var receivedPrompt string
	mockLLM := &gollem_mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
			return &gollem_mock.SessionMock{
				GenerateFunc: func(ctx context.Context, input []gollem.Input, _ ...gollem.GenerateOption) (*gollem.Response, error) {
					if txt, ok := input[0].(gollem.Text); ok {
						receivedPrompt = string(txt)
					}
					return &gollem.Response{
						Texts: []string{`{
							"title": "Generated title",
							"description": "Generated description",
							"attributes": [],
							"tags": ["malware", "unknown-tag", "network"]
						}`},
					}, nil
				},
			}, nil
		},
		GenerateEmbeddingFunc: func(ctx context.Context, dimension int, input []string) ([][]float64, error) {
			result := make([][]float64, len(input))
			for i := range input {
				result[i] = []float64{0.1, 0.2, 0.3}
			}
			return result, nil
		},
	}

	available := []*tag.Tag{
		{ID: tag.NewID(), Name: "malware", Description: "Malware-related events"},
		{ID: tag.NewID(), Name: "network", Description: "Network-related events"},
		{ID: tag.NewID(), Name: "phishing", Description: "Phishing-related events"},
	}

	a := alert.New(t.Context(), "test.alert.v1", map[string]any{
		"event": "suspicious_binary_download",
	}, alert.Metadata{
		Tags: []string{"policy-origin"}, // pre-existing policy-provided tag
	})

	gt.NoError(t, a.FillMetadata(t.Context(), mockLLM, available))

	// Title / description from LLM response
	gt.Equal(t, a.Title, "Generated title")
	gt.Equal(t, a.Description, "Generated description")

	// Tags: policy-origin preserved, malware+network added, unknown-tag filtered out
	gt.A(t, a.Tags).Length(3).Equal([]string{"policy-origin", "malware", "network"})

	// Prompt must include the available tags so the LLM can choose from them
	gt.True(t, strings.Contains(receivedPrompt, "malware"))
	gt.True(t, strings.Contains(receivedPrompt, "phishing"))
	gt.True(t, strings.Contains(receivedPrompt, "Network-related events"))
}

func TestFormatAvailableTags(t *testing.T) {
	t.Run("empty input yields an explicit no-tags notice", func(t *testing.T) {
		got := alert.FormatAvailableTags(nil)
		gt.True(t, strings.Contains(got, "No tags are registered"))
		gt.True(t, strings.Contains(got, "empty array"))
	})

	t.Run("nil and empty-name entries are skipped, the rest become a bullet list", func(t *testing.T) {
		got := alert.FormatAvailableTags([]*tag.Tag{
			nil,
			{ID: "x", Name: ""},
			{ID: tag.NewID(), Name: "malware", Description: "Malware-related events"},
			{ID: tag.NewID(), Name: "network"}, // description empty -> name only
		})
		gt.Equal(t, got, "- `malware`: Malware-related events\n- `network`\n")
	})
}

func TestMergeInferredTags(t *testing.T) {
	t.Run("no inferred tags returns existing unchanged", func(t *testing.T) {
		existing := []string{"a", "b"}
		got := alert.MergeInferredTags(existing, nil, map[string]bool{"a": true, "c": true})
		gt.A(t, got).Length(2).Equal([]string{"a", "b"})
	})

	t.Run("no allowed set returns existing unchanged", func(t *testing.T) {
		existing := []string{"a"}
		got := alert.MergeInferredTags(existing, []string{"c", "d"}, map[string]bool{})
		gt.A(t, got).Length(1).Equal([]string{"a"})
	})

	t.Run("appends only allowed inferred tags", func(t *testing.T) {
		allowed := map[string]bool{"security": true, "malware": true, "network": true}
		got := alert.MergeInferredTags(
			[]string{"policy"},
			[]string{"malware", "unknown", "network"},
			allowed,
		)
		// policy is preserved even though not in allowed (it was already set by upstream)
		gt.A(t, got).Length(3).Equal([]string{"policy", "malware", "network"})
	})

	t.Run("deduplicates against existing", func(t *testing.T) {
		allowed := map[string]bool{"security": true, "malware": true}
		got := alert.MergeInferredTags(
			[]string{"security"},
			[]string{"security", "malware"},
			allowed,
		)
		gt.A(t, got).Length(2).Equal([]string{"security", "malware"})
	})

	t.Run("ignores empty tag names", func(t *testing.T) {
		allowed := map[string]bool{"malware": true}
		got := alert.MergeInferredTags(nil, []string{"", "malware", ""}, allowed)
		gt.A(t, got).Length(1).Equal([]string{"malware"})
	})
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
