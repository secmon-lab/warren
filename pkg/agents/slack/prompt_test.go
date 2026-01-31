package slack_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	slackagent "github.com/secmon-lab/warren/pkg/agents/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/memory"
)

func TestBuildSystemPrompt(t *testing.T) {
	prompt, err := slackagent.ExportedBuildSystemPrompt()
	gt.NoError(t, err)
	gt.V(t, prompt).NotEqual("")
	gt.True(t, len(prompt) > 0)
}

func TestNewPromptTemplate(t *testing.T) {
	template, err := slackagent.ExportedNewPromptTemplate()
	gt.NoError(t, err)
	gt.V(t, template).NotNil()

	// Check that template has expected parameters
	params := template.Parameters()
	gt.V(t, len(params)).NotEqual(0)

	// Check request parameter exists and is required
	requestParam, hasRequest := params["request"]
	gt.True(t, hasRequest)
	gt.V(t, requestParam).NotNil()
	gt.True(t, requestParam.Required)
	gt.V(t, requestParam.Type).Equal("string")

	// Check limit parameter exists and is optional
	limitParam, hasLimit := params["limit"]
	gt.True(t, hasLimit)
	gt.V(t, limitParam).NotNil()
	gt.False(t, limitParam.Required)
	gt.V(t, limitParam.Type).Equal("number")

	// Check that _memory_context is NOT in parameters (internal only)
	_, hasMemoryContext := params["_memory_context"]
	gt.False(t, hasMemoryContext)
}

func TestFormatMemoryContext(t *testing.T) {
	t.Run("empty memories", func(t *testing.T) {
		result := slackagent.ExportedFormatMemoryContext(nil)
		gt.V(t, result).Equal("")

		result = slackagent.ExportedFormatMemoryContext([]*memory.AgentMemory{})
		gt.V(t, result).Equal("")
	})

	t.Run("single memory", func(t *testing.T) {
		memories := []*memory.AgentMemory{
			{
				ID:      "mem-1",
				AgentID: "slack_search",
				Claim:   "Test claim A",
			},
		}
		result := slackagent.ExportedFormatMemoryContext(memories)
		gt.V(t, result).NotEqual("")
		gt.S(t, result).Contains("Past Experiences")
		gt.S(t, result).Contains("Experience A")
		gt.S(t, result).Contains("Test claim A")
	})

	t.Run("multiple memories", func(t *testing.T) {
		memories := []*memory.AgentMemory{
			{
				ID:      "mem-1",
				AgentID: "slack_search",
				Claim:   "First claim",
			},
			{
				ID:      "mem-2",
				AgentID: "slack_search",
				Claim:   "Second claim",
			},
			{
				ID:      "mem-3",
				AgentID: "slack_search",
				Claim:   "Third claim",
			},
		}
		result := slackagent.ExportedFormatMemoryContext(memories)
		gt.V(t, result).NotEqual("")
		gt.S(t, result).Contains("Experience A")
		gt.S(t, result).Contains("Experience B")
		gt.S(t, result).Contains("Experience C")
		gt.S(t, result).Contains("First claim")
		gt.S(t, result).Contains("Second claim")
		gt.S(t, result).Contains("Third claim")
	})
}
