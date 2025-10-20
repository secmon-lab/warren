package memory_test

import (
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/memory"
)

func TestExecutionMemory_Validate(t *testing.T) {
	now := time.Now()

	t.Run("valid memory with all fields", func(t *testing.T) {
		mem := &memory.ExecutionMemory{
			SchemaID:  "test-schema",
			Keep:      "successful patterns",
			Change:    "areas to improve",
			Notes:     "other insights",
			CreatedAt: now,
			UpdatedAt: now,
			Version:   1,
		}

		err := mem.Validate()
		gt.NoError(t, err)
	})

	t.Run("valid memory with only Keep", func(t *testing.T) {
		mem := &memory.ExecutionMemory{
			SchemaID:  "test-schema",
			Keep:      "successful patterns",
			CreatedAt: now,
			UpdatedAt: now,
			Version:   1,
		}

		err := mem.Validate()
		gt.NoError(t, err)
	})

	t.Run("valid memory with only Change", func(t *testing.T) {
		mem := &memory.ExecutionMemory{
			SchemaID:  "test-schema",
			Change:    "areas to improve",
			CreatedAt: now,
			UpdatedAt: now,
			Version:   1,
		}

		err := mem.Validate()
		gt.NoError(t, err)
	})

	t.Run("valid memory with only Notes", func(t *testing.T) {
		mem := &memory.ExecutionMemory{
			SchemaID:  "test-schema",
			Notes:     "other insights",
			CreatedAt: now,
			UpdatedAt: now,
			Version:   1,
		}

		err := mem.Validate()
		gt.NoError(t, err)
	})

	t.Run("missing schema_id", func(t *testing.T) {
		mem := &memory.ExecutionMemory{
			Keep:      "successful patterns",
			CreatedAt: now,
			UpdatedAt: now,
			Version:   1,
		}

		err := mem.Validate()
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("schema_id is required")
	})

	t.Run("all content fields empty", func(t *testing.T) {
		mem := &memory.ExecutionMemory{
			SchemaID:  "test-schema",
			CreatedAt: now,
			UpdatedAt: now,
			Version:   1,
		}

		err := mem.Validate()
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("at least one of keep, change, or notes is required")
	})

	t.Run("negative version", func(t *testing.T) {
		mem := &memory.ExecutionMemory{
			SchemaID:  "test-schema",
			Keep:      "successful patterns",
			CreatedAt: now,
			UpdatedAt: now,
			Version:   -1,
		}

		err := mem.Validate()
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("version must be non-negative")
	})
}

func TestExecutionMemory_IsEmpty(t *testing.T) {
	t.Run("empty memory", func(t *testing.T) {
		mem := &memory.ExecutionMemory{
			SchemaID: "test-schema",
		}

		gt.True(t, mem.IsEmpty())
	})

	t.Run("non-empty with Keep", func(t *testing.T) {
		mem := &memory.ExecutionMemory{
			SchemaID: "test-schema",
			Keep:     "successful patterns",
		}

		gt.False(t, mem.IsEmpty())
	})

	t.Run("non-empty with Change", func(t *testing.T) {
		mem := &memory.ExecutionMemory{
			SchemaID: "test-schema",
			Change:   "areas to improve",
		}

		gt.False(t, mem.IsEmpty())
	})

	t.Run("non-empty with Notes", func(t *testing.T) {
		mem := &memory.ExecutionMemory{
			SchemaID: "test-schema",
			Notes:    "other insights",
		}

		gt.False(t, mem.IsEmpty())
	})
}

func TestExecutionMemory_String(t *testing.T) {
	t.Run("all fields present", func(t *testing.T) {
		mem := &memory.ExecutionMemory{
			SchemaID: "test-schema",
			Keep:     "successful patterns",
			Change:   "areas to improve",
			Notes:    "other insights",
		}

		result := mem.String()
		gt.S(t, result).Contains("*Keep (Successful Patterns):*")
		gt.S(t, result).Contains("successful patterns")
		gt.S(t, result).Contains("*Change (Areas for Improvement):*")
		gt.S(t, result).Contains("areas to improve")
		gt.S(t, result).Contains("*Notes (Other Insights):*")
		gt.S(t, result).Contains("other insights")
	})

	t.Run("only Keep", func(t *testing.T) {
		mem := &memory.ExecutionMemory{
			SchemaID: "test-schema",
			Keep:     "successful patterns",
		}

		result := mem.String()
		gt.S(t, result).Contains("*Keep (Successful Patterns):*")
		gt.S(t, result).Contains("successful patterns")
		gt.S(t, result).NotContains("*Change")
		gt.S(t, result).NotContains("*Notes")
	})

	t.Run("only Change", func(t *testing.T) {
		mem := &memory.ExecutionMemory{
			SchemaID: "test-schema",
			Change:   "areas to improve",
		}

		result := mem.String()
		gt.S(t, result).Contains("*Change (Areas for Improvement):*")
		gt.S(t, result).Contains("areas to improve")
		gt.S(t, result).NotContains("*Keep")
		gt.S(t, result).NotContains("*Notes")
	})

	t.Run("empty memory", func(t *testing.T) {
		mem := &memory.ExecutionMemory{
			SchemaID: "test-schema",
		}

		result := mem.String()
		gt.Equal(t, result, "")
	})
}
