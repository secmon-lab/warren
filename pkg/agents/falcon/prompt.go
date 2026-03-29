package falcon

import (
	_ "embed"

	"github.com/m-mizutani/gollem"
)

//go:embed prompt/system.md
var systemPromptTemplate string

// buildSystemPrompt returns the system prompt for the Falcon agent.
func buildSystemPrompt() string {
	return systemPromptTemplate
}

// newPromptTemplate creates a PromptTemplate for the SubAgent.
func newPromptTemplate() (*gollem.PromptTemplate, error) {
	return gollem.NewPromptTemplate(
		"{{.request}}",
		map[string]*gollem.Parameter{
			"request": {
				Type:        gollem.TypeString,
				Description: "Describe what CrowdStrike Falcon data you need. Include incident/alert/behavior details, time ranges, severity filters, or specific IDs. Example: 'Find high-severity alerts from the last 24 hours', 'Get details for incident inc:abc123:def456'.",
				Required:    true,
			},
		},
	)
}
