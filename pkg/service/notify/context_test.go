package notify_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/service/notify"
	slack_sdk "github.com/slack-go/slack"
)

func TestContext(t *testing.T) {
	ctx := context.Background()

	mockClient := &slackClientMock{
		PostMessageContextFunc: func(ctx context.Context, channelID string, options ...slack_sdk.MsgOption) (string, string, error) {
			return "C01234567890", "T01234567890", nil
		},
	}

	thread := notify.NewSlackThread(mockClient, slack.Thread{
		ChannelID: "C01234567890",
		ThreadID:  "T01234567890",
	})
	ctx = notify.WithThread(ctx, thread)
	notify.Notify(ctx, "test")
	gt.Array(t, mockClient.PostMessageContextCalls()).Length(1)
}

func TestContext_NoSend(t *testing.T) {
	ctx := context.Background()
	notify.Notify(ctx, "test")
	// no panic
}

func TestContext_NewMessageContext(t *testing.T) {
	ctx := context.Background()
	msgCtx := notify.NewMessageContext(ctx, "test")
	gt.NotNil(t, msgCtx)
	msgCtx.Append(ctx, "test2")
	// No panic
}
