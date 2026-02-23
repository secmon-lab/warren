package slackctx

import (
	"context"

	"github.com/secmon-lab/warren/pkg/domain/model/slack"
)

type ctxKey string

const (
	slackURLKey    ctxKey = "slack_url"
	slackThreadKey ctxKey = "slack_thread"
)

// WithSlackURL embeds Slack message URL into context
func WithSlackURL(ctx context.Context, url string) context.Context {
	return context.WithValue(ctx, slackURLKey, url)
}

// SlackURL extracts Slack message URL from context
func SlackURL(ctx context.Context) string {
	if url, ok := ctx.Value(slackURLKey).(string); ok {
		return url
	}
	return ""
}

// WithThread sets the current Slack thread in context
func WithThread(ctx context.Context, thread slack.Thread) context.Context {
	return context.WithValue(ctx, slackThreadKey, thread)
}

// ThreadFrom retrieves the current Slack thread from context, returns nil if not set
func ThreadFrom(ctx context.Context) *slack.Thread {
	if thread, ok := ctx.Value(slackThreadKey).(slack.Thread); ok {
		return &thread
	}
	return nil
}
