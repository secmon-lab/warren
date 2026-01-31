package agents

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/agents/bigquery"
	"github.com/secmon-lab/warren/pkg/agents/slack"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/urfave/cli/v3"
)

// All is the list of all available agent factories.
// To add a new agent, simply append its Factory to this list.
var All = []AgentFactory{
	&bigquery.Factory{},
	&slack.Factory{},
}

// AllFlags returns CLI flags for all registered agents.
func AllFlags() []cli.Flag {
	var flags []cli.Flag
	for _, factory := range All {
		flags = append(flags, factory.Flags()...)
	}
	return flags
}

// ConfigureAll initializes all configured agents and returns a slice of SubAgents.
// Agents that are not configured will return nil and be skipped.
func ConfigureAll(ctx context.Context, llmClient gollem.LLMClient, repo interfaces.Repository) ([]*gollem.SubAgent, error) {
	var subAgents []*gollem.SubAgent

	for _, factory := range All {
		subAgent, err := factory.Configure(ctx, llmClient, repo)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to configure agent")
		}

		if subAgent != nil {
			subAgents = append(subAgents, subAgent)
		}
	}

	return subAgents, nil
}
