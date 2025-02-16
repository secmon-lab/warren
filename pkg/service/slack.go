package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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
	userID        string
	teamID        string
	botID         string
}

var _ interfaces.SlackService = &Slack{}

type SlackThread struct {
	channelID   string
	threadID    string
	slackClient *slack.Client
}

var _ interfaces.SlackThreadService = &SlackThread{}

func (x *SlackThread) ChannelID() string {
	return x.channelID
}

func (x *SlackThread) ThreadID() string {
	return x.threadID
}

func NewSlack(oauthToken, signingSecret, channelID string) (*Slack, error) {
	if oauthToken == "" {
		return nil, goerr.New("oauthToken is empty")
	}

	s := &Slack{
		signingSecret: signingSecret,
		channelID:     channelID,
		slackClient:   slack.New(oauthToken),
	}

	authTest, err := s.slackClient.AuthTest()
	if err != nil {
		return nil, goerr.Wrap(err, "failed to auth test of slack", goerr.V("oauthToken.len", len(oauthToken)))
	}

	s.userID = authTest.UserID
	s.teamID = authTest.TeamID
	s.botID = authTest.BotID

	return s, nil
}

func (x *Slack) TrimMention(message string) string {
	mention := "<@" + x.userID + ">"
	idx := strings.LastIndex(message, mention)
	if idx == -1 {
		return message
	}

	return strings.TrimSpace(message[idx+len(mention):])
}

func (x *Slack) NewThread(alert model.Alert) interfaces.SlackThreadService {
	return &SlackThread{
		channelID:   alert.SlackThread.ChannelID,
		threadID:    alert.SlackThread.ThreadID,
		slackClient: x.slackClient,
	}
}

func buildAlertBlocks(alert model.Alert) []slack.Block {
	blocks := []slack.Block{
		slack.NewHeaderBlock(
			slack.NewTextBlockObject("plain_text", alert.Title, false, false),
		),
		slack.NewSectionBlock(
			slack.NewTextBlockObject("mrkdwn", alert.Description, false, false),
			nil,
			nil,
		),
		slack.NewDividerBlock(),
		slack.NewSectionBlock(
			slack.NewTextBlockObject("mrkdwn", "*ID:* `"+string(alert.ID)+"`\n*Schema:* `"+alert.Schema+"`\n*Status:* "+func() string {
				switch alert.Status {
				case model.AlertStatusNew:
					return ":new: NEW"
				case model.AlertStatusAcknowledged:
					return ":eyes: ACKNOWLEDGED"
				case model.AlertStatusClosed:
					return ":white_check_mark: CLOSED"
				default:
					return string(alert.Status)
				}
			}()+"\n*Assignee:* "+func() string {
				if alert.Assignee == nil {
					return ":no_entry: unassigned"
				}
				return ":bust_in_silhouette: <@" + alert.Assignee.ID + ">"
			}()+"\n*Severity:* "+func() string {
				if alert.Finding == nil {
					return ":question: not available"
				}

				switch alert.Finding.Severity {
				case model.AlertSeverityCritical:
					return ":rotating_light: *CRITICAL* :rotating_light:"
				case model.AlertSeverityHigh:
					return ":exclamation: *HIGH*"
				case model.AlertSeverityMedium:
					return ":warning: MEDIUM"
				case model.AlertSeverityLow:
					return ":eyes: LOW"
				case model.AlertSeverityUnknown:
					return ":gray_question: unknown"
				default:
					return string(alert.Finding.Severity)
				}
			}(), false, false),
			nil,
			nil,
		),
	}
	blocks = append(blocks, slack.NewDividerBlock())

	if len(alert.Attributes) > 0 {
		fields := make([]*slack.TextBlockObject, 0, len(alert.Attributes)*2)
		for _, attr := range alert.Attributes {
			var value string
			if attr.Link != "" {
				value = "<" + attr.Link + "|" + attr.Value + ">"
			} else {
				value = "`" + attr.Value + "`"
			}
			fields = append(fields,
				slack.NewTextBlockObject("mrkdwn", "*"+attr.Key+":*\n"+value, false, false),
			)
		}
		blocks = append(blocks, slack.NewSectionBlock(nil, fields, nil))
	}
	if alert.Finding != nil {
		blocks = append(blocks,
			slack.NewSectionBlock(
				slack.NewTextBlockObject("mrkdwn", "📝 *Summary:*\n"+alert.Finding.Summary, false, false),
				nil,
				nil,
			),
			slack.NewDividerBlock(),
			slack.NewSectionBlock(
				slack.NewTextBlockObject("mrkdwn", "🔍 *Reason:*\n"+alert.Finding.Reason, false, false),
				nil,
				nil,
			),
			slack.NewDividerBlock(),
			slack.NewSectionBlock(
				slack.NewTextBlockObject("mrkdwn", "💡 *Recommendation:*\n"+alert.Finding.Recommendation, false, false),
				nil,
				nil,
			),
		)
	}

	// Add action buttons
	buttons := []slack.BlockElement{}
	if alert.Finding == nil {
		buttons = append(buttons,
			slack.NewButtonBlockElement(
				"inspect",
				alert.ID.String(),
				slack.NewTextBlockObject("plain_text", "Inspect", false, false),
			).WithStyle(slack.StyleDefault),
		)
	}

	if alert.Status == model.AlertStatusNew {
		buttons = append(buttons,
			slack.NewButtonBlockElement(
				"ack",
				alert.ID.String(),
				slack.NewTextBlockObject("plain_text", "Acknowledge", false, false),
			).WithStyle(slack.StylePrimary),
		)
	}

	if alert.Status != model.AlertStatusClosed {
		buttons = append(buttons,
			slack.NewButtonBlockElement(
				"close",
				alert.ID.String(),
				slack.NewTextBlockObject("plain_text", "Close", false, false),
			).WithStyle(slack.StyleDanger),
		)
	}

	blocks = append(blocks, slack.NewActionBlock("alert_actions", buttons...))

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

	thread := &SlackThread{
		channelID:   channelID,
		threadID:    timestamp,
		slackClient: x.slackClient,
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(alert.Data); err != nil {
		return nil, goerr.Wrap(err, "failed to encode alert data")
	}

	if err := thread.AttachFile(ctx, "Original Alert", "alert.json", buf.Bytes()); err != nil {
		return nil, goerr.Wrap(err, "failed to attach file to slack")
	}

	return thread, nil
}

func (x *SlackThread) UpdateAlert(ctx context.Context, alert model.Alert) error {
	blocks := buildAlertBlocks(alert)

	_, _, _, err := x.slackClient.UpdateMessageContext(
		ctx,
		alert.SlackThread.ChannelID,
		alert.SlackThread.ThreadID,
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
		return goerr.Wrap(err, "failed to post next action to slack")
	}

	return nil
}

// buildNextActionBlocks builds the blocks for the next action message in the thread.
func buildNextActionBlocks(action prompt.ActionPromptResult) []slack.Block {
	var fields []*slack.TextBlockObject
	for key, arg := range action.Args {
		fields = append(fields, slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("*%s:* `%s`", key, arg), false, false))
	}

	nextMsg := fmt.Sprintf("🔜 Action: *%s*\n", action.Action)
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
	if err != nil {
		return goerr.Wrap(err, "failed to upload file to slack")
	}

	return nil
}

