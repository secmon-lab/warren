package slackctx

import "context"

type ctxKey string

const (
	slackURLKey ctxKey = "slack_url"
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
