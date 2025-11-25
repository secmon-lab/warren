package policy_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/policy"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

func TestEnrichTask_EnsureID(t *testing.T) {
	t.Run("generates ID when not set", func(t *testing.T) {
		task := policy.EnrichTask{
			Inline: "test prompt",
			Format: types.GenAIContentFormatText,
		}

		gt.Equal(t, task.ID, "")
		task.EnsureID()
		gt.NotEqual(t, task.ID, "")
		gt.True(t, len(task.ID) > 0)
	})

	t.Run("preserves existing ID", func(t *testing.T) {
		task := policy.EnrichTask{
			ID:     "existing-id",
			Inline: "test prompt",
			Format: types.GenAIContentFormatText,
		}

		task.EnsureID()
		gt.Equal(t, task.ID, "existing-id")
	})

	t.Run("generates unique IDs", func(t *testing.T) {
		task1 := policy.EnrichTask{Inline: "test1"}
		task2 := policy.EnrichTask{Inline: "test2"}

		task1.EnsureID()
		task2.EnsureID()

		gt.NotEqual(t, task1.ID, task2.ID)
	})
}

func TestEnrichTask_Validate(t *testing.T) {
	t.Run("valid with template file", func(t *testing.T) {
		task := policy.EnrichTask{
			ID:       "test-task",
			Template: "path/to/template.md",
			Format:   types.GenAIContentFormatText,
		}

		err := task.Validate()
		gt.NoError(t, err)
	})

	t.Run("valid with inline prompt", func(t *testing.T) {
		task := policy.EnrichTask{
			ID:     "test-task",
			Inline: "Analyze this alert",
			Format: types.GenAIContentFormatJSON,
		}

		err := task.Validate()
		gt.NoError(t, err)
	})

	t.Run("invalid with both template and inline", func(t *testing.T) {
		task := policy.EnrichTask{
			ID:       "test-task",
			Template: "path/to/template.md",
			Inline:   "Analyze this alert",
			Format:   types.GenAIContentFormatText,
		}

		err := task.Validate()
		gt.Error(t, err)
	})

	t.Run("invalid with neither template nor inline", func(t *testing.T) {
		task := policy.EnrichTask{
			ID:     "test-task",
			Format: types.GenAIContentFormatText,
		}

		err := task.Validate()
		gt.Error(t, err)
	})
}

func TestEnrichTask_HasTemplateFile(t *testing.T) {
	t.Run("returns true when template is set", func(t *testing.T) {
		task := policy.EnrichTask{
			Template: "path/to/template.md",
		}

		gt.True(t, task.HasTemplateFile())
	})

	t.Run("returns false when inline is set", func(t *testing.T) {
		task := policy.EnrichTask{
			Inline: "Analyze this alert",
		}

		gt.False(t, task.HasTemplateFile())
	})
}

func TestEnrichTask_GetPromptContent(t *testing.T) {
	t.Run("returns inline when set", func(t *testing.T) {
		task := policy.EnrichTask{
			Inline: "Analyze this alert",
		}

		gt.Equal(t, task.GetPromptContent(), "Analyze this alert")
	})

	t.Run("returns template path when inline is not set", func(t *testing.T) {
		task := policy.EnrichTask{
			Template: "path/to/template.md",
		}

		gt.Equal(t, task.GetPromptContent(), "path/to/template.md")
	})
}

func TestEnrichPolicyResult_TaskCount(t *testing.T) {
	t.Run("counts prompt tasks", func(t *testing.T) {
		result := policy.EnrichPolicyResult{
			Prompts: []policy.EnrichTask{
				{ID: "p1"},
				{ID: "p2"},
				{ID: "p3"},
			},
		}

		gt.Equal(t, result.TaskCount(), 3)
	})

	t.Run("returns 0 for empty result", func(t *testing.T) {
		result := policy.EnrichPolicyResult{}

		gt.Equal(t, result.TaskCount(), 0)
	})
}

func TestEnrichPolicyResult_EnsureTaskIDs(t *testing.T) {
	t.Run("ensures IDs for all tasks", func(t *testing.T) {
		result := policy.EnrichPolicyResult{
			Prompts: []policy.EnrichTask{
				{Inline: "prompt1"},
				{ID: "existing", Inline: "prompt2"},
				{Inline: "prompt3"},
			},
		}

		result.EnsureTaskIDs()

		// Check all tasks have IDs
		gt.NotEqual(t, result.Prompts[0].ID, "")
		gt.Equal(t, result.Prompts[1].ID, "existing")
		gt.NotEqual(t, result.Prompts[2].ID, "")
	})
}
