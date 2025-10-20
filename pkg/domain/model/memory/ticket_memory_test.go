package memory_test

import (
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/memory"
)

func TestTicketMemory_Validate(t *testing.T) {
	now := time.Now()

	t.Run("valid memory", func(t *testing.T) {
		mem := &memory.TicketMemory{
			SchemaID:  "test-schema",
			Insights:  "organizational knowledge",
			CreatedAt: now,
			UpdatedAt: now,
			Version:   1,
		}

		err := mem.Validate()
		gt.NoError(t, err)
	})

	t.Run("missing schema_id", func(t *testing.T) {
		mem := &memory.TicketMemory{
			Insights:  "organizational knowledge",
			CreatedAt: now,
			UpdatedAt: now,
			Version:   1,
		}

		err := mem.Validate()
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("schema_id is required")
	})

	t.Run("empty insights", func(t *testing.T) {
		mem := &memory.TicketMemory{
			SchemaID:  "test-schema",
			CreatedAt: now,
			UpdatedAt: now,
			Version:   1,
		}

		err := mem.Validate()
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("insights is required")
	})

	t.Run("negative version", func(t *testing.T) {
		mem := &memory.TicketMemory{
			SchemaID:  "test-schema",
			Insights:  "organizational knowledge",
			CreatedAt: now,
			UpdatedAt: now,
			Version:   -1,
		}

		err := mem.Validate()
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("version must be non-negative")
	})
}

func TestTicketMemory_IsEmpty(t *testing.T) {
	t.Run("empty memory", func(t *testing.T) {
		mem := &memory.TicketMemory{
			SchemaID: "test-schema",
		}

		gt.True(t, mem.IsEmpty())
	})

	t.Run("non-empty memory", func(t *testing.T) {
		mem := &memory.TicketMemory{
			SchemaID: "test-schema",
			Insights: "organizational knowledge",
		}

		gt.False(t, mem.IsEmpty())
	})
}

func TestTicketMemory_String(t *testing.T) {
	t.Run("with insights", func(t *testing.T) {
		mem := &memory.TicketMemory{
			SchemaID: "test-schema",
			Insights: "organizational knowledge",
		}

		result := mem.String()
		gt.S(t, result).Contains("*Organizational Knowledge:*")
		gt.S(t, result).Contains("organizational knowledge")
	})

	t.Run("empty insights", func(t *testing.T) {
		mem := &memory.TicketMemory{
			SchemaID: "test-schema",
		}

		result := mem.String()
		gt.Equal(t, result, "")
	})
}
