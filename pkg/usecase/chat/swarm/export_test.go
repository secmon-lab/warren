package swarm

import (
	"context"

	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/model/agent"
)

// FilterToolSets exposes filterToolSets for testing.
func FilterToolSets(ctx context.Context, allTools []gollem.ToolSet, allowedNames []string) []gollem.ToolSet {
	return filterToolSets(ctx, allTools, allowedNames)
}

// FilterSubAgents exposes filterSubAgents for testing.
func FilterSubAgents(allAgents []*agent.SubAgent, allowedNames []string) []*agent.SubAgent {
	return filterSubAgents(allAgents, allowedNames)
}
