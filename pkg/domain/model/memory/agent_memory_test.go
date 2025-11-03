package memory_test

import (
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/memory"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

func TestAgentMemory_Validate(t *testing.T) {
	t.Run("valid memory", func(t *testing.T) {
		mem := &memory.AgentMemory{
			ID:        types.NewAgentMemoryID(),
			AgentID:   "bigquery",
			TaskQuery: "test query",
			Timestamp: time.Now(),
			Duration:  time.Second,
		}

		err := mem.Validate()
		gt.NoError(t, err)
	})

	t.Run("missing ID", func(t *testing.T) {
		mem := &memory.AgentMemory{
			AgentID:   "bigquery",
			TaskQuery: "test query",
		}

		err := mem.Validate()
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("ID is required")
	})

	t.Run("missing AgentID", func(t *testing.T) {
		mem := &memory.AgentMemory{
			ID:        types.NewAgentMemoryID(),
			TaskQuery: "test query",
		}

		err := mem.Validate()
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("agent ID is required")
	})

	t.Run("missing TaskQuery", func(t *testing.T) {
		mem := &memory.AgentMemory{
			ID:      types.NewAgentMemoryID(),
			AgentID: "bigquery",
		}

		err := mem.Validate()
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("task query is required")
	})
}

func TestAgentMemory_Structure(t *testing.T) {
	now := time.Now()
	embedding := []float32{0.1, 0.2, 0.3}

	mem := &memory.AgentMemory{
		ID:                 types.NewAgentMemoryID(),
		AgentID:            "bigquery",
		TaskQuery:          "get login errors",
		QueryEmbedding:     embedding,
		Timestamp:          now,
		Duration:           5 * time.Second,
		SuccessDescription: "Successfully executed 3 tool calls",
		SuccessResult: map[string]any{
			"tool_call_count": 3,
			"tools_used":      []string{"bigquery_query", "bigquery_result"},
		},
		Problems:     []string{"Query exceeded scan size limit"},
		Improvements: []string{"Add WHERE clause to reduce scan size"},
	}

	gt.V(t, mem.AgentID).Equal("bigquery")
	gt.V(t, mem.TaskQuery).Equal("get login errors")
	gt.V(t, len(mem.QueryEmbedding)).Equal(3)
	gt.V(t, mem.QueryEmbedding[0]).Equal(float32(0.1))
	gt.V(t, mem.Duration).Equal(5 * time.Second)
	gt.V(t, mem.SuccessDescription).Equal("Successfully executed 3 tool calls")
	gt.V(t, mem.SuccessResult["tool_call_count"]).Equal(3)
	gt.V(t, len(mem.Problems)).Equal(1)
	gt.V(t, mem.Problems[0]).Equal("Query exceeded scan size limit")
	gt.V(t, len(mem.Improvements)).Equal(1)
	gt.V(t, mem.Improvements[0]).Equal("Add WHERE clause to reduce scan size")
}
