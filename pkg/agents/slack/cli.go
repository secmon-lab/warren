package slack

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	slackSDK "github.com/slack-go/slack"
	"github.com/urfave/cli/v3"
)

// CLI integration for Slack Search Agent

// Flags returns CLI flags for Slack Agent configuration
func Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:     "agent-slack-user-token",
			Usage:    "Slack User OAuth Token for message search (requires search:read scope)",
			Category: "Agent:Slack",
			Sources:  cli.EnvVars("WARREN_AGENT_SLACK_USER_TOKEN"),
		},
	}
}

// NewSubAgentFromCLI creates a Slack SubAgent from CLI context
// Returns nil if the agent is not configured (no token provided)
func NewSubAgentFromCLI(ctx context.Context, c *cli.Command, llmClient gollem.LLMClient, repo interfaces.Repository) (*gollem.SubAgent, error) {
	userToken := c.String("agent-slack-user-token")
	if userToken == "" {
		return nil, nil
	}

	// Create Slack client
	slackClient := slackSDK.New(userToken)

	// Create agent
	agent := New(ctx, slackClient, llmClient, repo)
	subAgent, err := agent.SubAgent()
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create Slack SubAgent")
	}

	return subAgent, nil
}
