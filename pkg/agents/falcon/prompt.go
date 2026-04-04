package falcon

import (
	_ "embed"
)

//go:embed prompt/system.md
var systemPromptTemplate string

// buildSystemPrompt returns the system prompt for the Falcon agent.
func buildSystemPrompt() string {
	return systemPromptTemplate
}
