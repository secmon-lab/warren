package agents

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/agents/bigquery"
	"github.com/secmon-lab/warren/pkg/agents/falcon"
	"github.com/secmon-lab/warren/pkg/agents/slack"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/urfave/cli/v3"
)

// All is the list of all available agent factories.
// To add a new agent, simply append its Factory to this list.
var All = []ToolSetFactory{
	&bigquery.Factory{},
	&falcon.Factory{},
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

// ConfigureAll initializes all configured agents and returns a slice of ToolSets.
// Agents that are not configured will return nil and be skipped.
func ConfigureAll(ctx context.Context) ([]interfaces.ToolSet, error) {
	var toolSets []interfaces.ToolSet

	for _, factory := range All {
		ts, err := factory.Configure(ctx)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to configure agent")
		}

		if ts != nil {
			toolSets = append(toolSets, ts)
		}
	}

	return toolSets, nil
}
