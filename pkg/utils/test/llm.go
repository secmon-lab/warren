package test

import (
	"testing"

	"github.com/gollem-dev/gollem"
	"github.com/gollem-dev/gollem/llm/gemini"
)

func NewGeminiClient(t *testing.T) gollem.LLMClient {
	ctx := t.Context()
	vars := NewEnvVars(t, "TEST_GEMINI_PROJECT_ID", "TEST_GEMINI_LOCATION")

	llmClient, err := gemini.New(ctx, vars.Get("TEST_GEMINI_PROJECT_ID"), vars.Get("TEST_GEMINI_LOCATION"),
		gemini.WithModel("gemini-2.5-flash"),
		gemini.WithThinkingBudget(0), // Disable thinking feature
	)
	if err != nil {
		t.Fatalf("failed to create gemini client: %v", err)
	}

	return llmClient
}
