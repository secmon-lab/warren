package bigquery

import (
	"bytes"
	"context"
	_ "embed"
	"text/template"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/memory"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

//go:embed prompt/base.md
var basePrompt string

//go:embed prompt/system_with_memories.md
var systemWithMemoriesTemplate string

//go:embed prompt/tool_description.md
var toolDescriptionTemplate string

//go:embed prompt/runbooks.md
var runbooksTemplate string

var systemWithMemoriesTmpl *template.Template
var toolDescriptionTmpl *template.Template
var runbooksTmpl *template.Template

func init() {
	systemWithMemoriesTmpl = template.Must(template.New("system_with_memories").Parse(systemWithMemoriesTemplate))
	toolDescriptionTmpl = template.Must(template.New("tool_description").Parse(toolDescriptionTemplate))
	runbooksTmpl = template.Must(template.New("runbooks").Parse(runbooksTemplate))
}

// promptData represents the data for system prompt template
type promptData struct {
	Tables      []TableConfig
	HasMemories bool
	Memories    []*memory.AgentMemory
	Letters     []string
	Runbooks    map[string]interface{}
}

// buildSystemPromptWithMemories builds system prompt with KPT-formatted memories using templates
func (a *Agent) buildSystemPromptWithMemories(ctx context.Context, memories []*memory.AgentMemory) (string, error) {
	log := logging.From(ctx)
	log.Debug("Building system prompt with memories", "memory_count", len(memories), "table_count", len(a.config.Tables), "runbooks", len(a.config.Runbooks))

	// Build base prompt
	var buf bytes.Buffer
	buf.WriteString(basePrompt)
	buf.WriteString("\n\n")

	// Generate letters for experiences (A, B, C, ...)
	letters := make([]string, len(memories))
	for i := range memories {
		letters[i] = string(rune('A' + i))
	}

	// Convert runbooks map to interface{} for template
	runbooksData := make(map[string]interface{})
	for id, entry := range a.config.Runbooks {
		runbooksData[id.String()] = entry
	}

	// Prepare template data
	data := promptData{
		Tables:      a.config.Tables,
		HasMemories: len(memories) > 0,
		Memories:    memories,
		Letters:     letters,
		Runbooks:    runbooksData,
	}

	// Execute main template
	if err := systemWithMemoriesTmpl.Execute(&buf, data); err != nil {
		return "", goerr.Wrap(err, "failed to execute system prompt template")
	}

	// Append runbooks section
	if len(a.config.Runbooks) > 0 {
		buf.WriteString("\n\n")
		if err := runbooksTmpl.Execute(&buf, data); err != nil {
			return "", goerr.Wrap(err, "failed to execute runbooks template")
		}
	}

	prompt := buf.String()
	log.Debug("System prompt built successfully", "total_length", len(prompt))

	return prompt, nil
}
