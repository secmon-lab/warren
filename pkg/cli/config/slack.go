package config

import (
	"log/slog"

	"github.com/m-mizutani/goerr/v2"
	model "github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/service/slack"
	"github.com/urfave/cli/v3"

	sdk "github.com/slack-go/slack"
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

func (x *Slack) Configure() (*slack.Service, error) {
	if x.oauthToken == "" {
		return nil, goerr.New("slack oauth token is not set")
	}

	client := sdk.New(x.oauthToken)

	return slack.New(client, x.channelID)
}

func (x *Slack) ConfigureWithFrontendURL(frontendURL string) (*slack.Service, error) {
	if x.oauthToken == "" {
		return nil, goerr.New("slack oauth token is not set")
	}

	client := sdk.New(x.oauthToken)

	var opts []slack.ServiceOption
	if frontendURL != "" {
		opts = append(opts, slack.WithFrontendURL(frontendURL))
	}

	return slack.New(client, x.channelID, opts...)
}

func (x *Slack) Verifier() model.PayloadVerifier {
	if x.signingSecret == "" {
		return nil
	}

	return model.NewPayloadVerifier(x.signingSecret)
}
