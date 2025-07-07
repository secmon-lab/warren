package config

import (
	"log/slog"

	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/service/slack"
	"github.com/secmon-lab/warren/pkg/usecase"
	"github.com/urfave/cli/v3"
)

type WebUI struct {
	clientID     string
	clientSecret string
	frontendURL  string
	devUser      string
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
		&cli.StringFlag{
			Name:        "dev-user",
			Usage:       "Enable development mode with specified user ID (skips Slack OAuth)",
			Category:    "Web UI",
			Sources:     cli.EnvVars("WARREN_DEV_USER"),
			Destination: &x.devUser,
		},
	}
}

func (x WebUI) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("client-id.len", x.clientID),
		slog.Int("client-secret.len", len(x.clientSecret)),
		slog.String("frontend-url", x.frontendURL),
		slog.String("dev-user", x.devUser),
	)
}

func (x *WebUI) IsDevMode() bool {
	return x.devUser != ""
}

func (x *WebUI) IsConfigured() bool {
	// In dev mode, we don't need OAuth configuration
	if x.IsDevMode() {
		return true
	}
	return x.clientID != "" && x.clientSecret != "" && x.frontendURL != ""
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

func (x *WebUI) Configure(repo interfaces.Repository, slackSvc *slack.Service) (usecase.AuthUseCaseInterface, error) {
	if !x.IsConfigured() {
		return nil, nil // Return nil if not configured (authentication is optional)
	}

	// Use dev mode authentication if dev-user is specified
	if x.IsDevMode() {
		return usecase.NewAuthUseCaseForDev(x.devUser), nil
	}

	callbackURL := x.GetCallbackURL()
	return usecase.NewAuthUseCase(repo, slackSvc, x.clientID, x.clientSecret, callbackURL), nil
}