func (x *SlackThread) Reply(ctx context.Context, message string) error {
	_, _, err := x.slackClient.PostMessageContext(
		ctx,
		x.channelID,
		slack.MsgOptionText(message, false),
		slack.MsgOptionTS(x.threadID),
	)

	if err != nil {
		return goerr.Wrap(err, "failed to reply to slack")
	}

	return nil
}

func (x *SlackThread) PostFinding(ctx context.Context, finding model.AlertFinding) error {
	blocks := buildFindingBlocks(finding)

	_, _, err := x.slackClient.PostMessageContext(
		ctx,
		x.channelID,
		slack.MsgOptionBlocks(blocks...),
		slack.MsgOptionTS(x.threadID),
	)
	if err != nil {
		return goerr.Wrap(err, "failed to post finding to slack")
	}

	return nil
}

func buildFindingBlocks(finding model.AlertFinding) []slack.Block {
	return []slack.Block{
		slack.NewHeaderBlock(
			slack.NewTextBlockObject("plain_text", "Severity: "+string(finding.Severity), false, false),
		),
		slack.NewSectionBlock(
			slack.NewTextBlockObject("mrkdwn", "*Summary:*\n"+finding.Summary, false, false),
			nil,
			nil,
		),
		slack.NewSectionBlock(
			slack.NewTextBlockObject("mrkdwn", "*Reason:*\n"+finding.Reason, false, false),
			nil,
			nil,
		),
		slack.NewSectionBlock(
			slack.NewTextBlockObject("mrkdwn", "*Recommendation:*\n"+finding.Recommendation, false, false),
			nil,
			nil,
		),
	}
}

func NewSlackPayloadVerifier(signingSecret string) interfaces.SlackPayloadVerifier {
	return func(ctx context.Context, header http.Header, payload []byte) error {
		eb := goerr.NewBuilder(goerr.V("body", string(payload)), goerr.V("header", header))
		verifier, err := slack.NewSecretsVerifier(header, signingSecret)
		if err != nil {
			return eb.Wrap(err, "failed to create secrets verifier")
		}

		if _, err := verifier.Write(payload); err != nil {
			return eb.Wrap(err, "failed to write request body to verifier")
		}

		if err := verifier.Ensure(); err != nil {
			return eb.Wrap(err, "invalid slack signature")
		}

		return nil
	}
}
