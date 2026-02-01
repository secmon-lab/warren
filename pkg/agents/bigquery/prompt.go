package bigquery

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
	"text/template"

	"github.com/dustin/go-humanize"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/model/memory"
)

//go:embed prompt/base.md
var basePrompt string

//go:embed prompt/system_with_memories.md
var systemWithMemoriesTemplate string

//go:embed prompt/runbooks.md
var runbooksTemplate string

//go:embed prompt/tool_description.md
var toolDescriptionTemplate string

var systemWithMemoriesTmpl *template.Template
var runbooksTmpl *template.Template
var toolDescriptionTmpl *template.Template

func init() {
	systemWithMemoriesTmpl = template.Must(template.New("system_with_memories").Parse(systemWithMemoriesTemplate))
	runbooksTmpl = template.Must(template.New("runbooks").Parse(runbooksTemplate))
	toolDescriptionTmpl = template.Must(template.New("tool_description").Parse(toolDescriptionTemplate))
}

// promptData represents the data for system prompt template
type promptData struct {
	Tables      []TableConfig
	HasMemories bool
	Memories    []*memory.AgentMemory
	Letters     []string
	Runbooks    map[string]interface{}
}

// newPromptTemplate creates a PromptTemplate for the SubAgent
func newPromptTemplate() (*gollem.PromptTemplate, error) {
	return gollem.NewPromptTemplate(
		// Template can use both _memory_context and query
		// _memory_context is injected by middleware (not visible to LLM as a parameter)
		"{{if ._memory_context}}{{._memory_context}}\n\n{{end}}{{.query}}",
		map[string]*gollem.Parameter{
			// Only define parameters that LLM should know about
			"query": {
				Type:        gollem.TypeString,
				Description: "ONLY specify the conditions for data retrieval (e.g., 'records containing package name X from the last 7 days', 'login events in the past week'). Do NOT include analysis instructions, interpretation requests, or questions - ONLY data retrieval conditions.",
				Required:    true,
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
		buf.WriteString(fmt.Sprintf("## Experience %s\n", letter))
		buf.WriteString(fmt.Sprintf("**Claim:** %s\n", mem.Claim))
		buf.WriteString("\n")
	}

	return buf.String()
}

// promptHintData represents the data for tool_description.md template
type promptHintData struct {
	HasTables     bool
	Tables        []TableConfig
	ScanSizeLimit string
	QueryTimeout  string
}

// buildPromptHint renders the tool_description.md template with config data.
// The result is intended to be included in the parent agent's system prompt.
func buildPromptHint(config *Config) (string, error) {
	data := promptHintData{
		HasTables: len(config.Tables) > 0,
		Tables:    config.Tables,
	}

	if config.ScanSizeLimit > 0 {
		data.ScanSizeLimit = humanize.IBytes(config.ScanSizeLimit)
	}
	if config.QueryTimeout > 0 {
		data.QueryTimeout = config.QueryTimeout.String()
	}

	var buf bytes.Buffer
	if err := toolDescriptionTmpl.Execute(&buf, data); err != nil {
		return "", goerr.Wrap(err, "failed to execute tool description template")
	}

	return buf.String(), nil
}

// buildSystemPrompt builds system prompt without memories (for factory)
func buildSystemPrompt(config *Config) (string, error) {
	// Build base prompt
	var buf bytes.Buffer
	buf.WriteString(basePrompt)
	buf.WriteString("\n\n")

	// Prepare template data
	runbooksData := make(map[string]interface{})
	for id, entry := range config.Runbooks {
		runbooksData[id.String()] = entry
	}

	data := promptData{
		Tables:      config.Tables,
		HasMemories: false,
		Memories:    nil,
		Letters:     []string{},
		Runbooks:    runbooksData,
	}

	// Execute main template
	if err := systemWithMemoriesTmpl.Execute(&buf, data); err != nil {
		return "", goerr.Wrap(err, "failed to execute system prompt template")
	}

	// Append runbooks section
	if len(config.Runbooks) > 0 {
		buf.WriteString("\n\n")
		if err := runbooksTmpl.Execute(&buf, data); err != nil {
			return "", goerr.Wrap(err, "failed to execute runbooks template")
		}
	}

	return buf.String(), nil
}
