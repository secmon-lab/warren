package swarm

import (
	"context"

	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/model/agent"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

// FilterToolSets exposes filterToolSets for testing.
func FilterToolSets(ctx context.Context, allTools []gollem.ToolSet, allowedNames []string) []gollem.ToolSet {
	return filterToolSets(ctx, allTools, allowedNames)
}

// FilterSubAgents exposes filterSubAgents for testing.
func FilterSubAgents(allAgents []*agent.SubAgent, allowedNames []string) []*agent.SubAgent {
	return filterSubAgents(allAgents, allowedNames)
}

// StartSessionMonitor exposes startSessionMonitor for testing.
func (c *SwarmChat) StartSessionMonitor(ctx context.Context, sessionID types.SessionID) (context.Context, func()) {
	return c.startSessionMonitor(ctx, sessionID)
}
