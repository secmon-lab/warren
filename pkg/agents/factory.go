package agents

import (
	"context"

	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/urfave/cli/v3"
)

// AgentFactory is the interface that all agent packages must implement.
// This interface provides a unified way for the CLI layer to interact with agents.
type AgentFactory interface {
	// Flags returns CLI flags for this agent
	Flags() []cli.Flag

	// Configure creates and initializes the agent, returning a gollem.SubAgent and
	// an optional prompt hint string for the parent agent's system prompt.
	// Returns (nil, "", nil) if the agent is not configured.
	Configure(ctx context.Context, llmClient gollem.LLMClient, repo interfaces.Repository) (*gollem.SubAgent, string, error)
}
