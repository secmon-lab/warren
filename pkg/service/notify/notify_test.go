package notify_test

import (
	"os"
	"testing"

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
