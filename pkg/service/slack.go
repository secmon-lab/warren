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
	"github.com/secmon-lab/warren/pkg/utils/errs"
	"github.com/slack-go/slack"
)

type Slack struct {
	signingSecret string
	channelID     string
	slackClient   *slack.Client
	slackMetadata
}

type slackMetadata struct {
	teamID       string
	teamName     string
	botID        string
	userID       string
	enterpriseID string
}

func (x slackMetadata) ToMsgURL(channelID, threadID string) string {
	if x.enterpriseID == "" {
		return fmt.Sprintf("https://%s.slack.com/archives/%s/p%s", x.teamName, channelID, threadID)
	}

	return fmt.Sprintf("https://%s.slack.com/archives/%s/p%s", x.enterpriseID, channelID, threadID)
}

var _ interfaces.SlackService = &Slack{}

type SlackThread struct {
	channelID   string
	threadID    string
	slackClient *slack.Client
	slackMetadata
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
	s.teamName = authTest.Team
	s.enterpriseID = authTest.EnterpriseID

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

func (x *Slack) PostMessage(ctx context.Context, message string) (*SlackThread, error) {
	channelID, timestamp, err := x.slackClient.PostMessageContext(ctx, x.channelID, slack.MsgOptionText(message, false))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to post message to slack")
	}

	return &SlackThread{
		slackMetadata: x.slackMetadata,
		channelID:     channelID,
		threadID:      timestamp,
		slackClient:   x.slackClient,
	}, nil
}

func (x *Slack) NewThread(thread model.SlackThread) interfaces.SlackThreadService {
	return &SlackThread{
		slackMetadata: x.slackMetadata,
		channelID:     thread.ChannelID,
		threadID:      thread.ThreadID,
		slackClient:   x.slackClient,
	}
}

func buildAlertBlocks(alert model.Alert) []slack.Block {
	lines := []string{
		"*ID:* `" + alert.ID.String() + "`",
		"*Schema:* `" + alert.Schema + "`",
		"*Status:* " + func() string {
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
		}(),
		"*Assignee:* " + func() string {
			if alert.Assignee == nil {
				return ":no_entry: unassigned"
			}
			return ":bust_in_silhouette: <@" + alert.Assignee.ID + ">"
		}(),
		"*Severity:* " + func() string {
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
		}(),
	}

	title := alert.Title
	if len(title) > 140 {
		title = title[:140] + "..."
	}

	description := "_no description_"
	if alert.Description != "" {
		description = alert.Description
	}

	blocks := []slack.Block{
		slack.NewHeaderBlock(
			slack.NewTextBlockObject(slack.PlainTextType, title, false, false),
		),
		slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, description, false, false),
			nil,
			nil,
		),
	}

	if alert.Conclusion != "" {
		blocks = append(blocks, slack.NewDividerBlock())
		blocks = append(blocks, slack.NewSectionBlock(
			slack.NewTextBlockObject("mrkdwn", "*Conclusion:* "+alert.Conclusion.Label(), false, false),
			nil,
			nil,
		))

		if alert.Comment != "" {
			blocks = append(blocks, slack.NewSectionBlock(
				slack.NewTextBlockObject("mrkdwn", alert.Comment, false, false),
				nil,
				nil,
			))
		}
	}

	blocks = append(blocks, []slack.Block{
		slack.NewDividerBlock(),
		slack.NewSectionBlock(
			slack.NewTextBlockObject("mrkdwn", strings.Join(lines, "\n"), false, false),
			nil,
			nil,
		),
		slack.NewDividerBlock(),
	}...)

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
			slack.NewDividerBlock(),
			slack.NewHeaderBlock(
				slack.NewTextBlockObject(slack.PlainTextType, "🤖 AI Analysis Result", false, false),
			),
			slack.NewSectionBlock(
				slack.NewTextBlockObject("mrkdwn", "Severity ➡️ *"+alert.Finding.Severity.String()+"*", false, false),
				nil,
				nil,
			),
			slack.NewSectionBlock(
				slack.NewTextBlockObject("mrkdwn", "📝 *Summary:*\n"+alert.Finding.Summary, false, false),
				nil,
				nil,
			),
			slack.NewSectionBlock(
				slack.NewTextBlockObject("mrkdwn", "🔍 *Reason:*\n"+alert.Finding.Reason, false, false),
				nil,
				nil,
			),
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

	if len(buttons) > 0 {
		blocks = append(blocks, slack.NewActionBlock("alert_actions", buttons...))
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
		return nil, goerr.Wrap(err, "failed to post message to slack", goerr.V("blocks", blocks))
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

	if err := thread.AttachFile(ctx, "Original Alert", "alert."+alert.ID.String()+".json", buf.Bytes()); err != nil {
		return nil, goerr.Wrap(err, "failed to attach file to slack")
	}

	return thread, nil
}

