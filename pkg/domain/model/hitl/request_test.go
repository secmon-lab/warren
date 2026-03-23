package hitl_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/hitl"
)

func TestNewQuestionPayload(t *testing.T) {
	payload := hitl.NewQuestionPayload("Is this IP internal?", []string{"Yes", "No", "Unknown"})
	gt.Value(t, payload["question"]).Equal("Is this IP internal?")

	options, ok := payload["options"].([]string)
	gt.Value(t, ok).Equal(true)
	gt.Array(t, options).Length(3).Required()
	gt.Value(t, options[0]).Equal("Yes")
	gt.Value(t, options[1]).Equal("No")
	gt.Value(t, options[2]).Equal("Unknown")
}

func TestRequest_QuestionData(t *testing.T) {
	req := &hitl.Request{
		Payload: hitl.NewQuestionPayload("What is this?", []string{"A", "B"}),
	}
	q := req.QuestionData()
	gt.Value(t, q.Question).Equal("What is this?")
	gt.Array(t, q.Options).Length(2).Required()
	gt.Value(t, q.Options[0]).Equal("A")
	gt.Value(t, q.Options[1]).Equal("B")
}

func TestRequest_QuestionData_NilPayload(t *testing.T) {
	req := &hitl.Request{}
	q := req.QuestionData()
	gt.Value(t, q.Question).Equal("")
	gt.Array(t, q.Options).Length(0)
}

func TestRequest_ResponseAnswer(t *testing.T) {
	req := &hitl.Request{
		Response: map[string]any{"answer": "Yes, internal VPN"},
	}
	gt.Value(t, req.ResponseAnswer()).Equal("Yes, internal VPN")
}

func TestRequest_ResponseAnswer_NilResponse(t *testing.T) {
	req := &hitl.Request{}
	gt.Value(t, req.ResponseAnswer()).Equal("")
}

func TestRequest_ResponseComment(t *testing.T) {
	req := &hitl.Request{
		Response: map[string]any{"comment": "confirmed by ops team"},
	}
	gt.Value(t, req.ResponseComment()).Equal("confirmed by ops team")
}

func TestRequest_ToolApproval(t *testing.T) {
	req := &hitl.Request{
		Payload: hitl.NewToolApprovalPayload("web_fetch", map[string]any{"url": "https://example.com"}),
	}
	p := req.ToolApproval()
	gt.Value(t, p.ToolName).Equal("web_fetch")
	gt.Value(t, p.ToolArgs["url"]).Equal("https://example.com")
}
