package slack

import (
	"bytes"
	"context"
	_ "embed"
	"text/template"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/memory"
)

//go:embed prompt/system.md
var systemPromptTemplate string

func buildSystemPrompt(ctx context.Context, limit int, memories []*memory.AgentMemory) (string, error) {
	tmpl, err := template.New("system").Parse(systemPromptTemplate)
	if err != nil {
		return "", goerr.Wrap(err, "failed to parse system prompt template")
	}

	data := map[string]any{
		"limit":    limit,
		"memories": memories,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", goerr.Wrap(err, "failed to execute system prompt template")
	}

	return buf.String(), nil
}
