package service

import (
	"context"
	"net/http"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/slack-go/slack"
)

type Slack struct {
	signingSecret string
	channelID     string
	slackClient   *slack.Client
}

func NewSlack(oauthToken, signingSecret, channelID string) *Slack {
	return &Slack{
		signingSecret: signingSecret,
		channelID:     channelID,
		slackClient:   slack.New(oauthToken),
	}
}

func buildAlertBlocks(alert model.Alert) []slack.Block {
	blocks := []slack.Block{
		slack.NewHeaderBlock(
			slack.NewTextBlockObject("plain_text", alert.Title, false, false),
		),
		slack.NewSectionBlock(
			slack.NewTextBlockObject("mrkdwn", "*Schema:* "+alert.Schema, false, false),
			nil,
			nil,
		),
	}
	blocks = append(blocks, slack.NewDividerBlock())

	if len(alert.Attributes) > 0 {
		fields := make([]*slack.TextBlockObject, 0, len(alert.Attributes)*2)
		for _, attr := range alert.Attributes {
			value := attr.Value
			if attr.Link != "" {
				value = "<" + attr.Link + "|" + attr.Value + ">"
			}
			fields = append(fields,
				slack.NewTextBlockObject("mrkdwn", "*"+attr.Key+":*\n"+value, false, false),
			)
		}
		blocks = append(blocks, slack.NewSectionBlock(nil, fields, nil))
	}

	return blocks
}

func (x *Slack) PostAlert(ctx context.Context, alert model.Alert) (string, string, error) {
	blocks := buildAlertBlocks(alert)

	channelID, timestamp, err := x.slackClient.PostMessageContext(
		ctx,
		x.channelID,
		slack.MsgOptionBlocks(blocks...),
	)
	return channelID, timestamp, err
}

func (x *Slack) UpdateAlert(ctx context.Context, alert model.Alert) error {
	blocks := buildAlertBlocks(alert)

	_, _, _, err := x.slackClient.UpdateMessageContext(
		ctx,
		alert.SlackChannel,
		alert.SlackMessageID,
		slack.MsgOptionBlocks(blocks...),
	)
	if err != nil {
		return goerr.Wrap(err, "failed to update message to slack")
	}

	return nil
}

func (x *Slack) VerifyRequest(header http.Header, body []byte) error {
	sv, err := slack.NewSecretsVerifier(header, string(body))
	if err != nil {
		return goerr.Wrap(err, "failed to create secrets verifier")
	}

	if err := sv.Ensure(); err != nil {
		return goerr.Wrap(err, "failed to verify request")
	}

	return nil
}
