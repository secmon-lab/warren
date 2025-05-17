package test

import (
	"testing"

	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/llm/gemini"
)

func NewGeminiClient(t *testing.T) gollem.LLMClient {
	ctx := t.Context()
	vars := NewEnvVars(t, "TEST_GEMINI_PROJECT_ID", "TEST_GEMINI_LOCATION")

	llmClient, err := gemini.New(ctx, vars.Get("TEST_GEMINI_PROJECT_ID"), vars.Get("TEST_GEMINI_LOCATION"))
	if err != nil {
		t.Fatalf("failed to create gemini client: %v", err)
	}

	return llmClient
}
