package bigquery

import (
	"context"
	"strings"

	"github.com/m-mizutani/gollem"
)

// toolSet implements interfaces.ToolSet by wrapping the internalTool
// and providing Warren-specific metadata for the planner.
type toolSet struct {
	config *Config
	tool   *internalTool
}

// ID implements interfaces.ToolSet.
func (ts *toolSet) ID() string {
	return "bigquery"
}

// Description implements interfaces.ToolSet.
// It dynamically includes available table names so the planner
// knows which tables this tool set has access to.
func (ts *toolSet) Description() string {
	base := "Retrieve data from BigQuery tables. This tool ONLY extracts data records - it does NOT analyze or interpret the data. After receiving the data, YOU must analyze it yourself and provide a complete answer to the user based on the retrieved data. The tool handles table selection, query construction, and returns raw data records."

	if len(ts.config.Tables) > 0 {
		var tables []string
		for _, t := range ts.config.Tables {
			tables = append(tables, strings.Join([]string{t.ProjectID, t.DatasetID, t.TableID}, "."))
		}
		base += " Available tables: " + strings.Join(tables, ", ") + "."
	}

	return base
}

// Prompt implements interfaces.ToolSet.
func (ts *toolSet) Prompt(ctx context.Context) (string, error) {
	return buildSystemPrompt(ts.config)
}

// Specs implements gollem.ToolSet (delegated to internalTool).
func (ts *toolSet) Specs(ctx context.Context) ([]gollem.ToolSpec, error) {
	return ts.tool.Specs(ctx)
}

// Run implements gollem.ToolSet (delegated to internalTool).
func (ts *toolSet) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	return ts.tool.Run(ctx, name, args)
}
