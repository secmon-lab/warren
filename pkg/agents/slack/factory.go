package slack

import (
	"context"

	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	slackSDK "github.com/slack-go/slack"
	"github.com/urfave/cli/v3"
)

// Factory implements agents.ToolSetFactory interface.
type Factory struct {
	oauthToken string
}

// Flags implements agents.ToolSetFactory
func (f *Factory) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "agent-slack-user-token",
			Usage:       "Slack User OAuth Token for message search (requires search:read scope)",
			Destination: &f.oauthToken,
			Category:    "Agent:Slack",
			Sources:     cli.EnvVars("WARREN_AGENT_SLACK_USER_TOKEN"),
		},
	}
}

// Configure implements agents.ToolSetFactory.
// Returns (nil, nil) if the OAuth token is not set.
func (f *Factory) Configure(ctx context.Context) (interfaces.ToolSet, error) {
	if f.oauthToken == "" {
		return nil, nil
	}

	slackClient := slackSDK.New(f.oauthToken)

	ts := &toolSet{
		tool: &internalTool{slackClient: slackClient},
	}

	logging.From(ctx).Info("Slack Search Agent configured")

	return ts, nil
}
