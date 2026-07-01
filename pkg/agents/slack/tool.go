package slack

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gollem-dev/gollem"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/utils/toolset"
	slackSDK "github.com/slack-go/slack"
)

type internalTool struct {
	slackClient interfaces.SlackClient
	maxLimit    int // Maximum number of results allowed from search

	tools gollem.ToolSet
}

// newInternalTool creates an internalTool and builds its type-safe tool set.
func newInternalTool(slackClient interfaces.SlackClient, maxLimit int) *internalTool {
	t := &internalTool{
		slackClient: slackClient,
		maxLimit:    maxLimit,
	}

	t.tools = toolset.New(
		gollem.MustNewTool("slack_search_messages", descSearchMessages, t.searchMessages),
		gollem.MustNewTool("slack_get_thread_messages", descGetThreadMessages, t.getThreadMessages),
		gollem.MustNewTool("slack_get_context_messages", descGetContextMessages, t.getContextMessages),
	)

	return t
}

// parseSlackTimestamp parses a Slack timestamp string (e.g., "1234567890.123456")
// and returns a time.Time with sub-second precision preserved.
func parseSlackTimestamp(tsStr string) time.Time {
	parts := strings.Split(tsStr, ".")
	if len(parts) == 0 {
		return time.Time{}
	}

	// Parse seconds
	sec, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return time.Time{}
	}

	// Parse nanoseconds from fractional part
	var nsec int64
	if len(parts) > 1 {
		// Pad or truncate to 9 digits (nanoseconds)
		fracStr := parts[1]
		if len(fracStr) > 9 {
			fracStr = fracStr[:9]
		} else {
			// Pad with zeros to get 9 digits
			fracStr = fracStr + strings.Repeat("0", 9-len(fracStr))
		}
		nsec, err = strconv.ParseInt(fracStr, 10, 64)
		if err != nil {
			// If fractional part is invalid, just use seconds
			nsec = 0
		}
	}

	return time.Unix(sec, nsec)
}

// Tool descriptions. Kept as constants so the typed-tool registration stays
// readable and the wire-level descriptions are unchanged.
const (
	descSearchMessages     = "Search for messages in Slack workspace using the search.messages API"
	descGetThreadMessages  = "Get all messages in a thread"
	descGetContextMessages = "Get messages before and after a specific message timestamp"
)

// Typed inputs for each tool. Counts use float64 so the inferred JSON schema
// stays "number", matching the wire-level type given before this migration.
type searchMessagesInput struct {
	Query     string  `json:"query" required:"true" description:"The search query (e.g., 'from:@user', 'in:general', 'has:link')"`
	Sort      string  `json:"sort" description:"Sort order: 'score' (relevance) or 'timestamp' (newest first)"`
	SortDir   string  `json:"sort_dir" description:"Sort direction: 'asc' or 'desc'"`
	Count     float64 `json:"count" description:"Number of results to return (default: 20, max: 100)"`
	Page      float64 `json:"page" description:"Page number for pagination (default: 1)"`
	Highlight bool    `json:"highlight" description:"Enable highlighting of search terms in results"`
}

type getThreadMessagesInput struct {
	Channel  string  `json:"channel" required:"true" description:"Channel ID"`
	ThreadTS string  `json:"thread_ts" required:"true" description:"Thread timestamp (ts of the parent message)"`
	Limit    float64 `json:"limit" description:"Maximum number of messages to return (default: 50, max: 200)"`
}

type getContextMessagesInput struct {
	Channel  string `json:"channel" required:"true" description:"Channel ID"`
	AroundTS string `json:"around_ts" required:"true" description:"Timestamp of the message to get context around"`
	// Before/After are pointers so an omitted argument (nil → default 10) can be
	// distinguished from an explicit 0 ("fetch nothing on this side"), which a
	// plain value field would collapse into the default. The pointer is unwrapped
	// to float64 for schema inference, so the wire type stays "number".
	Before *float64 `json:"before" description:"Number of messages before the timestamp (default: 10)"`
	After  *float64 `json:"after" description:"Number of messages after the timestamp (default: 10)"`
}

// Startup assertions: validate each tool's In/Out types form a valid schema at
// package init, so a malformed type fails immediately rather than at first use.
var (
	_ = gollem.MustToolSchema[searchMessagesInput, map[string]any]()
	_ = gollem.MustToolSchema[getThreadMessagesInput, map[string]any]()
	_ = gollem.MustToolSchema[getContextMessagesInput, map[string]any]()
)

func (t *internalTool) Specs(ctx context.Context) ([]gollem.ToolSpec, error) {
	return t.tools.Specs(ctx)
}

func (t *internalTool) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	return t.tools.Run(ctx, name, args)
}

