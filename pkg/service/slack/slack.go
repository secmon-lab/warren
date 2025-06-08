package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	model "github.com/secmon-lab/warren/pkg/domain/model/slack"
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
	// User profile cache
	profileCache      map[string]*userProfileCache
	profileCacheMutex sync.RWMutex
	// Singleton rate-limited updater shared across all threads
	rateLimitedUpdater AlertUpdater
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

// ServiceOption represents a configuration option for Service
type ServiceOption func(*Service)

// WithUpdaterOptions sets options for the rate-limited updater
func WithUpdaterOptions(opts ...UpdaterOption) ServiceOption {
	return func(s *Service) {
		s.rateLimitedUpdater = NewRateLimitedUpdater(s.client, opts...)
	}
}

func New(client interfaces.SlackClient, channelID string, opts ...ServiceOption) (*Service, error) {
	s := &Service{
		channelID:          channelID,
		client:             client,
		iconCache:          make(map[string]*userIconCache),
		iconCacheExpiry:    time.Hour, // 1時間でキャッシュ更新
		profileCache:       make(map[string]*userProfileCache),
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

func (x *Service) ToMsgURL(channelID, threadID string) string {
	return x.slackMetadata.ToMsgURL(channelID, threadID)
}

func (x *Service) NewThread(thread model.Thread) *ThreadService {
	return &ThreadService{
		slackMetadata:      x.slackMetadata,
		channelID:          thread.ChannelID,
		threadID:           thread.ThreadID,
		client:             x.client,
		rateLimitedUpdater: x.rateLimitedUpdater,
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
		channelID:          channelID,
		threadID:           timestamp,
		client:             x.client,
		rateLimitedUpdater: x.rateLimitedUpdater,
		slackMetadata:      x.slackMetadata,
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

func (x *Service) UpdateAlerts(ctx context.Context, alerts alert.Alerts) {
	for _, alert := range alerts {
		x.rateLimitedUpdater.UpdateAlert(ctx, *alert)
	}
}

type ThreadService struct {
	channelID          string
	threadID           string
	client             interfaces.SlackClient
	rateLimitedUpdater AlertUpdater
	slackMetadata
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

func (x *ThreadService) UpdateAlert(ctx context.Context, alert alert.Alert) error {
	x.rateLimitedUpdater.UpdateAlert(ctx, alert)
	return nil // Return immediately, processing is done asynchronously
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

// PostTicketOutsideThread posts a ticket outside the current thread and returns the new thread service
func (x *ThreadService) PostTicketOutsideThread(ctx context.Context, ticket ticket.Ticket, alerts alert.Alerts) (*ThreadService, string, error) {
	blocks := buildTicketBlocks(ticket, alerts, x.slackMetadata)

	channelID, ts, err := x.client.PostMessageContext(
		ctx,
		x.channelID,
		slack.MsgOptionBlocks(blocks...),
		slack.MsgOptionBroadcast(),
	)
	if err != nil {
		return nil, "", goerr.Wrap(err, "failed to post ticket outside thread", goerr.V("channelID", x.channelID), goerr.V("blocks", blocks))
	}

	newThread := &ThreadService{
		channelID:          channelID,
		threadID:           ts,
		client:             x.client,
		rateLimitedUpdater: x.rateLimitedUpdater,
		slackMetadata:      x.slackMetadata,
	}

	return newThread, ts, nil
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
	alerts, err := list.Alerts()
	if err != nil {
		return goerr.Wrap(err, "failed to get alerts")
	}
	blocks := buildNewAlertListBlocks(list, alerts, x.slackMetadata)

	_, _, err = x.client.PostMessageContext(ctx,
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

func buildNewAlertListBlocks(list *alert.List, alerts alert.Alerts, metadata slackMetadata) []slack.Block {
	blocks := []slack.Block{
		slack.NewHeaderBlock(
			slack.NewTextBlockObject("plain_text", fmt.Sprintf("📑 New list with %d alerts", len(alerts)), false, false),
		),
		slack.NewDividerBlock(),
	}

	blocks = append(blocks, buildAlertListBlocks(list, alerts, metadata)...)

	return blocks
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
		slack.MsgOptionBroadcast(),
	)
	if err != nil {
		return goerr.Wrap(err, "failed to post ticket list to slack", goerr.V("blocks", blocks))
	}

	return nil
}

func (x *Service) ShowResolveAlertListModal(ctx context.Context, list *alert.List, triggerID string) error {
	return nil
}

func (x *Service) ShowBindToTicketModal(ctx context.Context, callbackID model.CallbackID, tickets []*ticket.Ticket, triggerID string, metadata string) error {
	req := buildBindToTicketModalViewRequest(ctx, callbackID, tickets, metadata)
	if _, err := x.client.OpenView(triggerID, req); err != nil {
		return goerr.Wrap(err, "failed to open view", goerr.V("req", req))
	}

	return nil
}

func (x *Service) ShowResolveTicketModal(ctx context.Context, ticket *ticket.Ticket, triggerID string) error {
	req := buildResolveTicketModalViewRequest(model.CallbackSubmitResolveTicket, ticket)
	if _, err := x.client.OpenView(triggerID, req); err != nil {
		return goerr.Wrap(err, "failed to open view", goerr.V("req", req))
	}

	return nil
}

func (x *ThreadService) PostAlert(ctx context.Context, alert alert.Alert) error {
	blocks := buildAlertBlocks(alert)
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

	// Fetch user info from Slack
	user, err := x.client.GetUserInfo(userID)
	if err != nil {
		return nil, "", goerr.Wrap(err, "failed to get user info from slack", goerr.V("user_id", userID))
	}

	// Check if user data is valid
	if user == nil {
		return nil, "", goerr.New("user data is nil", goerr.V("user_id", userID))
	}

	// Get profile image URL (use image_192 for good quality)
	imageURL := user.Profile.Image192
	if imageURL == "" {
		// Fallback to other sizes if image_192 is not available
		imageURL = user.Profile.Image512
		if imageURL == "" {
			imageURL = user.Profile.Image72
			if imageURL == "" {
				return nil, "", goerr.New("no profile image available", goerr.V("user_id", userID))
			}
		}
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

	// Fetch user info from Slack
	user, err := x.client.GetUserInfo(userID)
	if err != nil {
		return "", goerr.Wrap(err, "failed to get user info from slack", goerr.V("user_id", userID))
	}

	// Check if user data is valid
	if user == nil {
		return "", goerr.New("user data is nil", goerr.V("user_id", userID))
	}

	// Get the display name, with fallback to real name
	displayName := user.Profile.DisplayName
	if displayName == "" {
		displayName = user.Profile.RealName
		if displayName == "" {
			return "", goerr.New("no user name available", goerr.V("user_id", userID))
		}
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
