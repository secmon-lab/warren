package config

import (
	"log/slog"

	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/usecase"
	"github.com/urfave/cli/v3"
)

type Auth struct {
	clientID     string
	clientSecret string
	frontendURL  string
}

func (x *Auth) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "slack-client-id",
			Usage:       "Slack OAuth Client ID",
			Category:    "Authentication",
			Sources:     cli.EnvVars("WARREN_SLACK_CLIENT_ID"),
			Destination: &x.clientID,
		},
		&cli.StringFlag{
			Name:        "slack-client-secret",
			Usage:       "Slack OAuth Client Secret",
			Category:    "Authentication",
			Sources:     cli.EnvVars("WARREN_SLACK_CLIENT_SECRET"),
			Destination: &x.clientSecret,
		},
		&cli.StringFlag{
			Name:        "frontend-url",
			Usage:       "Frontend URL for OAuth callback (e.g., http://localhost:3000)",
			Category:    "Authentication",
			Sources:     cli.EnvVars("WARREN_FRONTEND_URL"),
			Destination: &x.frontendURL,
		},
	}
}

func (x Auth) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int("client-id.len", len(x.clientID)),
		slog.Int("client-secret.len", len(x.clientSecret)),
		slog.String("frontend-url", x.frontendURL),
	)
}

func (x *Auth) IsConfigured() bool {
	return x.clientID != "" && x.clientSecret != "" && x.frontendURL != ""
}

func (x *Auth) GetCallbackURL() string {
	if x.frontendURL == "" {
		return ""
	}
	return x.frontendURL + "/api/auth/callback"
}

func (x *Auth) Configure(repo interfaces.Repository) (*usecase.AuthUseCase, error) {
	if !x.IsConfigured() {
		return nil, nil // Return nil if not configured (authentication is optional)
	}

	callbackURL := x.GetCallbackURL()
	return usecase.NewAuthUseCase(repo, x.clientID, x.clientSecret, callbackURL), nil
}
