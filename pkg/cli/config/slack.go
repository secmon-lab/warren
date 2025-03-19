package config

import (
	"log/slog"

	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/service"
	"github.com/urfave/cli/v3"
)

type Slack struct {
	oauthToken    string
	signingSecret string
	channelID     string
}

func (x *Slack) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "slack-oauth-token",
			Usage:       "Slack OAuth token",
			Category:    "Slack",
			Destination: &x.oauthToken,
			Sources:     cli.EnvVars("WARREN_SLACK_OAUTH_TOKEN"),
		},
		&cli.StringFlag{
			Name:        "slack-signing-secret",
			Usage:       "Slack signing secret",
			Category:    "Slack",
			Destination: &x.signingSecret,
			Sources:     cli.EnvVars("WARREN_SLACK_SIGNING_SECRET"),
		},
		&cli.StringFlag{
			Name:        "slack-channel-name",
			Usage:       "Slack channel name, `#` is not required",
			Category:    "Slack",
			Destination: &x.channelID,
			Sources:     cli.EnvVars("WARREN_SLACK_CHANNEL_NAME"),
		},
	}
}

func (x Slack) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int("oauth-token.len", len(x.oauthToken)),
		slog.Int("signing-secret.len", len(x.signingSecret)),
		slog.String("channel-id", x.channelID),
	)
}

func (x *Slack) Configure() (*service.Slack, error) {
	if x.oauthToken == "" {
		return nil, nil
	}

	return service.NewSlack(x.oauthToken, x.signingSecret, x.channelID)
}

func (x *Slack) Verifier() interfaces.SlackPayloadVerifier {
	if x.signingSecret == "" {
		return nil
	}

	return service.NewSlackPayloadVerifier(x.signingSecret)
}
