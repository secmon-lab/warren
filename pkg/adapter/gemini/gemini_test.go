package gemini_test

import (
	"testing"

	"cloud.google.com/go/vertexai/genai"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/adapter/gemini"
	model "github.com/secmon-lab/warren/pkg/domain/model/gemini"
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

func TestFunctionCall(t *testing.T) {
	ctx := t.Context()

	tools := []*genai.Tool{
		{
			FunctionDeclarations: []*genai.FunctionDeclaration{
				{
					Name:        "get_user_info",
					Description: "Get user info",
					Parameters: &genai.Schema{
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"user_name": {
								Type:        genai.TypeString,
								Description: "The name of the user",
							},
						},
					},
				},
			},
		},
	}

	toolConfig := &genai.ToolConfig{
		FunctionCallingConfig: &genai.FunctionCallingConfig{
			Mode: genai.FunctionCallingAuto,
		},
	}

	client := gemini.NewTestClient(t)

	ssn := client.StartChat(
		model.WithModel("gemini-2.0-flash"),
		model.WithTools(tools),
		model.WithToolConfig(toolConfig),
		model.WithContentType("text/plain"),
	)
	resp, err := ssn.SendMessage(ctx, genai.Text("Get the user info of the user who has the name 'John'"))
	gt.NoError(t, err).Must()
	gt.NotNil(t, resp)
	gt.A(t, resp.Candidates).Longer(0).At(0, func(t testing.TB, v *genai.Candidate) {
		gt.A(t, v.Content.Parts).Longer(0).At(0, func(t testing.TB, v genai.Part) {
			call := gt.Cast[genai.FunctionCall](t, v)
			gt.S(t, call.Name).Equal("get_user_info")
			gt.M(t, call.Args).HaveKey("user_name")
		})
	})

	resp, err = ssn.SendMessage(ctx, genai.Text("User data is following: {user_name: 'John', role: 'admin', age: 30}. Describe the user info."))
	gt.NoError(t, err).Must()
	gt.NotNil(t, resp)
	gt.A(t, resp.Candidates).Longer(0).At(0, func(t testing.TB, v *genai.Candidate) {
		gt.A(t, v.Content.Parts).Longer(0).At(0, func(t testing.TB, v genai.Part) {
			txt := gt.Cast[genai.Text](t, v)
			gt.S(t, string(txt)).Contains("John")
		})
	})
}
