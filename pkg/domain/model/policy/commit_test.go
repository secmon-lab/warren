package policy_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/policy"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

func TestCommitPolicyResult_ApplyTo(t *testing.T) {
	t.Run("applies title from commit policy", func(t *testing.T) {
		ctx := context.Background()
		a := alert.New(ctx, "test-schema", nil, alert.Metadata{
			Title:       "Original Title",
			Description: "Original Description",
		})

		result := policy.CommitPolicyResult{
			Title: "New Title from Policy",
		}

		result.ApplyTo(&a)

		gt.Equal(t, a.Metadata.Title, "New Title from Policy")
		gt.Equal(t, a.Metadata.TitleSource, types.SourcePolicy)
		gt.Equal(t, a.Metadata.Description, "Original Description") // unchanged
	})

	t.Run("applies description from commit policy", func(t *testing.T) {
		ctx := context.Background()
		a := alert.New(ctx, "test-schema", nil, alert.Metadata{
			Title:       "Original Title",
			Description: "Original Description",
		})

		result := policy.CommitPolicyResult{
			Description: "New Description from Policy",
		}

		result.ApplyTo(&a)

		gt.Equal(t, a.Metadata.Description, "New Description from Policy")
		gt.Equal(t, a.Metadata.DescriptionSource, types.SourcePolicy)
		gt.Equal(t, a.Metadata.Title, "Original Title") // unchanged
	})

	t.Run("applies channel from commit policy", func(t *testing.T) {
		ctx := context.Background()
		a := alert.New(ctx, "test-schema", nil, alert.Metadata{})

		result := policy.CommitPolicyResult{
			Channel: "security-alerts",
		}

		result.ApplyTo(&a)

		gt.Equal(t, a.Metadata.Channel, "security-alerts")
	})

	t.Run("applies attributes from commit policy", func(t *testing.T) {
		ctx := context.Background()
		a := alert.New(ctx, "test-schema", nil, alert.Metadata{
			Attributes: []alert.Attribute{
				{Key: "existing", Value: "attr1"},
			},
		})

		result := policy.CommitPolicyResult{
			Attr: []alert.Attribute{
				{Key: "severity", Value: "high"},
				{Key: "score", Value: "95", Link: "https://example.com"},
			},
		}

		result.ApplyTo(&a)

		gt.Equal(t, len(a.Metadata.Attributes), 3)

		// Check existing attribute is preserved
		gt.Equal(t, a.Metadata.Attributes[0].Key, "existing")
		gt.Equal(t, a.Metadata.Attributes[0].Value, "attr1")
		gt.False(t, a.Metadata.Attributes[0].Auto)

		// Check new attributes are added and marked as auto
		gt.Equal(t, a.Metadata.Attributes[1].Key, "severity")
		gt.Equal(t, a.Metadata.Attributes[1].Value, "high")
		gt.True(t, a.Metadata.Attributes[1].Auto)

		gt.Equal(t, a.Metadata.Attributes[2].Key, "score")
		gt.Equal(t, a.Metadata.Attributes[2].Value, "95")
		gt.Equal(t, a.Metadata.Attributes[2].Link, "https://example.com")
		gt.True(t, a.Metadata.Attributes[2].Auto)
	})

	t.Run("applies all fields from commit policy", func(t *testing.T) {
		ctx := context.Background()
		a := alert.New(ctx, "test-schema", nil, alert.Metadata{})

		result := policy.CommitPolicyResult{
			Title:       "Complete Title",
			Description: "Complete Description",
			Channel:     "alerts-channel",
			Attr: []alert.Attribute{
				{Key: "severity", Value: "critical"},
			},
		}

		result.ApplyTo(&a)

		gt.Equal(t, a.Metadata.Title, "Complete Title")
		gt.Equal(t, a.Metadata.TitleSource, types.SourcePolicy)
		gt.Equal(t, a.Metadata.Description, "Complete Description")
		gt.Equal(t, a.Metadata.DescriptionSource, types.SourcePolicy)
		gt.Equal(t, a.Metadata.Channel, "alerts-channel")
		gt.Equal(t, len(a.Metadata.Attributes), 1)
		gt.True(t, a.Metadata.Attributes[0].Auto)
	})

	t.Run("does not modify alert when result is empty", func(t *testing.T) {
		ctx := context.Background()
		a := alert.New(ctx, "test-schema", nil, alert.Metadata{
			Title:       "Original Title",
			Description: "Original Description",
			Attributes: []alert.Attribute{
				{Key: "existing", Value: "attr"},
			},
		})

		result := policy.CommitPolicyResult{}

		originalTitleSource := a.Metadata.TitleSource
		originalDescriptionSource := a.Metadata.DescriptionSource

		result.ApplyTo(&a)

		gt.Equal(t, a.Metadata.Title, "Original Title")
		gt.Equal(t, a.Metadata.TitleSource, originalTitleSource)
		gt.Equal(t, a.Metadata.Description, "Original Description")
		gt.Equal(t, a.Metadata.DescriptionSource, originalDescriptionSource)
		gt.Equal(t, a.Metadata.Channel, "")
		gt.Equal(t, len(a.Metadata.Attributes), 1)
	})

	t.Run("handles attributes with numeric values from JSON", func(t *testing.T) {
		// This tests that numeric values from OPA policy are correctly converted to strings
		ctx := context.Background()
		a := alert.New(ctx, "test-schema", nil, alert.Metadata{})

		// Simulate what would come from OPA policy JSON unmarshaling
		jsonData := `{
			"attr": [
				{"key": "count", "value": 42, "link": "", "auto": false},
				{"key": "score", "value": 3.14, "link": "", "auto": false},
				{"key": "is_critical", "value": true, "link": "", "auto": false}
			]
		}`

		var result policy.CommitPolicyResult
		err := json.Unmarshal([]byte(jsonData), &result)
		gt.NoError(t, err)

		result.ApplyTo(&a)

		gt.Equal(t, len(a.Metadata.Attributes), 3)

		// Verify numeric values are converted to strings
		gt.Equal(t, a.Metadata.Attributes[0].Key, "count")
		gt.Equal(t, a.Metadata.Attributes[0].Value, "42")
		gt.True(t, a.Metadata.Attributes[0].Auto) // Should be marked as auto

		gt.Equal(t, a.Metadata.Attributes[1].Key, "score")
		gt.Equal(t, a.Metadata.Attributes[1].Value, "3.14")
		gt.True(t, a.Metadata.Attributes[1].Auto)

		gt.Equal(t, a.Metadata.Attributes[2].Key, "is_critical")
		gt.Equal(t, a.Metadata.Attributes[2].Value, "true")
		gt.True(t, a.Metadata.Attributes[2].Auto)
	})
}
