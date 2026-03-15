package slack

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/m-mizutani/goerr/v2"
	model "github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	slackSDK "github.com/slack-go/slack"
)

const (
	historyMaxMessages = 100
	historyMaxAge      = 24 * time.Hour
	threadRootMessages = 50
	threadRootMaxAge   = 12 * time.Hour
)

// GetMessageHistory retrieves conversation history from Slack based on the message context.
// For root messages, it fetches recent channel messages.
// For thread messages, it fetches thread replies plus preceding root messages for context.
func (x *Service) GetMessageHistory(ctx context.Context, slackMsg *model.Message) ([]model.HistoryMessage, error) {
	if slackMsg.InThread() {
		return x.getThreadHistory(ctx, slackMsg)
	}
	return x.getRootHistory(ctx, slackMsg)
}

// getRootHistory fetches recent channel messages (up to 24h or 100 messages).
func (x *Service) getRootHistory(ctx context.Context, slackMsg *model.Message) ([]model.HistoryMessage, error) {
	oldest := time.Now().Add(-historyMaxAge)

	resp, err := x.client.GetConversationHistoryContext(ctx, &slackSDK.GetConversationHistoryParameters{
		ChannelID: slackMsg.ChannelID(),
		Oldest:    formatSlackTimestamp(oldest),
		Limit:     historyMaxMessages,
		Inclusive: false,
	})
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get conversation history")
	}

	return x.convertMessages(ctx, resp.Messages, false), nil
}

// getThreadHistory fetches thread replies and preceding root messages for context.
func (x *Service) getThreadHistory(ctx context.Context, slackMsg *model.Message) ([]model.HistoryMessage, error) {
	logger := logging.From(ctx)

	// Get thread replies
	threadOldest := time.Now().Add(-historyMaxAge)
	replies, _, _, err := x.client.GetConversationRepliesContext(ctx, &slackSDK.GetConversationRepliesParameters{
		ChannelID: slackMsg.ChannelID(),
		Timestamp: slackMsg.ThreadID(),
		Oldest:    formatSlackTimestamp(threadOldest),
		Limit:     historyMaxMessages,
		Inclusive: true,
	})
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get conversation replies")
	}

	threadMessages := x.convertMessages(ctx, replies, true)

	// Get root messages before thread start for additional context
	threadStartTS := slackMsg.ThreadID()
	threadStartTime, parseErr := parseSlackTimestamp(threadStartTS)
	if parseErr != nil {
		logger.Warn("failed to parse thread start timestamp, skipping root context", "error", parseErr, "ts", threadStartTS)
		return threadMessages, nil
	}

	rootOldest := threadStartTime.Add(-threadRootMaxAge)
	resp, err := x.client.GetConversationHistoryContext(ctx, &slackSDK.GetConversationHistoryParameters{
		ChannelID: slackMsg.ChannelID(),
		Latest:    threadStartTS,
		Oldest:    formatSlackTimestamp(rootOldest),
		Limit:     threadRootMessages,
		Inclusive: false,
	})
	if err != nil {
		logger.Warn("failed to get root context messages", "error", err)
		return threadMessages, nil
	}

	rootMessages := x.convertMessages(ctx, resp.Messages, false)

	// Combine: root context first, then thread messages
	result := make([]model.HistoryMessage, 0, len(rootMessages)+len(threadMessages))
	result = append(result, rootMessages...)
	result = append(result, threadMessages...)
	return result, nil
}

// convertMessages converts Slack SDK messages to HistoryMessage models.
func (x *Service) convertMessages(ctx context.Context, messages []slackSDK.Message, isThread bool) []model.HistoryMessage {
	result := make([]model.HistoryMessage, 0, len(messages))
	for _, m := range messages {
		ts, err := parseSlackTimestamp(m.Timestamp)
		if err != nil {
			logging.From(ctx).Warn("failed to parse message timestamp", "error", err, "ts", m.Timestamp)
			continue
		}

		userName := x.resolveUserName(ctx, m.User)

		result = append(result, model.HistoryMessage{
			UserID:    m.User,
			UserName:  userName,
			Text:      m.Text,
			Timestamp: ts,
			IsBot:     m.BotID != "" || m.SubType == "bot_message",
			IsThread:  isThread,
		})
	}
	return result
}

// resolveUserName looks up the display name for a user ID.
func (x *Service) resolveUserName(ctx context.Context, userID string) string {
	if userID == "" {
		return "unknown"
	}

	name, err := x.GetUserProfile(ctx, userID)
	if err != nil {
		logging.From(ctx).Debug("failed to resolve user name", "error", err, "user_id", userID)
		return userID
	}
	if name != "" {
		return name
	}
	return userID
}

// formatSlackTimestamp converts a time.Time to Slack's timestamp format.
func formatSlackTimestamp(t time.Time) string {
	return strconv.FormatInt(t.Unix(), 10) + ".000000"
}

// parseSlackTimestamp converts a Slack timestamp string to time.Time.
func parseSlackTimestamp(ts string) (time.Time, error) {
	parts := strings.SplitN(ts, ".", 2)
	if len(parts) == 0 || parts[0] == "" {
		return time.Time{}, goerr.New("empty timestamp")
	}

	sec, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return time.Time{}, goerr.Wrap(err, "failed to parse timestamp seconds", goerr.V("ts", ts))
	}

	return time.Unix(sec, 0), nil
}
