package slackctx_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/utils/slackctx"
)

func TestSlackURL(t *testing.T) {
	ctx := context.Background()

	t.Run("WithSlackURL and SlackURL", func(t *testing.T) {
		url := "https://slack.com/archives/C1234567890/p1234567890123456"
		ctx := slackctx.WithSlackURL(ctx, url)
		got := slackctx.SlackURL(ctx)
		gt.Equal(t, got, url)
	})

	t.Run("SlackURL returns empty string when not set", func(t *testing.T) {
		got := slackctx.SlackURL(ctx)
		gt.Equal(t, got, "")
	})

	t.Run("SlackURL returns empty string when wrong type", func(t *testing.T) {
		// This shouldn't happen in practice, but test defensive behavior
		ctx := context.WithValue(ctx, "slack_url", 123) // wrong type
		got := slackctx.SlackURL(ctx)
		gt.Equal(t, got, "")
	})
}
