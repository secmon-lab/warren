package gemini_test

import (
	"testing"

	"cloud.google.com/go/vertexai/genai"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/adapter/gemini"
)

func TestGeminiClient(t *testing.T) {
	ctx := t.Context()

	client := gemini.NewTestClient(t, gemini.WithResponseMIMEType("text/plain"))

	ssn := client.StartChat()
	resp, err := ssn.SendMessage(ctx, genai.Text("My color is blue. Please remember it."))
	gt.NoError(t, err)

	t.Log("resp", resp)

	history := ssn.GetHistory()
	gt.A(t, history).Longer(0)

	newSsn := client.StartChat()
	newSsn.SetHistory(history...)
	resp, err = newSsn.SendMessage(ctx, genai.Text("What is my color?"))
	gt.NoError(t, err)

	history = newSsn.GetHistory()
	gt.A(t, history).Longer(0)
	txt := gt.Cast[genai.Text](t, resp.Candidates[0].Content.Parts[0])
	gt.S(t, string(txt)).Contains("blue")
}
