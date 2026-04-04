package falcon

import (
	"context"

	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/urfave/cli/v3"
)

const defaultBaseURL = "https://api.crowdstrike.com"

// Factory implements agents.ToolSetFactory interface for CrowdStrike Falcon.
type Factory struct {
	clientID     string
	clientSecret string
	baseURL      string
}

// Flags implements agents.ToolSetFactory.
func (f *Factory) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "agent-falcon-client-id",
			Usage:       "CrowdStrike Falcon API Client ID",
			Destination: &f.clientID,
			Category:    "Agent:Falcon",
			Sources:     cli.EnvVars("WARREN_AGENT_FALCON_CLIENT_ID"),
		},
		&cli.StringFlag{
			Name:        "agent-falcon-client-secret",
			Usage:       "CrowdStrike Falcon API Client Secret",
			Destination: &f.clientSecret,
			Category:    "Agent:Falcon",
			Sources:     cli.EnvVars("WARREN_AGENT_FALCON_CLIENT_SECRET"),
		},
		&cli.StringFlag{
			Name:        "agent-falcon-base-url",
			Usage:       "CrowdStrike Falcon API Base URL (default: US-1)",
			Destination: &f.baseURL,
			Category:    "Agent:Falcon",
			Sources:     cli.EnvVars("WARREN_AGENT_FALCON_BASE_URL"),
			Value:       defaultBaseURL,
		},
	}
}

// Configure implements agents.ToolSetFactory.
// Returns (nil, nil) if client_id or client_secret is not set.
func (f *Factory) Configure(ctx context.Context) (interfaces.ToolSet, error) {
	if f.clientID == "" || f.clientSecret == "" {
		return nil, nil
	}

	baseURL := f.baseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	tp := newTokenProvider(f.clientID, f.clientSecret, baseURL)

	logging.From(ctx).Info("CrowdStrike Falcon Agent configured",
		"base_url", baseURL,
		"client_id", f.clientID,
		"client_secret_length", len(f.clientSecret),
	)

	return &toolSet{
		internal: newInternalTool(tp, baseURL),
	}, nil
}
