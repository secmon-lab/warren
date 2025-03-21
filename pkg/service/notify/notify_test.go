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

func TestNotify_with_Slack(t *testing.T) {
	apiToken, ok := os.LookupEnv("TEST_SLACK_OAUTH_TOKEN")
	if !ok {
		t.Skip("TEST_SLACK_OAUTH_TOKEN is required")
	}
	testCh, ok := os.LookupEnv("TEST_SLACK_CHANNEL")
	if !ok {
		t.Skip("TEST_SLACK_CHANNEL is required")
	}

	client := slack.New(apiToken)

	chID, ts, err := client.PostMessage(testCh, slack.MsgOptionText("notify test", false))
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

	loopCount := 400
	for range loopCount {
		notifier.Notify(t.Context(), "Take me back to the green love")
	}

	// PostMessage need to be called multiple times because the message will be overflowed max block size.
	gt.Array(t, mockClient.PostMessageContextCalls()).Longer(1)
	postCallCount := len(mockClient.PostMessageContextCalls())
	// UpdateMessage need to be called multiple times because to show the new message.
	gt.Array(t, mockClient.UpdateMessageContextCalls()).Length(loopCount - postCallCount)
}
