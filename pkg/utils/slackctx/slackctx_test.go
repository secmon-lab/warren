package slackctx_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/utils/slackctx"
)

func TestSlackURL(t *testing.T) {
	t.Run("WithSlackURL and SlackURL", func(t *testing.T) {
		ctx := t.Context()
		url := "https://slack.com/archives/C1234567890/p1234567890123456"
		ctx = slackctx.WithSlackURL(ctx, url)
		got := slackctx.SlackURL(ctx)
		gt.Equal(t, got, url)
	})

	t.Run("SlackURL returns empty string when not set", func(t *testing.T) {
		ctx := t.Context()
		got := slackctx.SlackURL(ctx)
		gt.Equal(t, got, "")
	})
}
