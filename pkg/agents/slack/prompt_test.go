package slack_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	slackagent "github.com/secmon-lab/warren/pkg/agents/slack"
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

	// Check that _slack_context is NOT in parameters (internal only)
	_, hasSlackContext := params["_slack_context"]
	gt.False(t, hasSlackContext)

	t.Run("render with slack context", func(t *testing.T) {
		rendered, err := template.Render(map[string]any{
			"request":        "find messages about security alerts",
			"_slack_context": "Current Slack context: channel_id=C67890, thread_ts=123.456, team_id=T12345",
		})
		gt.NoError(t, err)
		gt.S(t, rendered).Contains("Current Slack Context")
		gt.S(t, rendered).Contains("channel_id=C67890")
		gt.S(t, rendered).Contains("find messages about security alerts")
	})

	t.Run("render without slack context", func(t *testing.T) {
		rendered, err := template.Render(map[string]any{
			"request": "find messages about security alerts",
		})
		gt.NoError(t, err)
		gt.S(t, rendered).NotContains("Current Slack Context")
		gt.S(t, rendered).Contains("find messages about security alerts")
	})
}