func (x *Slack) ShowCloseAlertModal(ctx context.Context, alert model.Alert, triggerID string) error {

	conclusionOptions := []struct {
		Conclusion  model.AlertConclusion
		Label       string
		Description string
	}{
		{
			Conclusion:  model.AlertConclusionUnaffected,
			Label:       model.AlertConclusionUnaffected.Label(),
			Description: "The alert indicates actual attack or vulnerability, but it is no impact.",
		},
		{
			Conclusion:  model.AlertConclusionIntended,
			Label:       model.AlertConclusionIntended.Label(),
			Description: "The alert is intended behavior or configuration.",
		},
		{
			Conclusion:  model.AlertConclusionFalsePositive,
			Label:       model.AlertConclusionFalsePositive.Label(),
			Description: "The alert is not attack or impact on the system.",
		},
		{
			Conclusion:  model.AlertConclusionTruePositive,
			Label:       model.AlertConclusionTruePositive.Label(),
			Description: "The alert has actual impact on the system.",
		},
	}

	conclusionOptionBlocks := make([]*slack.OptionBlockObject, 0, len(conclusionOptions))
	for _, option := range conclusionOptions {
		conclusionOptionBlocks = append(conclusionOptionBlocks,
			slack.NewOptionBlockObject(
				option.Conclusion.String(),
				slack.NewTextBlockObject(slack.PlainTextType, option.Label, false, false),
				slack.NewTextBlockObject(slack.PlainTextType, option.Description, false, false),
			),
		)
	}

	_, err := x.slackClient.OpenView(triggerID, slack.ModalViewRequest{
		Type: slack.VTModal,
		Title: &slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "Close Alert",
		},
		Blocks: slack.Blocks{
			BlockSet: []slack.Block{
				slack.NewSectionBlock(
					slack.NewTextBlockObject(slack.PlainTextType, "Please input the conclusion and comment.", false, false),
					nil,
					nil,
				),
				slack.NewInputBlock(
					"conclusion",
					slack.NewTextBlockObject(slack.PlainTextType, "Conclusion", false, false),
					slack.NewTextBlockObject(slack.PlainTextType, "Select the conclusion", false, false),
					slack.NewOptionsSelectBlockElement(
						slack.OptTypeStatic,
						slack.NewTextBlockObject(slack.PlainTextType, "Select a conclusion", false, false),
						"conclusion",
						conclusionOptionBlocks...,
					),
				),
				slack.NewInputBlock(
					"comment",
					slack.NewTextBlockObject(slack.PlainTextType, "Comment", false, false),
					slack.NewTextBlockObject(slack.PlainTextType, "Add any reason, context, or information.", false, false),
					slack.NewPlainTextInputBlockElement(
						slack.NewTextBlockObject(slack.PlainTextType, "comment", false, false),
						"comment",
					),
				).WithOptional(true),
			},
		},
		CallbackID:      "close_submit",
		PrivateMetadata: alert.ID.String(),
		Submit: &slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "Close",
		},
		Close: &slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "Cancel",
		},
	})
	if err != nil {
		return goerr.Wrap(err, "failed to open view")
	}

	return nil
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
		return goerr.Wrap(err, "failed to update message to slack", goerr.V("channelID", x.channelID), goerr.V("threadID", x.threadID), goerr.V("blocks", blocks))
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

	nextMsg := fmt.Sprintf("⚡ Action: *%s*\n", action.Action)
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
	if len(data) == 0 {
		msg := fmt.Sprintf("No data to attach: %s", title)
		if _, _, err := x.slackClient.PostMessageContext(ctx, x.channelID, slack.MsgOptionText(msg, false), slack.MsgOptionTS(x.threadID)); err != nil {
			return goerr.Wrap(err, "failed to post no data message to slack", goerr.V("title", title), goerr.V("fileName", fileName))
		}
		return nil
	}

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

