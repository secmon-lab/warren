package slack

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/model/memory"
)

//go:embed prompt/system.md
var systemPromptTemplate string

// buildSystemPrompt builds system prompt without memories (for factory)
func buildSystemPrompt() (string, error) {
	// The system prompt template is now simple and doesn't need memories
	// Memories are injected via middleware into the prompt template
	return systemPromptTemplate, nil
}

// newPromptTemplate creates a PromptTemplate for the SubAgent
func newPromptTemplate() (*gollem.PromptTemplate, error) {
	return gollem.NewPromptTemplate(
		// Template can use both _memory_context and request
		// _memory_context is injected by middleware (not visible to LLM as a parameter)
		"{{if ._memory_context}}{{._memory_context}}\n\n{{end}}{{.request}}",
		map[string]*gollem.Parameter{
			// Only define parameters that LLM should know about
			"request": {
				Type:        gollem.TypeString,
				Description: "DO NOT specify search keywords or terms. Describe ONLY the concept/situation in natural language. The agent will determine all search keywords and variations. ✗ BAD: 'search for authentication keyword', 'messages containing auth error', 'find keyword login' ✓ GOOD: 'people having authentication problems', 'discussions about performance issues', 'error reports in #security-alerts channel'. Include: (1) What concept/situation to find (NOT keywords), (2) Time period if relevant, (3) Channel/user scope if relevant. The Slack agent handles all keyword selection, variations, and multilingual terms automatically.",
				Required:    true,
			},
			"limit": {
				Type:        gollem.TypeNumber,
				Description: "Maximum number of messages to return in the response (default: 50, max: 200). Use this to control response size.",
			},
			// _memory_context is NOT included - it's an internal parameter
		},
	)
}

// formatMemoryContext formats memories for injection into the prompt
func formatMemoryContext(memories []*memory.AgentMemory) string {
	if len(memories) == 0 {
		return ""
	}

	var buf strings.Builder
	buf.WriteString("# Past Experiences\n\n")
	buf.WriteString("You have access to relevant past experiences that may help with this task:\n\n")

	for i, mem := range memories {
		letter := string(rune('A' + i))
		fmt.Fprintf(&buf, "## Experience %s\n", letter)
		fmt.Fprintf(&buf, "**Claim:** %s\n", mem.Claim)
		buf.WriteString("\n")
	}

	return buf.String()
}
