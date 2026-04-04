package amber_test

import (
	"encoding/json"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/usecase/chat/amber"
)

func TestReplanResult_WithQuestion(t *testing.T) {
	raw := `{
		"message": "Need to ask about internal IP",
		"tasks": [],
		"question": {
			"question": "Is 10.0.0.5 an internal IP?",
			"options": ["Yes, VPN gateway", "Yes, dev server", "No", "Unknown"],
			"reason": "Could not determine IP ownership from available tools"
		}
	}`

	var result amber.ReplanResult
	gt.NoError(t, json.Unmarshal([]byte(raw), &result)).Required()
	gt.Value(t, result.Message).Equal("Need to ask about internal IP")
	gt.Value(t, result.Question != nil).Equal(true)
	gt.Value(t, result.Question.Question).Equal("Is 10.0.0.5 an internal IP?")
	gt.Array(t, result.Question.Options).Length(4).Required()
	gt.Value(t, result.Question.Options[0]).Equal("Yes, VPN gateway")
	gt.Value(t, result.Question.Reason).Equal("Could not determine IP ownership from available tools")
}

func TestReplanResult_WithoutQuestion(t *testing.T) {
	raw := `{
		"message": "Continuing investigation",
		"tasks": [{"id": "t1", "title": "Check logs", "description": "desc", "tools": ["bigquery"], "sub_agents": []}]
	}`

	var result amber.ReplanResult
	gt.NoError(t, json.Unmarshal([]byte(raw), &result)).Required()
	gt.Value(t, result.Question == nil).Equal(true)
	gt.Array(t, result.Tasks).Length(1).Required()
	gt.Value(t, result.Tasks[0].ID).Equal("t1")
}

func TestReplanResult_EmptyTasksNoQuestion(t *testing.T) {
	raw := `{"message": "Done", "tasks": []}`

	var result amber.ReplanResult
	gt.NoError(t, json.Unmarshal([]byte(raw), &result)).Required()
	gt.Value(t, result.Question == nil).Equal(true)
	gt.Array(t, result.Tasks).Length(0)
}

func TestReplanResult_QuestionAndTasks_QuestionTakesPriority(t *testing.T) {
	raw := `{
		"message": "Both set",
		"tasks": [{"id": "t1", "title": "task", "description": "d", "tools": [], "sub_agents": []}],
		"question": {
			"question": "Is this allowed?",
			"options": ["Yes", "No"],
			"reason": "Policy unclear"
		}
	}`

	var result amber.ReplanResult
	gt.NoError(t, json.Unmarshal([]byte(raw), &result)).Required()
	// Both are parsed, but the caller should prioritize question
	gt.Value(t, result.Question != nil).Equal(true)
	gt.Array(t, result.Tasks).Length(1)
}
