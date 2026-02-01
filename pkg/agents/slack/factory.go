package slack

import (
	"context"

	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/service/memory"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	slackSDK "github.com/slack-go/slack"
	"github.com/urfave/cli/v3"
)

// Factory implements agents.AgentFactory interface.
type Factory struct {
	oauthToken string
}

// Flags implements agents.AgentFactory
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

// Configure implements agents.AgentFactory
func (f *Factory) Configure(ctx context.Context, llmClient gollem.LLMClient, repo interfaces.Repository) (*gollem.SubAgent, string, error) {
	if f.oauthToken == "" {
		return nil, "", nil
	}

	slackClient := slackSDK.New(f.oauthToken)

	a := &agent{
		slackClient:   slackClient,
		llmClient:     llmClient,
		repo:          repo,
		internalTool:  &internalTool{slackClient: slackClient},
		memoryService: memory.New("slack_search", llmClient, repo),
	}

	logging.From(ctx).Info("Slack Search Agent configured")

	subAgent, err := a.subAgent()
	if err != nil {
		return nil, "", err
	}

	// Slack agent has no config-dependent prompt hint
	return subAgent, "", nil
}
