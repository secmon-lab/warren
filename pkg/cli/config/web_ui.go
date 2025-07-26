package config

import (
	"context"
	"log/slog"

	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/service/slack"
	"github.com/secmon-lab/warren/pkg/usecase"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/urfave/cli/v3"
)

type WebUI struct {
	clientID         string
	clientSecret     string
	frontendURL      string
	noAuthentication bool
}

// SetFrontendURL sets the frontend URL
func (x *WebUI) SetFrontendURL(url string) {
	x.frontendURL = url
}

func (x *WebUI) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "slack-client-id",
			Usage:       "Slack OAuth Client ID",
			Category:    "Web UI",
			Sources:     cli.EnvVars("WARREN_SLACK_CLIENT_ID"),
			Destination: &x.clientID,
		},
		&cli.StringFlag{
			Name:        "slack-client-secret",
			Usage:       "Slack OAuth Client Secret",
			Category:    "Web UI",
			Sources:     cli.EnvVars("WARREN_SLACK_CLIENT_SECRET"),
			Destination: &x.clientSecret,
		},
		&cli.StringFlag{
			Name:        "frontend-url",
			Usage:       "Frontend URL for OAuth callback (e.g., http://localhost:3000)",
			Category:    "Web UI",
			Sources:     cli.EnvVars("WARREN_FRONTEND_URL"),
			Destination: &x.frontendURL,
		},
		&cli.BoolFlag{
			Name:        "no-authentication",
			Usage:       "Disable authentication (for local development only)",
			Category:    "Web UI",
			Sources:     cli.EnvVars("WARREN_NO_AUTHENTICATION"),
			Destination: &x.noAuthentication,
			Aliases:     []string{"no-authn"}, // Short alias
		},
	}
}

func (x WebUI) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("client-id.len", x.clientID),
		slog.Int("client-secret.len", len(x.clientSecret)),
		slog.String("frontend-url", x.frontendURL),
		slog.Bool("no-authn", x.noAuthentication),
	)
}

func (x *WebUI) IsConfigured() bool {
	return (x.clientID != "" && x.clientSecret != "" && x.frontendURL != "") || x.noAuthentication
}

func (x *WebUI) GetCallbackURL() string {
	if x.frontendURL == "" {
		return ""
	}
	return x.frontendURL + "/api/auth/callback"
}

func (x *WebUI) GetFrontendURL() string {
	return x.frontendURL
}

func (x *WebUI) Configure(ctx context.Context, repo interfaces.Repository, slackSvc *slack.Service) (usecase.AuthUseCaseInterface, error) {
	// Check if Slack authentication is configured
	if x.clientID != "" && x.clientSecret != "" {
		if x.noAuthentication {
			slog.Warn("--no-authentication flag is ignored because Slack authentication is configured")
		}
		callbackURL := x.GetCallbackURL()
		return usecase.NewAuthUseCase(repo, slackSvc, x.clientID, x.clientSecret, callbackURL), nil
	} else if x.noAuthentication {
		logging.From(ctx).Warn("Authentication is disabled. This mode is for local development only.")
		return usecase.NewNoAuthnUseCase(repo), nil
	}

	// No authentication configured
	return nil, nil
}
