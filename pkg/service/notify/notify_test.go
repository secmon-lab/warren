package notify_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	model "github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/service/notify"

	"github.com/slack-go/slack"
)

func newClient(t *testing.T) (*slack.Client, string) {
	apiToken, ok := os.LookupEnv("TEST_SLACK_OAUTH_TOKEN")
	if !ok {
		t.Skip("TEST_SLACK_OAUTH_TOKEN is required")
	}

	testCh, ok := os.LookupEnv("TEST_SLACK_CHANNEL_ID")
	if !ok {
		t.Skip("TEST_SLACK_CHANNEL_ID is required")
	}

	client := slack.New(apiToken)

	return client, testCh
}

func TestNotify(t *testing.T) {
	client, testCh := newClient(t)
	chID, ts, err := client.PostMessageContext(t.Context(), testCh, slack.MsgOptionText("notify thread test", false))
	gt.NoError(t, err)

	thread := model.Thread{
		ChannelID: chID,
		ThreadID:  ts,
	}

	notifier := notify.NewSlackThread(client, thread)

	testMessages := []string{
		"😂 test1",
		"😁 test2",
		"😊 test3",
	}

	for _, msg := range testMessages {
		notifier.Notify(t.Context(), msg)
	}
}

func TestNotifyContext(t *testing.T) {
	client, testCh := newClient(t)
	chID, ts, err := client.PostMessageContext(t.Context(), testCh, slack.MsgOptionText("notify context test", false))
	gt.NoError(t, err)

	thread := model.Thread{
		ChannelID: chID,
		ThreadID:  ts,
	}

	notifier := notify.NewSlackThread(client, thread)

	msgCtx := notifier.NewMessageContext(t.Context(), "test")

	testMessages := []string{
		"😂 test1",
		"😁 test2",
		"😊 test3",
	}

	for _, msg := range testMessages {
		msgCtx.Append(t.Context(), msg)
	}
}

func TestNotifyNewPostWithManyMessages(t *testing.T) {
	msgID := fmt.Sprintf("%d.%d", time.Now().Unix(), time.Now().Nanosecond())
	mockClient := &slackClientMock{
		PostMessageContextFunc: func(ctx context.Context, channelID string, options ...slack.MsgOption) (string, string, error) {
			gt.Value(t, channelID).Equal("test-channel")
			return "", msgID, nil
		},
		UpdateMessageContextFunc: func(ctx context.Context, channelID, messageID string, options ...slack.MsgOption) (string, string, string, error) {
			gt.Value(t, channelID).Equal("test-channel")
			gt.Value(t, messageID).Equal(msgID)
			return "", "", "", nil
		},
	}

	notifier := notify.NewSlackThread(mockClient, model.Thread{
		ChannelID: "test-channel",
		ThreadID:  "test-thread",
	})

	loopCount := 10
	for range loopCount {
		notifier.Notify(t.Context(), "Take me back to the green love")
	}

	gt.Array(t, mockClient.PostMessageContextCalls()).Length(10)
	gt.Array(t, mockClient.UpdateMessageContextCalls()).Length(0)
}
