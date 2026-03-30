package interfaces

import (
	"context"

	"github.com/m-mizutani/gollem"
)

// ToolSet extends gollem.ToolSet with Warren-specific metadata for planning.
// The planner LLM selects tools by ToolSet ID, and all tools within a
// selected ToolSet are provided to the task agent as a group.
type ToolSet interface {
	gollem.ToolSet

	// ID returns the unique identifier for this tool set used by the planner
	// (e.g. "bigquery", "virustotal", "falcon").
	ID() string

	// Description returns a concise description of the tool set for the planner.
	Description() string

	// Prompt returns additional system prompt content to inject into the task
	// agent when this tool set is selected. Return empty string if not needed.
	Prompt(ctx context.Context) (string, error)
}

// WrapToolSet wraps a plain gollem.ToolSet into a Warren ToolSet with the
// given ID and description. Useful for MCP tool sets and other external tools.
func WrapToolSet(ts gollem.ToolSet, id, description string) ToolSet {
	return &wrappedToolSet{ToolSet: ts, id: id, description: description}
}

type wrappedToolSet struct {
	gollem.ToolSet
	id          string
	description string
}

func (w *wrappedToolSet) ID() string                               { return w.id }
func (w *wrappedToolSet) Description() string                      { return w.description }
func (w *wrappedToolSet) Prompt(_ context.Context) (string, error) { return "", nil }