func (t *internalTool) searchMessages(ctx context.Context, in searchMessagesInput) (map[string]any, error) {
	query := in.Query
	if query == "" {
		return nil, goerr.New("query is required")
	}

	params := slackSDK.SearchParameters{
		Count: 20,
		Page:  1,
	}

	params.Sort = in.Sort
	params.SortDirection = in.SortDir
	if in.Count > 0 {
		params.Count = int(in.Count)
	}
	if in.Page > 0 {
		params.Page = int(in.Page)
	}
	params.Highlight = in.Highlight

	resp, err := t.slackClient.SearchMessagesContext(ctx, query, params)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to search messages")
	}

	messages := make([]any, 0, len(resp.Matches))
	for i, msg := range resp.Matches {
		// Enforce maxLimit from parent agent
		if t.maxLimit > 0 && i >= t.maxLimit {
			break
		}

		formattedTime := parseSlackTimestamp(msg.Timestamp)

		// Thread timestamp - if this message is in a thread, use msg.Timestamp as thread_ts
		// Note: Slack's SearchMessage doesn't directly expose thread_ts, but for thread replies,
		// the timestamp can be used with slack_get_thread_messages
		threadTS := "" // Empty if not a thread message
		// If the message has Previous context, it might be a thread reply
		// For simplicity, we include timestamp which can be used for thread operations
		if msg.Previous.Text != "" || msg.Previous2.Text != "" {
			threadTS = msg.Timestamp
		}

		item := map[string]any{
			"channel_id":     msg.Channel.ID,
			"channel_name":   msg.Channel.Name,
			"user_name":      msg.Username,
			"text":           msg.Text,
			"timestamp":      msg.Timestamp,
			"thread_ts":      threadTS,
			"formatted_time": formattedTime.Format(time.RFC3339),
		}

		messages = append(messages, item)
	}

	return map[string]any{
		"total":    float64(resp.Total),
		"messages": messages,
	}, nil
}

func (t *internalTool) getThreadMessages(ctx context.Context, in getThreadMessagesInput) (map[string]any, error) {
	channel := in.Channel
	if channel == "" {
		return nil, goerr.New("channel is required")
	}

	threadTS := in.ThreadTS
	if threadTS == "" {
		return nil, goerr.New("thread_ts is required")
	}

	limit := 50
	if in.Limit > 0 {
		limit = int(in.Limit)
	}
	if limit > 200 {
		limit = 200
	}

	params := &slackSDK.GetConversationRepliesParameters{
		ChannelID: channel,
		Timestamp: threadTS,
		Limit:     limit,
	}

	msgs, _, _, err := t.slackClient.GetConversationRepliesContext(ctx, params)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get thread messages")
	}

	messages := make([]any, 0, len(msgs))
	for _, msg := range msgs {
		formattedTime := parseSlackTimestamp(msg.Timestamp)

		messages = append(messages, map[string]any{
			"user_name":      msg.Username,
			"text":           msg.Text,
			"timestamp":      msg.Timestamp,
			"formatted_time": formattedTime.Format(time.RFC3339),
		})
	}

	return map[string]any{
		"messages": messages,
	}, nil
}

func (t *internalTool) getContextMessages(ctx context.Context, in getContextMessagesInput) (map[string]any, error) {
	channel := in.Channel
	if channel == "" {
		return nil, goerr.New("channel is required")
	}

	aroundTS := in.AroundTS
	if aroundTS == "" {
		return nil, goerr.New("around_ts is required")
	}

	// nil (argument omitted) falls back to the documented default of 10; an
	// explicit value — including 0, meaning "fetch nothing on this side" — is
	// honored as-is.
	before := 10
	if in.Before != nil {
		before = int(*in.Before)
	}

	after := 10
	if in.After != nil {
		after = int(*in.After)
	}

	// Parse the timestamp
	ts, err := strconv.ParseFloat(aroundTS, 64)
	if err != nil {
		return nil, goerr.Wrap(err, "invalid timestamp format")
	}

	// Get messages before the timestamp
	var beforeMessages []slackSDK.Message
	if before > 0 {
		beforeParams := &slackSDK.GetConversationHistoryParameters{
			ChannelID: channel,
			Latest:    fmt.Sprintf("%.6f", ts),
			Inclusive: false,
			Limit:     before,
		}
		beforeResp, err := t.slackClient.GetConversationHistoryContext(ctx, beforeParams)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to get messages before timestamp")
		}
		beforeMessages = beforeResp.Messages
	}

	// Get messages after the timestamp
	var afterMessages []slackSDK.Message
	if after > 0 {
		afterParams := &slackSDK.GetConversationHistoryParameters{
			ChannelID: channel,
			Oldest:    fmt.Sprintf("%.6f", ts),
			Inclusive: false,
			Limit:     after,
		}
		afterResp, err := t.slackClient.GetConversationHistoryContext(ctx, afterParams)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to get messages after timestamp")
		}
		afterMessages = afterResp.Messages
	}

	formatMessages := func(msgs []slackSDK.Message) []any {
		result := make([]any, 0, len(msgs))
		for _, msg := range msgs {
			formattedTime := parseSlackTimestamp(msg.Timestamp)

			result = append(result, map[string]any{
				"user_name":      msg.Username,
				"text":           msg.Text,
				"timestamp":      msg.Timestamp,
				"formatted_time": formattedTime.Format(time.RFC3339),
			})
		}
		return result
	}

	return map[string]any{
		"before_messages": formatMessages(beforeMessages),
		"after_messages":  formatMessages(afterMessages),
	}, nil
}
