package bigquery

import (
	"bytes"
	_ "embed"
	"text/template"

	"github.com/dustin/go-humanize"
	"github.com/m-mizutani/goerr/v2"
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
	Tables   []TableConfig
	Runbooks map[string]interface{}
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
		Tables:   config.Tables,
		Runbooks: runbooksData,
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
