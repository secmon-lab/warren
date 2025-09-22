package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	model "github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/tag"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/test"

	"github.com/slack-go/slack"
)

type userIconCache struct {
	ImageData []byte
	ExpiresAt time.Time
}

type userProfileCache struct {
	Name      string
	ExpiresAt time.Time
}

const UserProfileCacheExpiry = 10 * time.Minute

type Service struct {
	channelID string
	client    interfaces.SlackClient
	slackMetadata
	// User icon cache
	iconCache       map[string]*userIconCache
	iconCacheMutex  sync.RWMutex
	iconCacheExpiry time.Duration
	// User icon lock per userID to prevent concurrent API calls
	iconLocks      map[string]*sync.Mutex
	iconLocksMutex sync.Mutex
	// User profile cache
	profileCache      map[string]*userProfileCache
	profileCacheMutex sync.RWMutex
	// User profile lock per userID to prevent concurrent API calls
	profileLocks      map[string]*sync.Mutex
	profileLocksMutex sync.Mutex
	// Singleton rate-limited updater shared across all threads
	rateLimitedUpdater AlertUpdater
	// Frontend URL for generating ticket detail URLs
	frontendURL string
}

type slackMetadata struct {
	teamID          string
	teamName        string
	workspaceDomain string // Add workspace domain for external URLs
	botID           string
	userID          string
	enterpriseID    string
}

func (x slackMetadata) ToMsgURL(channelID, threadID string) string {
	if x.enterpriseID == "" {
		return fmt.Sprintf("https://%s.slack.com/archives/%s/p%s", x.teamName, channelID, threadID)
	}

	return fmt.Sprintf("https://%s.slack.com/archives/%s/p%s", x.enterpriseID, channelID, threadID)
}

// ToExternalMsgURL generates an external Slack URL that can be accessed from outside Slack
// This is different from ToMsgURL which is designed for internal Slack navigation
func (x slackMetadata) ToExternalMsgURL(channelID, messageID, threadID string) string {
	// Format timestamp for URL (remove decimal point and add 'p' prefix)
	formattedTimestamp := "p" + strings.ReplaceAll(messageID, ".", "")

	baseURL := fmt.Sprintf("https://%s.slack.com", x.workspaceDomain)

	// Basic external message URL format
	msgURL := fmt.Sprintf("%s/archives/%s/%s", baseURL, channelID, formattedTimestamp)

	// If this is a message in a thread, add thread parameters
	if threadID != "" && threadID != messageID {
		msgURL += fmt.Sprintf("?thread_ts=%s&cid=%s", threadID, channelID)
	}

	return msgURL
}

// ServiceOption represents a configuration option for Service
type ServiceOption func(*Service)

// WithUpdaterOptions sets options for the rate-limited updater
func WithUpdaterOptions(opts ...UpdaterOption) ServiceOption {
	return func(s *Service) {
		s.rateLimitedUpdater = NewRateLimitedUpdater(s.client, opts...)
	}
}

// WithFrontendURL sets the frontend URL for generating ticket detail URLs
func WithFrontendURL(frontendURL string) ServiceOption {
	return func(s *Service) {
		s.frontendURL = frontendURL
	}
}

func New(client interfaces.SlackClient, channelID string, opts ...ServiceOption) (*Service, error) {
	s := &Service{
		channelID:          channelID,
		client:             client,
		iconCache:          make(map[string]*userIconCache),
		iconCacheExpiry:    time.Hour, // 1 hour
		iconLocks:          make(map[string]*sync.Mutex),
		profileCache:       make(map[string]*userProfileCache),
		profileLocks:       make(map[string]*sync.Mutex),
		rateLimitedUpdater: NewRateLimitedUpdater(client), // Default updater
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
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

	// Get workspace domain from team.info API
	teamInfo, err := s.client.GetTeamInfo()
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get team info from slack")
	}
	if teamInfo != nil {
		s.slackMetadata.workspaceDomain = teamInfo.Domain
	} else {
		logging.Default().Error("failed to get team info from slack", "error", err)
	}

	return s, nil
}

