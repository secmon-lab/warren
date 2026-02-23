package slackctx_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
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

func TestThread(t *testing.T) {
	t.Run("WithThread and ThreadFrom round-trip", func(t *testing.T) {
		ctx := t.Context()
		thread := slack.Thread{
			TeamID:    "T12345",
			ChannelID: "C67890",
			ThreadID:  "1234567890.123456",
		}
		ctx = slackctx.WithThread(ctx, thread)
		got := slackctx.ThreadFrom(ctx)
		gt.V(t, got).NotNil()
		gt.Equal(t, got.TeamID, "T12345")
		gt.Equal(t, got.ChannelID, "C67890")
		gt.Equal(t, got.ThreadID, "1234567890.123456")
	})

	t.Run("ThreadFrom returns nil when not set", func(t *testing.T) {
		ctx := t.Context()
		got := slackctx.ThreadFrom(ctx)
		gt.V(t, got).Nil()
	})
}
