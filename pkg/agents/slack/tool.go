package slack

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	slackSDK "github.com/slack-go/slack"
)

type internalTool struct {
	slackClient interfaces.SlackClient
	maxLimit    int // Maximum number of results allowed from search
}

func (t *internalTool) Specs(ctx context.Context) ([]gollem.ToolSpec, error) {
	return []gollem.ToolSpec{
		{
			Name:        "slack_search_messages",
			Description: "Search for messages in Slack workspace using the search.messages API",
			Parameters: map[string]*gollem.Parameter{
				"query": {
					Type:        gollem.TypeString,
					Description: "The search query (e.g., 'from:@user', 'in:general', 'has:link')",
				},
				"sort": {
					Type:        gollem.TypeString,
					Description: "Sort order: 'score' (relevance) or 'timestamp' (newest first)",
				},
				"sort_dir": {
					Type:        gollem.TypeString,
					Description: "Sort direction: 'asc' or 'desc'",
				},
				"count": {
					Type:        gollem.TypeNumber,
					Description: "Number of results to return (default: 20, max: 100)",
				},
				"page": {
					Type:        gollem.TypeNumber,
					Description: "Page number for pagination (default: 1)",
				},
				"highlight": {
					Type:        gollem.TypeBoolean,
					Description: "Enable highlighting of search terms in results",
				},
			},
			Required: []string{"query"},
		},
		{
			Name:        "slack_get_thread_messages",
			Description: "Get all messages in a thread",
			Parameters: map[string]*gollem.Parameter{
				"channel": {
					Type:        gollem.TypeString,
					Description: "Channel ID",
				},
				"thread_ts": {
					Type:        gollem.TypeString,
					Description: "Thread timestamp (ts of the parent message)",
				},
				"limit": {
					Type:        gollem.TypeNumber,
					Description: "Maximum number of messages to return (default: 50, max: 200)",
				},
			},
			Required: []string{"channel", "thread_ts"},
		},
		{
			Name:        "slack_get_context_messages",
			Description: "Get messages before and after a specific message timestamp",
			Parameters: map[string]*gollem.Parameter{
				"channel": {
					Type:        gollem.TypeString,
					Description: "Channel ID",
				},
				"around_ts": {
					Type:        gollem.TypeString,
					Description: "Timestamp of the message to get context around",
				},
				"before": {
					Type:        gollem.TypeNumber,
					Description: "Number of messages before the timestamp (default: 10)",
				},
				"after": {
					Type:        gollem.TypeNumber,
					Description: "Number of messages after the timestamp (default: 10)",
				},
			},
			Required: []string{"channel", "around_ts"},
		},
	}, nil
}

func (t *internalTool) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	switch name {
	case "slack_search_messages":
		return t.searchMessages(ctx, args)
	case "slack_get_thread_messages":
		return t.getThreadMessages(ctx, args)
	case "slack_get_context_messages":
		return t.getContextMessages(ctx, args)
	default:
		return nil, goerr.New("unknown tool name", goerr.V("name", name))
	}
}

func (t *internalTool) searchMessages(ctx context.Context, args map[string]any) (map[string]any, error) {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return nil, goerr.New("query is required")
	}

	params := slackSDK.SearchParameters{
		Count: 20,
		Page:  1,
	}

	if sort, ok := args["sort"].(string); ok {
		params.Sort = sort
	}
	if sortDir, ok := args["sort_dir"].(string); ok {
		params.SortDirection = sortDir
	}
	if count, ok := args["count"].(float64); ok {
		params.Count = int(count)
	}
	if page, ok := args["page"].(float64); ok {
		params.Page = int(page)
	}
	if highlight, ok := args["highlight"].(bool); ok {
		params.Highlight = highlight
	}

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

		var formattedTime time.Time
		if ts, err := strconv.ParseFloat(msg.Timestamp, 64); err == nil {
			formattedTime = time.Unix(int64(ts), 0)
		}

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

func (t *internalTool) getThreadMessages(ctx context.Context, args map[string]any) (map[string]any, error) {
	channel, ok := args["channel"].(string)
	if !ok || channel == "" {
		return nil, goerr.New("channel is required")
	}

	threadTS, ok := args["thread_ts"].(string)
	if !ok || threadTS == "" {
		return nil, goerr.New("thread_ts is required")
	}

	limit := 50
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
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
		var formattedTime time.Time
		if ts, err := strconv.ParseFloat(msg.Timestamp, 64); err == nil {
			formattedTime = time.Unix(int64(ts), 0)
		}

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

func (t *internalTool) getContextMessages(ctx context.Context, args map[string]any) (map[string]any, error) {
	channel, ok := args["channel"].(string)
	if !ok || channel == "" {
		return nil, goerr.New("channel is required")
	}

	aroundTS, ok := args["around_ts"].(string)
	if !ok || aroundTS == "" {
		return nil, goerr.New("around_ts is required")
	}

	before := 10
	if b, ok := args["before"].(float64); ok {
		before = int(b)
	}

	after := 10
	if a, ok := args["after"].(float64); ok {
		after = int(a)
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
			var formattedTime time.Time
			if ts, err := strconv.ParseFloat(msg.Timestamp, 64); err == nil {
				formattedTime = time.Unix(int64(ts), 0)
			}

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
