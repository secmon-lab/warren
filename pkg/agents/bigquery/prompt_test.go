package bigquery_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	bqagent "github.com/secmon-lab/warren/pkg/agents/bigquery"
	"github.com/secmon-lab/warren/pkg/domain/model/memory"
)

func TestBuildSystemPrompt(t *testing.T) {
	config := &bqagent.Config{
		Tables: []bqagent.TableConfig{
			{
				ProjectID:   "test-project",
				DatasetID:   "test-dataset",
				TableID:     "test-table",
				Description: "Test table",
			},
		},
		ScanSizeLimit: 1000000,
	}

	prompt, err := bqagent.ExportedBuildSystemPrompt(config)
	gt.NoError(t, err)
	gt.V(t, prompt).NotEqual("")
	gt.True(t, len(prompt) > 0)
	gt.S(t, prompt).Contains("test-project")
	gt.S(t, prompt).Contains("test-dataset")
	gt.S(t, prompt).Contains("test-table")
}

func TestNewPromptTemplate(t *testing.T) {
	template, err := bqagent.ExportedNewPromptTemplate()
	gt.NoError(t, err)
	gt.V(t, template).NotNil()

	// Check that template has expected parameters
	params := template.Parameters()
	gt.V(t, len(params)).NotEqual(0)

	// Check query parameter exists and is required
	queryParam, hasQuery := params["query"]
	gt.True(t, hasQuery)
	gt.V(t, queryParam).NotNil()
	gt.True(t, queryParam.Required)
	gt.V(t, queryParam.Type).Equal("string")

	// Check that _memory_context is NOT in parameters (internal only)
	_, hasMemoryContext := params["_memory_context"]
	gt.False(t, hasMemoryContext)
}

func TestFormatMemoryContext(t *testing.T) {
	t.Run("empty memories", func(t *testing.T) {
		result := bqagent.ExportedFormatMemoryContext(nil)
		gt.V(t, result).Equal("")

		result = bqagent.ExportedFormatMemoryContext([]*memory.AgentMemory{})
		gt.V(t, result).Equal("")
	})

	t.Run("single memory", func(t *testing.T) {
		memories := []*memory.AgentMemory{
			{
				ID:      "mem-1",
				AgentID: "bigquery",
				Claim:   "Test claim A",
			},
		}
		result := bqagent.ExportedFormatMemoryContext(memories)
		gt.V(t, result).NotEqual("")
		gt.S(t, result).Contains("Past Experiences")
		gt.S(t, result).Contains("Experience A")
		gt.S(t, result).Contains("Test claim A")
	})

	t.Run("multiple memories", func(t *testing.T) {
		memories := []*memory.AgentMemory{
			{
				ID:      "mem-1",
				AgentID: "bigquery",
				Claim:   "First claim",
			},
			{
				ID:      "mem-2",
				AgentID: "bigquery",
				Claim:   "Second claim",
			},
			{
				ID:      "mem-3",
				AgentID: "bigquery",
				Claim:   "Third claim",
			},
		}
		result := bqagent.ExportedFormatMemoryContext(memories)
		gt.V(t, result).NotEqual("")
		gt.S(t, result).Contains("Experience A")
		gt.S(t, result).Contains("Experience B")
		gt.S(t, result).Contains("Experience C")
		gt.S(t, result).Contains("First claim")
		gt.S(t, result).Contains("Second claim")
		gt.S(t, result).Contains("Third claim")
	})
}
