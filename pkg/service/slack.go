package service

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/interfaces"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/prompt"
	"github.com/slack-go/slack"
)

type Slack struct {
	signingSecret string
	channelID     string
	slackClient   *slack.Client
}

type SlackThread struct {
	channelID   string
	threadID    string
	slackClient *slack.Client
}

func (x *SlackThread) ChannelID() string {
	return x.channelID
}

func (x *SlackThread) ThreadID() string {
	return x.threadID
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

func (x *Slack) PostAlert(ctx context.Context, alert model.Alert) (interfaces.SlackThreadService, error) {
	blocks := buildAlertBlocks(alert)

	channelID, timestamp, err := x.slackClient.PostMessageContext(
		ctx,
		x.channelID,
		slack.MsgOptionBlocks(blocks...),
	)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to post message to slack")
	}

	return &SlackThread{
		channelID:   channelID,
		threadID:    timestamp,
		slackClient: x.slackClient,
	}, nil
}

func (x *SlackThread) UpdateAlert(ctx context.Context, alert model.Alert) error {
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

func (x *SlackThread) PostNextAction(ctx context.Context, action prompt.ActionPromptResult) error {
	blocks := buildNextActionBlocks(action)

	_, _, err := x.slackClient.PostMessageContext(
		ctx,
		x.channelID,
		slack.MsgOptionBlocks(blocks...),
		slack.MsgOptionTS(x.threadID),
	)
	if err != nil {
		return goerr.Wrap(err, "failed to update message to slack")
	}

	return nil
}

// buildNextActionBlocks builds the blocks for the next action message in the thread.
func buildNextActionBlocks(action prompt.ActionPromptResult) []slack.Block {
	var fields []*slack.TextBlockObject
	for key, arg := range action.Args {
		fields = append(fields, slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*%s:* `%s`", key, arg), false, false))
	}

	nextMsg := fmt.Sprintf("⏭️ Next: *%s*\n", action.Action)
	blocks := []slack.Block{
		slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, nextMsg, false, false),
			fields,
			nil,
		),
	}

	return blocks
}

func (x *SlackThread) AttachFile(ctx context.Context, title, fileName string, data []byte) error {
	_, err := x.slackClient.UploadFileV2Context(ctx, slack.UploadFileV2Parameters{
		Channel:         x.channelID,
		Reader:          bytes.NewReader(data),
		FileSize:        len(data),
		Filename:        fileName,
		Title:           title,
		ThreadTimestamp: x.threadID,
	})
	return err
}

func (x *Slack) VerifyRequest(header http.Header, body []byte) error {
	eb := goerr.NewBuilder(goerr.V("header", header), goerr.V("body", body))

	sv, err := slack.NewSecretsVerifier(header, x.signingSecret)
	if err != nil {
		return eb.Wrap(err, "failed to create secrets verifier")
	}

	if _, err := sv.Write(body); err != nil {
		return eb.Wrap(err, "failed to write request body")
	}

	if err := sv.Ensure(); err != nil {
		return eb.Wrap(err, "failed to verify request")
	}

	return nil
}
