package falcon

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/model/memory"
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
		"{{if ._memory_context}}{{._memory_context}}\n\n{{end}}{{.request}}",
		map[string]*gollem.Parameter{
			"request": {
				Type:        gollem.TypeString,
				Description: "Describe what CrowdStrike Falcon data you need. Include incident/alert/behavior details, time ranges, severity filters, or specific IDs. Example: 'Find high-severity alerts from the last 24 hours', 'Get details for incident inc:abc123:def456'.",
				Required:    true,
			},
		},
	)
}

// formatMemoryContext formats memories for injection into the prompt.
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
