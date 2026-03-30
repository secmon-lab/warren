package slack

import (
	"context"

	"github.com/m-mizutani/gollem"
)

// toolSet implements interfaces.ToolSet by wrapping the internalTool.
type toolSet struct {
	tool *internalTool
}

func (ts *toolSet) ID() string {
	return "slack_agent"
}

func (ts *toolSet) Description() string {
	return "Search for messages in Slack workspace, retrieve thread replies, and get context around specific messages."
}

func (ts *toolSet) Prompt(_ context.Context) (string, error) {
	return buildSystemPrompt()
}

func (ts *toolSet) Specs(ctx context.Context) ([]gollem.ToolSpec, error) {
	return ts.tool.Specs(ctx)
}

func (ts *toolSet) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	return ts.tool.Run(ctx, name, args)
}