func NewTestService(t *testing.T) *Service {
	envs := test.NewEnvVars(t, "TEST_SLACK_CHANNEL_ID", "TEST_SLACK_OAUTH_TOKEN")
	client := slack.New(envs.Get("TEST_SLACK_OAUTH_TOKEN"))

	// Use fast interval for testing
	svc, err := New(client, envs.Get("TEST_SLACK_CHANNEL_ID"),
		WithUpdaterOptions(WithInterval(100*time.Millisecond)))
	gt.NoError(t, err).Required()

	return svc
}

func (x *Service) IsBotUser(userID string) bool {
	return x.userID == userID
}

func (x *Service) BotID() string {
	return x.botID
}

func (x *Service) TeamID() string {
	return x.teamID
}

func (x *Service) DefaultChannelID() string {
	return x.channelID
}

func (x *Service) ToMsgURL(channelID, threadID string) string {
	return x.slackMetadata.ToMsgURL(channelID, threadID)
}

func (x *Service) ToExternalMsgURL(channelID, threadID string) string {
	return x.slackMetadata.ToExternalMsgURL(channelID, threadID, "")
}

func (x *Service) NewThread(thread model.Thread) interfaces.SlackThreadService {
	return &ThreadService{
		slackMetadata:      x.slackMetadata,
		channelID:          thread.ChannelID,
		threadID:           thread.ThreadID,
		client:             x.client,
		rateLimitedUpdater: x.rateLimitedUpdater,
		frontendURL:        x.frontendURL,
	}
}

// PostMessage posts a message to the channel and returns the thread. It's just for testing.
func (x *Service) PostMessage(ctx context.Context, message string) (*ThreadService, error) {
	ch, thread, err := x.client.PostMessageContext(ctx, x.channelID, slack.MsgOptionText(message, false))
	if err != nil {
		return nil, err
	}

	threadSvc := x.NewThread(model.Thread{
		ChannelID: ch,
		ThreadID:  thread,
	}).(*ThreadService)
	return threadSvc, nil
}

// resolveChannel determines the target channel for the alert
func (x *Service) resolveChannel(alert *alert.Alert) string {
	if alert.Metadata.Channel != "" {
		return x.normalizeChannel(alert.Metadata.Channel)
	}
	return x.channelID
}

// normalizeChannel removes # prefix and trims whitespace from channel name
func (x *Service) normalizeChannel(channel string) string {
	channel = strings.TrimSpace(channel)
	channel = strings.TrimPrefix(channel, "#")
	return channel
}

