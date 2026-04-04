package slack

import (
	_ "embed"

	"github.com/m-mizutani/gollem"
)

//go:embed prompt/system.md
var systemPromptTemplate string

// buildSystemPrompt builds system prompt (for factory)
func buildSystemPrompt() (string, error) {
	return systemPromptTemplate, nil
}

// newPromptTemplate creates a PromptTemplate for the SubAgent
func newPromptTemplate() (*gollem.PromptTemplate, error) {
	return gollem.NewPromptTemplate(
		"{{if ._slack_context}}## Current Slack Context\n{{._slack_context}}\nYou are being invoked from within this Slack context. When the user refers to \"this channel\" or \"here\", they mean the channel above. You can use the channel ID to scope your searches, for example: `in:C12345678`.\n\n{{end}}{{.request}}",
		map[string]*gollem.Parameter{
			"request": {
				Type:        gollem.TypeString,
				Description: "DO NOT specify search keywords or terms. Describe ONLY the concept/situation in natural language. The agent will determine all search keywords and variations. ✗ BAD: 'search for authentication keyword', 'messages containing auth error', 'find keyword login' ✓ GOOD: 'people having authentication problems', 'discussions about performance issues', 'error reports in #security-alerts channel'. Include: (1) What concept/situation to find (NOT keywords), (2) Time period if relevant, (3) Channel/user scope if relevant. The Slack agent handles all keyword selection, variations, and multilingual terms automatically.",
				Required:    true,
			},
			"limit": {
				Type:        gollem.TypeNumber,
				Description: "Maximum number of messages to return in the response (default: 50, max: 200). Use this to control response size.",
			},
		},
	)
}
