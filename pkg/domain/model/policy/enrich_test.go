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
	t.Run("valid with prompt file", func(t *testing.T) {
		task := policy.EnrichTask{
			ID:     "test-task",
			Prompt: "path/to/prompt.md",
			Format: types.GenAIContentFormatText,
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

	t.Run("invalid with both prompt and inline", func(t *testing.T) {
		task := policy.EnrichTask{
			ID:     "test-task",
			Prompt: "path/to/prompt.md",
			Inline: "Analyze this alert",
			Format: types.GenAIContentFormatText,
		}

		err := task.Validate()
		gt.Error(t, err)
	})

	t.Run("invalid with neither prompt nor inline", func(t *testing.T) {
		task := policy.EnrichTask{
			ID:     "test-task",
			Format: types.GenAIContentFormatText,
		}

		err := task.Validate()
		gt.Error(t, err)
	})
}

func TestEnrichTask_HasPromptFile(t *testing.T) {
	t.Run("returns true when prompt is set", func(t *testing.T) {
		task := policy.EnrichTask{
			Prompt: "path/to/prompt.md",
		}

		gt.True(t, task.HasPromptFile())
	})

	t.Run("returns false when inline is set", func(t *testing.T) {
		task := policy.EnrichTask{
			Inline: "Analyze this alert",
		}

		gt.False(t, task.HasPromptFile())
	})
}

func TestEnrichTask_GetPromptContent(t *testing.T) {
	t.Run("returns inline when set", func(t *testing.T) {
		task := policy.EnrichTask{
			Inline: "Analyze this alert",
		}

		gt.Equal(t, task.GetPromptContent(), "Analyze this alert")
	})

	t.Run("returns prompt path when inline is not set", func(t *testing.T) {
		task := policy.EnrichTask{
			Prompt: "path/to/prompt.md",
		}

		gt.Equal(t, task.GetPromptContent(), "path/to/prompt.md")
	})
}

func TestEnrichPolicyResult_TaskCount(t *testing.T) {
	t.Run("counts query and agent tasks", func(t *testing.T) {
		result := policy.EnrichPolicyResult{
			Query: []policy.QueryTask{
				{EnrichTask: policy.EnrichTask{ID: "q1"}},
				{EnrichTask: policy.EnrichTask{ID: "q2"}},
			},
			Agent: []policy.AgentTask{
				{EnrichTask: policy.EnrichTask{ID: "a1"}},
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
			Query: []policy.QueryTask{
				{EnrichTask: policy.EnrichTask{Inline: "query1"}},
				{EnrichTask: policy.EnrichTask{ID: "existing", Inline: "query2"}},
			},
			Agent: []policy.AgentTask{
				{EnrichTask: policy.EnrichTask{Inline: "agent1"}},
			},
		}

		result.EnsureTaskIDs()

		// Check all tasks have IDs
		gt.NotEqual(t, result.Query[0].ID, "")
		gt.Equal(t, result.Query[1].ID, "existing")
		gt.NotEqual(t, result.Agent[0].ID, "")
	})
}