func (x *SlackThread) Reply(ctx context.Context, message string) {
	_, _, err := x.slackClient.PostMessageContext(
		ctx,
		x.channelID,
		slack.MsgOptionText(message, false),
		slack.MsgOptionTS(x.threadID),
	)

	if err != nil {
		errs.Handle(ctx, goerr.Wrap(err, "failed to reply to slack"))
	}
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
		return goerr.Wrap(err, "failed to post finding to slack", goerr.V("blocks", blocks))
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

func (x *SlackThread) PostAlertGroups(ctx context.Context, groups []model.AlertGroup) error {
	blocks := buildAlertGroupsBlocks(groups, x.slackMetadata)

	_, _, err := x.slackClient.PostMessageContext(ctx,
		x.channelID,
		slack.MsgOptionBlocks(blocks...),
		slack.MsgOptionTS(x.threadID),
		slack.MsgOptionBroadcast(),
	)
	if err != nil {
		return goerr.Wrap(err, "failed to post alert groups to slack", goerr.V("groups", groups), goerr.V("blocks", blocks))
	}

	return nil
}

func buildAlertGroupsBlocks(groups []model.AlertGroup, metadata slackMetadata) []slack.Block {
	blocks := []slack.Block{
		slack.NewHeaderBlock(
			slack.NewTextBlockObject("plain_text", "Summary of Groups", false, false),
		),
	}

	for _, group := range groups {
		blocks = append(blocks, buildAlertGroupBlocks(group, metadata)...)
	}

	return blocks
}

func buildAlertGroupBlocks(group model.AlertGroup, metadata slackMetadata) []slack.Block {
	blocks := []slack.Block{
		slack.NewDividerBlock(),
		slack.NewHeaderBlock(
			slack.NewTextBlockObject("plain_text", "Group: "+group.Title, false, false),
		),
		slack.NewSectionBlock(
			slack.NewTextBlockObject("mrkdwn", group.Description, false, false),
			nil,
			nil,
		),
	}

	alertList := ""
	// slack link format: https://{team_id}.slack.com/archives/{channel_id}/p{timestamp}
	// Example: https://xxx.slack.com/archives/C07AR2FPG1F/p1740438108890669
	for _, alert := range group.Alerts {
		if alert.SlackThread != nil {
			link := metadata.ToMsgURL(alert.SlackThread.ChannelID, alert.SlackThread.ThreadID)
			alertList += fmt.Sprintf("• <%s|%s>\n", link, alert.Title)
		} else {
			alertList += fmt.Sprintf("• %s\n", alert.Title)
		}
	}
	if len(group.AlertIDs) > 3 {
		alertList += fmt.Sprintf("... and %d more alerts", len(group.AlertIDs)-3)
	}

	if alertList != "" {
		blocks = append(blocks, slack.NewSectionBlock(
			slack.NewTextBlockObject("mrkdwn", alertList, false, false),
			nil,
			nil,
		))
	}

	return blocks
}

func (x *SlackThread) PostPolicyDiff(ctx context.Context, diff *model.PolicyDiff) error {
	blocks := buildPolicyDiffBlocks(diff)

	_, _, err := x.slackClient.PostMessageContext(ctx,
		x.channelID,
		slack.MsgOptionBlocks(blocks...),
		slack.MsgOptionTS(x.threadID),
	)
	if err != nil {
		return goerr.Wrap(err, "failed to post policy diff to slack", goerr.V("blocks", blocks))
	}

	return nil
}

func buildPolicyDiffBlocks(diff *model.PolicyDiff) []slack.Block {
	blocks := []slack.Block{
		slack.NewHeaderBlock(
			slack.NewTextBlockObject("plain_text", "📒 New Ignore Policy: "+diff.Title, false, false),
		),
		slack.NewSectionBlock(
			slack.NewTextBlockObject("mrkdwn", diff.Description, false, false),
			nil,
			nil,
		),
	}

	for fileName, diff := range diff.DiffPolicy() {
		blocks = append(blocks, slack.NewDividerBlock())
		blocks = append(blocks, slack.NewSectionBlock(
			slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*%s*\n```\n%s\n```", fileName, diff), false, false),
			nil,
			nil,
		))
	}

	/*
		blocks = append(blocks, slack.NewDividerBlock())
		blocks = append(blocks, slack.NewActionBlock(
			"create_pr",
			slack.NewButtonBlockElement(
				"create_pr",
				"create_pr",
				slack.NewTextBlockObject("plain_text", "Create Pull Request", false, false),
			),
		))
	*/

	return blocks
}
