package falcon

import (
	"context"

	"github.com/m-mizutani/gollem"
)

// toolSet implements interfaces.ToolSet for CrowdStrike Falcon.
type toolSet struct {
	internal *internalTool
}

func (ts *toolSet) ID() string {
	return "falcon"
}

func (ts *toolSet) Description() string {
	return "CrowdStrike Falcon EDR queries for incidents, alerts, behaviors, devices, and raw events"
}

func (ts *toolSet) Prompt(_ context.Context) (string, error) {
	return buildSystemPrompt(), nil
}

func (ts *toolSet) Specs(ctx context.Context) ([]gollem.ToolSpec, error) {
	return ts.internal.Specs(ctx)
}

func (ts *toolSet) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	return ts.internal.Run(ctx, name, args)
}
