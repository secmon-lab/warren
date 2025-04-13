package session_test

import (
	"context"
	"testing"

	"cloud.google.com/go/vertexai/genai"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

func TestHistory(t *testing.T) {
	ctx := context.Background()

	type testCase struct {
		name     string
		input    []*genai.Content
		expected *session.History
	}

	runTest := func(tc testCase) func(t *testing.T) {
		return func(t *testing.T) {
			history := session.NewHistory(ctx, types.NewSessionID(), tc.input)

			// Exclude ID and CreatedAt from comparison as they are generated at runtime
			tc.expected.ID = history.ID
			tc.expected.CreatedAt = history.CreatedAt
			tc.expected.SessionID = history.SessionID

			gt.Value(t, history).Equal(tc.expected)

			// Test ToContents method
			contents := history.ToContents()
			gt.Value(t, contents).Equal(tc.input)
		}
	}

	t.Run("Text only content", runTest(testCase{
		name: "Text only content",
		input: []*genai.Content{
			{
				Role: "user",
				Parts: []genai.Part{
					genai.Text("Hello"),
				},
			},
		},
		expected: &session.History{
			Contents: session.Contents{
				{
					Role: "user",
					Parts: []session.Part{
						{
							Text: "Hello",
						},
					},
				},
			},
		},
	}))

	t.Run("Content with multiple parts", runTest(testCase{
		name: "Content with multiple parts",
		input: []*genai.Content{
			{
				Role: "assistant",
				Parts: []genai.Part{
					genai.Text("Hello"),
					genai.Blob{Data: []byte("test")},
				},
			},
		},
		expected: &session.History{
			Contents: session.Contents{
				{
					Role: "assistant",
					Parts: []session.Part{
						{
							Text: "Hello",
						},
						{
							Blob: []byte("test"),
						},
					},
				},
			},
		},
	}))

	t.Run("Content with function call", runTest(testCase{
		name: "Content with function call",
		input: []*genai.Content{
			{
				Role: "assistant",
				Parts: []genai.Part{
					&genai.FunctionCall{
						Name: "test_func",
						Args: map[string]interface{}{
							"arg1": "value1",
						},
					},
				},
			},
		},
		expected: &session.History{
			Contents: session.Contents{
				{
					Role: "assistant",
					Parts: []session.Part{
						{
							FuncCall: &genai.FunctionCall{
								Name: "test_func",
								Args: map[string]interface{}{
									"arg1": "value1",
								},
							},
						},
					},
				},
			},
		},
	}))
}
