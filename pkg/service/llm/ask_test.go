package llm_test

import (
	"os"
	"testing"

	"github.com/m-mizutani/gollem/llm/gemini"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/service/llm"
)

func TestAsk(t *testing.T) {
	projectID, ok := os.LookupEnv("TEST_GEMINI_PROJECT_ID")
	if !ok {
		t.Skip("GEMINI_PROJECT_ID is not set")
	}
	location, ok := os.LookupEnv("TEST_GEMINI_LOCATION")
	if !ok {
		t.Skip("GEMINI_LOCATION is not set")
	}

	ctx := t.Context()
	geminiClient, err := gemini.New(ctx, projectID, location,
		gemini.WithThinkingBudget(0), // Disable thinking feature
	)
	gt.NoError(t, err).Required()

	type resp struct {
		Message string `json:"message"`
	}

	prompt := `Reply for 'Hello, world!'. Format is following:
	{
		"message": "Hello, world!"
	}`

	result, err := llm.Ask[resp](ctx, geminiClient, prompt)
	gt.NoError(t, err)
	gt.NotNil(t, result)
	gt.S(t, result.Message).NotEqual("")
}