// postAlertToChannel posts an alert to the specified channel
func (x *Service) postAlertToChannel(ctx context.Context, targetChannel string, alert *alert.Alert) (interfaces.SlackThreadService, error) {
	blocks := buildAlertBlocks(*alert)

	channelID, timestamp, err := x.client.PostMessageContext(
		ctx,
		targetChannel,
		slack.MsgOptionBlocks(blocks...),
	)

	if err != nil {
		return nil, goerr.Wrap(err, "failed to post message to slack",
			goerr.V("channel", targetChannel),
			goerr.V("blocks", blocks))
	}

	thread := &ThreadService{
		channelID:          channelID,
		threadID:           timestamp,
		client:             x.client,
		rateLimitedUpdater: x.rateLimitedUpdater,
		slackMetadata:      x.slackMetadata,
		frontendURL:        x.frontendURL,
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

// postAlertWithFallback attempts to post to default channel with a warning after initial failure
func (x *Service) postAlertWithFallback(ctx context.Context, failedChannel string, alert *alert.Alert, originalErr error) (interfaces.SlackThreadService, error) {
	logger := logging.From(ctx)
	logger.Warn("Failed to post alert to channel, falling back to default",
		"failed_channel", failedChannel,
		"default_channel", x.channelID,
		"error", originalErr)

	// Create a copy of the alert with warning message prepended to description
	fallbackAlert := *alert
	fallbackAlert.Metadata.Description = fmt.Sprintf("⚠️ Failed to send to <#%s>: %v\nFalling back to default channel.\n\n%s",
		failedChannel, originalErr, alert.Metadata.Description)

	// Use existing postAlertToChannel method with modified alert
	return x.postAlertToChannel(ctx, x.channelID, &fallbackAlert)
}

func (x *Service) PostAlert(ctx context.Context, alert *alert.Alert) (interfaces.SlackThreadService, error) {
	// Determine target channel based on alert metadata
	targetChannel := x.resolveChannel(alert)

	// Try to post to the target channel
	thread, err := x.postAlertToChannel(ctx, targetChannel, alert)
	if err != nil && targetChannel != x.channelID {
		// If posting to a policy-specified channel failed, fallback to default
		return x.postAlertWithFallback(ctx, targetChannel, alert, err)
	}

	return thread, err
}

func (x *Service) UpdateAlerts(ctx context.Context, alerts alert.Alerts) {
	for _, alert := range alerts {
		x.rateLimitedUpdater.UpdateAlert(ctx, *alert)
	}
}

// PostTicket posts a ticket to a new thread and returns the thread service
func (x *Service) PostTicket(ctx context.Context, ticket *ticket.Ticket, alerts alert.Alerts) (interfaces.SlackThreadService, string, error) {
	blocks := buildTicketBlocks(*ticket, alerts, x.slackMetadata, x.frontendURL)

	channelID, ts, err := x.client.PostMessageContext(
		ctx,
		x.channelID,
		slack.MsgOptionBlocks(blocks...),
	)
	if err != nil {
		return nil, "", goerr.Wrap(err, "failed to post ticket", goerr.V("channelID", x.channelID), goerr.V("blocks", blocks))
	}

	newThread := &ThreadService{
		channelID:          channelID,
		threadID:           ts,
		client:             x.client,
		rateLimitedUpdater: x.rateLimitedUpdater,
		slackMetadata:      x.slackMetadata,
		frontendURL:        x.frontendURL,
	}

	return newThread, ts, nil
}

type ThreadService struct {
	channelID          string
	threadID           string
	client             interfaces.SlackClient
	rateLimitedUpdater AlertUpdater
	slackMetadata
	frontendURL string
}

func (x *ThreadService) ChannelID() string { return x.channelID }
func (x *ThreadService) ThreadID() string  { return x.threadID }

func (x *ThreadService) Entity() *model.Thread {
	return &model.Thread{
		ChannelID: x.channelID,
		ThreadID:  x.threadID,
		TeamID:    x.teamID,
	}
}

func (x *ThreadService) ToExternalMsgURL() string {
	return x.slackMetadata.ToExternalMsgURL(x.channelID, x.threadID, "")
}

func (x *ThreadService) UpdateAlert(ctx context.Context, alert alert.Alert) error {
	x.rateLimitedUpdater.UpdateAlert(ctx, alert)
	return nil // Return immediately, processing is done asynchronously
}

func (x *ThreadService) PostTicket(ctx context.Context, ticket *ticket.Ticket, alerts alert.Alerts) (string, error) {
	blocks := buildTicketBlocks(*ticket, alerts, x.slackMetadata, x.frontendURL)

	if ticket.SlackMessageID == "" {
		_, ts, err := x.client.PostMessageContext(
			ctx,
			x.channelID,
			slack.MsgOptionBlocks(blocks...),
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

// PostLinkToTicket posts a link to a ticket in the current thread
func (x *ThreadService) PostLinkToTicket(ctx context.Context, ticketURL, ticketTitle string) error {
	message := fmt.Sprintf("🎫 Ticket created: <%s|%s>", ticketURL, ticketTitle)

	_, _, err := x.client.PostMessageContext(
		ctx,
		x.channelID,
		slack.MsgOptionText(message, false),
		slack.MsgOptionTS(x.threadID),
	)
	if err != nil {
		return goerr.Wrap(err, "failed to post link to ticket", goerr.V("channelID", x.channelID), goerr.V("threadID", x.threadID), goerr.V("message", message))
	}

	return nil
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
		slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.MarkdownType, message, false, false),
			nil,
			nil,
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
	blocks := buildStateMessageBlocks([]string{message})

	if len(blocks) > 0 {
		msgID = x.postInitialMessage(ctx, blocks)
	}

	messages := []string{message}
	var mutex sync.Mutex

	return func(ctx context.Context, appendMsg string) {
		mutex.Lock()
		defer mutex.Unlock()

		messages = append(messages, appendMsg)
		blocks := buildStateMessageBlocks(messages)
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

func (x *ThreadService) NewUpdatableMessage(ctx context.Context, initialMessage string) func(ctx context.Context, newMsg string) {
	var msgID string
	var mutex sync.Mutex

	return func(ctx context.Context, newMsg string) {
		mutex.Lock()
		defer mutex.Unlock()

		blocks := buildStateMessageBlocks([]string{newMsg})
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

// NewTraceMessage creates a new trace message function that posts new context blocks
// when byte limits are exceeded instead of updating existing ones
func (x *ThreadService) NewTraceMessage(ctx context.Context, initialMessage string) func(ctx context.Context, traceMsg string) {
	var currentMsgID string
	var mutex sync.Mutex
	var currentMessages []string
	var currentBytes int

	const maxBlockBytes = 1900 // Leave some buffer for safety (Slack limit is 2000 bytes)

	// Add initial message if provided
	if initialMessage != "" {
		currentMessages = append(currentMessages, initialMessage)
		currentBytes = len([]byte(initialMessage))
		blocks := buildAccumulatedTraceMessageBlocks(currentMessages)
		if len(blocks) > 0 {
			currentMsgID = x.postInitialMessage(ctx, blocks)
		}
	}

	return func(ctx context.Context, traceMsg string) {
		mutex.Lock()
		defer mutex.Unlock()

		if traceMsg == "" {
			return
		}

		msgBytes := len([]byte(traceMsg))
		separatorBytes := 0
		if len(currentMessages) > 0 {
			separatorBytes = 1 // newline separator
		}

		// If adding this message would exceed the byte limit, create a new context block
		if currentBytes+msgBytes+separatorBytes > maxBlockBytes && len(currentMessages) > 0 {
			// Post new message with just the new trace message
			currentMessages = []string{traceMsg}
			currentBytes = msgBytes
			blocks := buildAccumulatedTraceMessageBlocks(currentMessages)
			if len(blocks) > 0 {
				currentMsgID = x.postInitialMessage(ctx, blocks)
			}
		} else {
			// Add to current messages and update existing message
			currentMessages = append(currentMessages, traceMsg)
			currentBytes += msgBytes + separatorBytes

			blocks := buildAccumulatedTraceMessageBlocks(currentMessages)
			if len(blocks) == 0 {
				return
			}

			if currentMsgID == "" {
				currentMsgID = x.postInitialMessage(ctx, blocks)
				return
			}

			x.updateMessage(ctx, currentMsgID, blocks)
		}
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

func (x *ThreadService) PostFinding(ctx context.Context, finding *ticket.Finding) error {
	blocks := buildFindingBlocks(*finding)

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

// PostComment posts a simple text comment to the thread
func (x *ThreadService) PostComment(ctx context.Context, comment string) error {
	_, _, err := x.client.PostMessageContext(
		ctx,
		x.channelID,
		slack.MsgOptionText(comment, false),
		slack.MsgOptionTS(x.threadID),
	)
	if err != nil {
		return goerr.Wrap(err, "failed to post comment to slack", goerr.V("comment", comment))
	}

	return nil
}

// PostCommentWithMessageID posts a simple text comment to the thread and returns the message ID
func (x *ThreadService) PostCommentWithMessageID(ctx context.Context, comment string) (string, error) {
	_, ts, err := x.client.PostMessageContext(
		ctx,
		x.channelID,
		slack.MsgOptionText(comment, false),
		slack.MsgOptionTS(x.threadID),
	)
	if err != nil {
		return "", goerr.Wrap(err, "failed to post comment to slack", goerr.V("comment", comment))
	}

	return ts, nil
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

func (x *ThreadService) PostAlertList(ctx context.Context, list *alert.List) (string, error) {
	alerts, err := list.Alerts()
	if err != nil {
		return "", goerr.Wrap(err, "failed to get alerts")
	}
	blocks := buildNewAlertListBlocks(list, alerts, x.slackMetadata)

	_, ts, err := x.client.PostMessageContext(ctx,
		x.channelID,
		slack.MsgOptionBlocks(blocks...),
		slack.MsgOptionTS(x.threadID),
	)
	if err != nil {
		return "", goerr.Wrap(err, "failed to post alert list to slack", goerr.V("blocks", blocks))
	}

	return ts, nil
}

// UpdateAlertList updates an alert list message with completion status
func (x *ThreadService) UpdateAlertList(ctx context.Context, list *alert.List, status string) error {
	if list.SlackMessageID == "" {
		return goerr.New("alert list has no slack message ID")
	}

	alerts, err := list.Alerts()
	if err != nil {
		return goerr.Wrap(err, "failed to get alerts")
	}
	blocks := buildCompletedAlertListBlocks(list, alerts, x.slackMetadata, status)

	_, _, _, err = x.client.UpdateMessageContext(
		ctx,
		x.channelID,
		list.SlackMessageID,
		slack.MsgOptionBlocks(blocks...),
	)
	if err != nil {
		return goerr.Wrap(err, "failed to update alert list message", goerr.V("blocks", blocks))
	}

	return nil
}

func (x *ThreadService) PostAlertLists(ctx context.Context, clusters []*alert.List) error {
	blocks, err := buildAlertClustersBlocks(clusters, x.slackMetadata)
	if err != nil {
		return goerr.Wrap(err, "failed to build alert clusters blocks")
	}

	_, _, err = x.client.PostMessageContext(ctx,
		x.channelID,
		slack.MsgOptionBlocks(blocks...),
		slack.MsgOptionTS(x.threadID),
	)
	if err != nil {
		return goerr.Wrap(err, "failed to post alert clusters to slack", goerr.V("blocks", blocks))
	}

	return nil
}

func (x *ThreadService) PostTicketList(ctx context.Context, tickets []*ticket.Ticket) error {
	blocks := buildTicketListBlocks(ctx, tickets, x.slackMetadata)

	_, _, err := x.client.PostMessageContext(ctx,
		x.channelID,
		slack.MsgOptionBlocks(blocks...),
		slack.MsgOptionTS(x.threadID),
	)
	if err != nil {
		return goerr.Wrap(err, "failed to post ticket list to slack", goerr.V("blocks", blocks))
	}

	return nil
}

func (x *Service) ShowBindToTicketModal(ctx context.Context, callbackID model.CallbackID, tickets []*ticket.Ticket, triggerID string, metadata string) error {
	req := buildBindToTicketModalViewRequest(ctx, callbackID, tickets, metadata)
	if _, err := x.client.OpenView(triggerID, req); err != nil {
		return goerr.Wrap(err, "failed to open view",
			goerr.V("trigger_id", triggerID),
			goerr.V("callback_id", callbackID),
			goerr.V("metadata", metadata),
			goerr.V("request", req))
	}

	return nil
}

func (x *Service) ShowResolveTicketModal(ctx context.Context, ticket *ticket.Ticket, triggerID string, availableTags []*tag.Tag) error {
	req := buildResolveTicketModalViewRequest(model.CallbackSubmitResolveTicket, ticket, availableTags)
	if _, err := x.client.OpenView(triggerID, req); err != nil {
		return goerr.Wrap(err, "failed to open view",
			goerr.V("callback_id", req.CallbackID),
			goerr.V("ticket_id", ticket.ID.String()),
			goerr.V("trigger_id", triggerID),
			goerr.V("request", req))
	}

	return nil
}

func (x *Service) ShowSalvageModal(ctx context.Context, ticket *ticket.Ticket, unboundAlerts alert.Alerts, triggerID string) error {
	// Use threshold 0.9 for initial display
	req := buildSalvageModalViewRequest(model.CallbackSubmitSalvage, ticket, unboundAlerts, 0.9, "")
	if _, err := x.client.OpenView(triggerID, req); err != nil {
		return goerr.Wrap(err, "failed to open view",
			goerr.V("trigger_id", triggerID),
			goerr.V("ticket_id", ticket.ID.String()),
			goerr.V("request", req))
	}

	return nil
}

func (x *Service) UpdateSalvageModal(ctx context.Context, ticket *ticket.Ticket, unboundAlerts alert.Alerts, viewID string, threshold float64, keyword string) error {
	req := buildSalvageModalViewRequest(model.CallbackSubmitSalvage, ticket, unboundAlerts, threshold, keyword)

	// Update the view using the correct parameters
	if _, err := x.client.UpdateView(req, "", "", viewID); err != nil {
		return goerr.Wrap(err, "failed to update view",
			goerr.V("view_id", viewID),
			goerr.V("ticket_id", ticket.ID.String()),
			goerr.V("threshold", threshold),
			goerr.V("keyword", keyword),
			goerr.V("request", req))
	}

	return nil
}

func (x *ThreadService) PostAlert(ctx context.Context, alert *alert.Alert) error {
	blocks := buildAlertBlocks(*alert)
	_, _, err := x.client.PostMessageContext(
		ctx,
		x.channelID,
		slack.MsgOptionBlocks(blocks...),
		slack.MsgOptionTS(x.threadID),
	)
	if err != nil {
		return goerr.Wrap(err, "failed to post message to slack", goerr.V("channelID", x.channelID), goerr.V("threadID", x.threadID), goerr.V("blocks", blocks))
	}

	return nil
}

// GetUserIcon returns the user's icon image data
func (x *Service) GetUserIcon(ctx context.Context, userID string) ([]byte, string, error) {
	logger := logging.From(ctx)

	// Check cache first
	x.iconCacheMutex.RLock()
	if cached, exists := x.iconCache[userID]; exists && time.Now().Before(cached.ExpiresAt) {
		x.iconCacheMutex.RUnlock()
		logger.Debug("returning cached user icon", "user_id", userID)
		return cached.ImageData, "image/jpeg", nil
	}
	x.iconCacheMutex.RUnlock()

	// Get or create a lock for this userID to prevent concurrent API calls
	x.iconLocksMutex.Lock()
	if _, exists := x.iconLocks[userID]; !exists {
		x.iconLocks[userID] = &sync.Mutex{}
	}
	userLock := x.iconLocks[userID]
	x.iconLocksMutex.Unlock()

	// Lock for this specific userID
	userLock.Lock()
	defer userLock.Unlock()

	// Check cache again after acquiring lock in case another goroutine already fetched it
	x.iconCacheMutex.RLock()
	if cached, exists := x.iconCache[userID]; exists && time.Now().Before(cached.ExpiresAt) {
		x.iconCacheMutex.RUnlock()
		logger.Debug("returning cached user icon after lock", "user_id", userID)
		return cached.ImageData, "image/jpeg", nil
	}
	x.iconCacheMutex.RUnlock()

	// Try to get user icon URL - handle both regular users and bots
	imageURL, err := x.fetchUserImageURL(ctx, userID)
	if err != nil {
		return nil, "", goerr.Wrap(err, "failed to get user image URL", goerr.V("user_id", userID))
	}

	// Download the image
	imageData, contentType, err := x.downloadUserImage(ctx, imageURL)
	if err != nil {
		return nil, "", goerr.Wrap(err, "failed to download user icon", goerr.V("user_id", userID), goerr.V("image_url", imageURL))
	}

	// Cache the image
	x.iconCacheMutex.Lock()
	x.iconCache[userID] = &userIconCache{
		ImageData: imageData,
		ExpiresAt: time.Now().Add(x.iconCacheExpiry),
	}
	x.iconCacheMutex.Unlock()

	logger.Debug("cached user icon", "user_id", userID, "image_size", len(imageData))
	return imageData, contentType, nil
}

// downloadUserImage downloads an image from the given URL
func (x *Service) downloadUserImage(ctx context.Context, url string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, "", goerr.Wrap(err, "failed to create request")
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", goerr.Wrap(err, "failed to download image")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", goerr.New("failed to download image",
			goerr.V("status_code", resp.StatusCode),
			goerr.V("status", resp.Status))
	}

	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", goerr.Wrap(err, "failed to read image data")
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg" // default fallback
	}

	return imageData, contentType, nil
}

// ClearExpiredIconCache removes expired cache entries
func (x *Service) ClearExpiredIconCache() {
	x.iconCacheMutex.Lock()
	defer x.iconCacheMutex.Unlock()

	now := time.Now()
	for userID, cached := range x.iconCache {
		if now.After(cached.ExpiresAt) {
			delete(x.iconCache, userID)
		}
	}
}

// fetchUserImageURL fetches profile image URL for both regular users and bots
func (x *Service) fetchUserImageURL(ctx context.Context, userID string) (string, error) {
	// Try users.info API first (works for both users and some bots)
	user, err := x.client.GetUserInfo(userID)
	if err == nil && user != nil {
		// Try different image sizes in order of preference
		if user.Profile.Image192 != "" {
			return user.Profile.Image192, nil
		}
		if user.Profile.Image512 != "" {
			return user.Profile.Image512, nil
		}
		if user.Profile.Image72 != "" {
			return user.Profile.Image72, nil
		}
		if user.Profile.Image48 != "" {
			return user.Profile.Image48, nil
		}
		if user.Profile.Image32 != "" {
			return user.Profile.Image32, nil
		}
		if user.Profile.Image24 != "" {
			return user.Profile.Image24, nil
		}
	}

	// If users.info failed or returned no image, try bots.info for bot users
	bot, err := x.client.GetBotInfoContext(ctx, slack.GetBotInfoParameters{
		Bot: userID,
	})
	if err == nil && bot != nil {
		// Try bot icons in order of preference (only use existing fields)
		if bot.Icons.Image72 != "" {
			return bot.Icons.Image72, nil
		}
		if bot.Icons.Image48 != "" {
			return bot.Icons.Image48, nil
		}
		if bot.Icons.Image36 != "" {
			return bot.Icons.Image36, nil
		}
	}

	// No profile image available
	return "", goerr.New("no profile image available", goerr.V("user_id", userID))
}

// fetchUserDisplayName fetches display name for both regular users and bots
func (x *Service) fetchUserDisplayName(ctx context.Context, userID string) (string, error) {
	// First try users.info API (works for both users and some bots)
	user, err := x.client.GetUserInfo(userID)
	if err == nil && user != nil {
		// For regular users, try display name first, then real name
		if user.Profile.DisplayName != "" {
			return user.Profile.DisplayName, nil
		}
		if user.Profile.RealName != "" {
			return user.Profile.RealName, nil
		}
		// For bots, user.Name might be available
		if user.Name != "" {
			return user.Name, nil
		}
	}

	// If users.info failed or returned no useful name, try bots.info for bot users
	bot, err := x.client.GetBotInfoContext(ctx, slack.GetBotInfoParameters{
		Bot: userID,
	})
	if err == nil && bot != nil {
		if bot.Name != "" {
			return bot.Name, nil
		}
		if bot.AppID != "" {
			// Use app name as fallback
			return bot.AppID, nil
		}
	}

	// Fallback to userID if no display name found
	return userID, nil
}

// GetUserProfile returns the user's profile name
func (x *Service) GetUserProfile(ctx context.Context, userID string) (string, error) {
	logger := logging.From(ctx)

	// Check cache first
	x.profileCacheMutex.RLock()
	if cached, exists := x.profileCache[userID]; exists && time.Now().Before(cached.ExpiresAt) {
		x.profileCacheMutex.RUnlock()
		logger.Debug("returning cached user profile", "user_id", userID)
		return cached.Name, nil
	}
	x.profileCacheMutex.RUnlock()

	// Get or create a lock for this userID to prevent concurrent API calls
	x.profileLocksMutex.Lock()
	if _, exists := x.profileLocks[userID]; !exists {
		x.profileLocks[userID] = &sync.Mutex{}
	}
	userLock := x.profileLocks[userID]
	x.profileLocksMutex.Unlock()

	// Lock for this specific userID
	userLock.Lock()
	defer userLock.Unlock()

	// Check cache again after acquiring lock in case another goroutine already fetched it
	x.profileCacheMutex.RLock()
	if cached, exists := x.profileCache[userID]; exists && time.Now().Before(cached.ExpiresAt) {
		x.profileCacheMutex.RUnlock()
		logger.Debug("returning cached user profile after lock", "user_id", userID)
		return cached.Name, nil
	}
	x.profileCacheMutex.RUnlock()

	// Try to get user profile - handle both regular users and bots
	displayName, err := x.fetchUserDisplayName(ctx, userID)
	if err != nil {
		return "", goerr.Wrap(err, "failed to get user display name", goerr.V("user_id", userID))
	}

	// Cache the profile
	x.profileCacheMutex.Lock()
	x.profileCache[userID] = &userProfileCache{
		Name:      displayName,
		ExpiresAt: time.Now().Add(UserProfileCacheExpiry),
	}
	x.profileCacheMutex.Unlock()

	logger.Debug("cached user profile", "user_id", userID, "name", displayName)
	return displayName, nil
}

// ClearExpiredProfileCache removes expired profile cache entries
func (x *Service) ClearExpiredProfileCache() {
	x.profileCacheMutex.Lock()
	defer x.profileCacheMutex.Unlock()

	now := time.Now()
	for userID, cached := range x.profileCache {
		if now.After(cached.ExpiresAt) {
			delete(x.profileCache, userID)
		}
	}
}

// GetChannelName returns the channel name for the given channel ID
func (x *Service) GetChannelName(ctx context.Context, channelID string) (string, error) {
	input := &slack.GetConversationInfoInput{
		ChannelID:     channelID,
		IncludeLocale: false,
	}
	channel, err := x.client.GetConversationInfo(input)
	if err != nil {
		return "", goerr.Wrap(err, "failed to get channel info from slack", goerr.V("channel_id", channelID))
	}

	if channel == nil {
		return "", goerr.New("channel data is nil", goerr.V("channel_id", channelID))
	}

	return channel.Name, nil
}

// GetUserGroupName returns the user group name for the given group ID
func (x *Service) GetUserGroupName(ctx context.Context, groupID string) (string, error) {
	groups, err := x.client.GetUserGroups()
	if err != nil {
		return "", goerr.Wrap(err, "failed to get user groups from slack", goerr.V("group_id", groupID))
	}

	for _, group := range groups {
		if group.ID == groupID {
			return group.Handle, nil
		}
	}

	return "", goerr.New("user group not found", goerr.V("group_id", groupID))
}

// Stop gracefully stops the service and its rate-limited updater
func (x *Service) Stop() {
	if x.rateLimitedUpdater != nil {
		x.rateLimitedUpdater.Stop()
	}
}

// GetClient returns the underlying SlackClient
func (x *Service) GetClient() interfaces.SlackClient {
	return x.client
}

// ToTicketURL generates a URL to the ticket detail page in the frontend
func (x *Service) ToTicketURL(ticketID string) string {
	if x.frontendURL == "" {
		return ""
	}
	return fmt.Sprintf("%s/tickets/%s", x.frontendURL, ticketID)
}
