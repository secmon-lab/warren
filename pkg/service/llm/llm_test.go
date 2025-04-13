package llm_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/adapter/gemini"
	"github.com/secmon-lab/warren/pkg/service/llm"
)

func TestAsk(t *testing.T) {
	geminiClient := gemini.NewTestClient(t, gemini.WithContentType("application/json"))
	ssn := geminiClient.StartChat()

	type resp struct {
		Message string `json:"message"`
	}

	prompt := `Reply for 'Hello, world!'. Format is following:
	{
		"message": "Hello, world!"
	}`

	result, err := llm.Ask[resp](t.Context(), ssn.SendMessage, prompt)
	gt.NoError(t, err)
	gt.NotNil(t, result)
	gt.S(t, result.Message).NotEqual("")
}
