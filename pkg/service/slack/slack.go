package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/domain/model/policy"
	model "github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/utils/test"

	"github.com/slack-go/slack"
)

type Service struct {
	channelID string
	client    interfaces.SlackClient
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

func New(client interfaces.SlackClient, channelID string) (*Service, error) {
	s := &Service{
		channelID: channelID,
		client:    client,
	}

	authTest, err := s.client.AuthTest()
	if err != nil {
		return nil, goerr.Wrap(err, "failed to auth test of slack")
	}

	s.userID = authTest.UserID
	s.teamID = authTest.TeamID
	s.teamName = authTest.Team
	s.enterpriseID = authTest.EnterpriseID

	s.botID = authTest.BotID

	return s, nil
}

func NewTestService(t *testing.T) *Service {
	envs := test.NewEnvVars(t, "TEST_SLACK_CHANNEL_ID", "TEST_SLACK_OAUTH_TOKEN")
	client := slack.New(envs.Get("TEST_SLACK_OAUTH_TOKEN"))

	svc, err := New(client, envs.Get("TEST_SLACK_CHANNEL_ID"))
	gt.NoError(t, err).Required()

	return svc
}

func (x *Service) IsBotUser(userID string) bool {
	return x.userID == userID
}

func (x *Service) NewThread(thread model.Thread) *ThreadService {
	return &ThreadService{
		slackMetadata: x.slackMetadata,
		channelID:     thread.ChannelID,
		threadID:      thread.ThreadID,
		client:        x.client,
	}
}

// PostMessage posts a message to the channel and returns the thread. It's just for testing.
func (x *Service) PostMessage(ctx context.Context, message string) (*ThreadService, error) {
	ch, thread, err := x.client.PostMessageContext(ctx, x.channelID, slack.MsgOptionText(message, false))
	if err != nil {
		return nil, err
	}

	return x.NewThread(model.Thread{
		ChannelID: ch,
		ThreadID:  thread,
	}), nil
}

func (x *Service) PostAlert(ctx context.Context, alert alert.Alert) (*ThreadService, error) {
	blocks := buildAlertBlocks(alert)

	channelID, timestamp, err := x.client.PostMessageContext(
		ctx,
		x.channelID,
		slack.MsgOptionBlocks(blocks...),
	)

	if err != nil {
		return nil, goerr.Wrap(err, "failed to post message to slack", goerr.V("blocks", blocks))
	}

	thread := &ThreadService{
		channelID:     channelID,
		threadID:      timestamp,
		client:        x.client,
		slackMetadata: x.slackMetadata,
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

func (x *Service) ShowResolveAlertModal(ctx context.Context, alert alert.Alert, triggerID string) error {
	req := buildResolveModalViewRequest(model.CallbackSubmitResolveAlert, alert.ID.String())
	if _, err := x.client.OpenView(triggerID, req); err != nil {
		return goerr.Wrap(err, "failed to open view", goerr.V("req", req))
	}

	return nil
}

func (x *Service) ShowResolveListModal(ctx context.Context, list alert.List, triggerID string) error {
	req := buildResolveModalViewRequest(model.CallbackSubmitResolveList, list.ID.String())
	if _, err := x.client.OpenView(triggerID, req); err != nil {
		return goerr.Wrap(err, "failed to open view", goerr.V("req", req))
	}

	return nil
}

type ThreadService struct {
	channelID string
	threadID  string
	client    interfaces.SlackClient
	slackMetadata
}

func (x *ThreadService) ChannelID() string { return x.channelID }
func (x *ThreadService) ThreadID() string  { return x.threadID }

func (x *ThreadService) UpdateAlert(ctx context.Context, alert alert.Alert) error {
	blocks := buildAlertBlocks(alert)

	_, _, _, err := x.client.UpdateMessageContext(
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

func (x *ThreadService) PostTicket(ctx context.Context, ticket ticket.Ticket, alerts alert.Alerts) (string, error) {
	blocks := buildTicketBlocks(ticket, alerts, x.slackMetadata)

	if ticket.SlackMessageID == "" {
		_, ts, err := x.client.PostMessageContext(
			ctx,
			x.channelID,
			slack.MsgOptionBlocks(blocks...),
			slack.MsgOptionBroadcast(),
			slack.MsgOptionTS(ticket.SlackThread.ThreadID),
		)
		if err != nil {
			return "", goerr.Wrap(err, "failed to post message to slack", goerr.V("channelID", x.channelID), goerr.V("blocks", blocks))
		}
		return ts, nil
	}

	_, _, _, err := x.client.UpdateMessageContext(
		ctx,
		ticket.SlackThread.ChannelID,
		ticket.SlackMessageID,
		slack.MsgOptionBlocks(blocks...),
	)
	if err != nil {
		return "", goerr.Wrap(err, "failed to update message to slack", goerr.V("channelID", x.channelID), goerr.V("threadID", x.threadID), goerr.V("blocks", blocks))
	}

	return ticket.SlackMessageID, nil
}

func (x *ThreadService) AttachFile(ctx context.Context, title, fileName string, data []byte) error {
	if len(data) == 0 {
		msg := fmt.Sprintf("No data to attach: %s", title)
		if _, _, err := x.client.PostMessageContext(ctx, x.channelID, slack.MsgOptionText(msg, false), slack.MsgOptionTS(x.threadID)); err != nil {
			return goerr.Wrap(err, "failed to post no data message to slack", goerr.V("title", title), goerr.V("fileName", fileName))
		}
		return nil
	}

	_, err := x.client.UploadFileV2Context(ctx, slack.UploadFileV2Parameters{
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

func (x *ThreadService) Reply(ctx context.Context, message string) {
	blocks := []slack.Block{
		slack.NewContextBlock(
			"",
			slack.NewTextBlockObject(slack.MarkdownType, message, false, false),
		),
	}

	_, _, err := x.client.PostMessageContext(
		ctx,
		x.channelID,
		slack.MsgOptionBlocks(blocks...),
		slack.MsgOptionTS(x.threadID),
	)

	if err != nil {
		errs.Handle(ctx, goerr.Wrap(err, "failed to reply to slack",
			goerr.V("channelID", x.channelID),
			goerr.V("threadID", x.threadID),
			goerr.V("message", message),
			goerr.V("blocks", blocks),
		))
	}
}

func (x *ThreadService) NewStateFunc(ctx context.Context, message string) func(ctx context.Context, msg string) {
	var msgID string
	blocks := buildStateMessageBlocks(message, []string{})

	if len(blocks) > 0 {
		msgID = x.postInitialMessage(ctx, blocks)
	}

	var messages []string
	var mutex sync.Mutex

	return func(ctx context.Context, appendMsg string) {
		mutex.Lock()
		defer mutex.Unlock()
		messages = append(messages, appendMsg)

		blocks := buildStateMessageBlocks(message, messages)
		if len(blocks) == 0 {
			return
		}

		if msgID == "" {
			msgID = x.postInitialMessage(ctx, blocks)
			return
		}

		x.updateMessage(ctx, msgID, blocks)
	}
}

func (x *ThreadService) postInitialMessage(ctx context.Context, blocks []slack.Block) string {
	_, ts, err := x.client.PostMessageContext(ctx,
		x.channelID,
		slack.MsgOptionBlocks(blocks...),
		slack.MsgOptionTS(x.threadID),
	)

	if err != nil {
		errs.Handle(ctx, goerr.Wrap(err, "failed to post message to slack",
			goerr.V("channelID", x.channelID),
			goerr.V("threadID", x.threadID),
			goerr.V("blocks", blocks),
		))
		return ""
	}

	return ts
}

func (x *ThreadService) updateMessage(ctx context.Context, msgID string, blocks []slack.Block) {
	_, _, _, err := x.client.UpdateMessageContext(ctx,
		x.channelID,
		msgID,
		slack.MsgOptionBlocks(blocks...),
	)
	if err != nil {
		errs.Handle(ctx, goerr.Wrap(err, "failed to update message to slack",
			goerr.V("channelID", x.channelID),
			goerr.V("threadID", x.threadID),
			goerr.V("msgID", msgID),
			goerr.V("blocks", blocks),
		))
	}
}

func (x *ThreadService) PostFinding(ctx context.Context, finding ticket.Finding) error {
	blocks := buildFindingBlocks(finding)

	_, _, err := x.client.PostMessageContext(
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

func buildFindingBlocks(finding ticket.Finding) []slack.Block {
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

func (x *ThreadService) PostPolicyDiff(ctx context.Context, diff *policy.Diff) error {
	for fileName, diffData := range diff.DiffPolicy() {
		_, err := x.client.UploadFileV2Context(ctx, slack.UploadFileV2Parameters{
			Channel:         x.channelID,
			Reader:          bytes.NewReader([]byte(diffData)),
			FileSize:        len(diffData),
			Filename:        fileName + ".diff",
			Title:           "✍️ " + diff.Title + " (" + fileName + ")",
			ThreadTimestamp: x.threadID,
		})
		if err != nil {
			return goerr.Wrap(err, "failed to upload file to slack")
		}
	}

	blocks := []slack.Block{
		slack.NewDividerBlock(),
		slack.NewActionBlock(
			"create_pr",
			slack.NewButtonBlockElement(
				"create_pr",
				diff.ID.String(),
				slack.NewTextBlockObject("plain_text", "Create Pull Request", false, false),
			),
		),
	}

	_, _, err := x.client.PostMessageContext(ctx,
		x.channelID,
		slack.MsgOptionBlocks(blocks...),
		slack.MsgOptionTS(x.threadID),
	)
	if err != nil {
		return goerr.Wrap(err, "failed to post policy diff to slack", goerr.V("blocks", blocks))
	}

	return nil
}

func (x *ThreadService) PostAlerts(ctx context.Context, alerts alert.Alerts) error {
	blocks := buildAlertsBlocks(alerts, x.slackMetadata)

	_, _, err := x.client.PostMessageContext(ctx,
		x.channelID,
		slack.MsgOptionBlocks(blocks...),
		slack.MsgOptionTS(x.threadID),
	)
	if err != nil {
		return goerr.Wrap(err, "failed to post alerts to slack", goerr.V("blocks", blocks))
	}

	return nil
}

func (x *ThreadService) PostAlertList(ctx context.Context, list *alert.List) error {
	blocks := buildNewAlertListBlocks(list, x.slackMetadata)

	_, _, err := x.client.PostMessageContext(ctx,
		x.channelID,
		slack.MsgOptionBlocks(blocks...),
		slack.MsgOptionTS(x.threadID),
		slack.MsgOptionBroadcast(),
	)
	if err != nil {
		return goerr.Wrap(err, "failed to post alert list to slack", goerr.V("blocks", blocks))
	}

	return nil
}

func buildNewAlertListBlocks(list *alert.List, metadata slackMetadata) []slack.Block {
	blocks := []slack.Block{
		slack.NewHeaderBlock(
			slack.NewTextBlockObject("plain_text", "📑 New list", false, false),
		),
		slack.NewDividerBlock(),
	}

	blocks = append(blocks, buildAlertListBlocks(list, metadata)...)

	return blocks
}

func (x *ThreadService) PostAlertClusters(ctx context.Context, clusters []alert.List) error {
	blocks := buildAlertClustersBlocks(clusters, x.slackMetadata)

	_, _, err := x.client.PostMessageContext(ctx,
		x.channelID,
		slack.MsgOptionBlocks(blocks...),
		slack.MsgOptionTS(x.threadID),
	)
	if err != nil {
		return goerr.Wrap(err, "failed to post alert clusters to slack", goerr.V("blocks", blocks))
	}

	return nil
}

func (x *Service) ShowResolveAlertListModal(ctx context.Context, list alert.List, triggerID string) error {
	return nil
}
