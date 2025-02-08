package service_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/prompt"
	"github.com/secmon-lab/warren/pkg/service"
	"github.com/secmon-lab/warren/pkg/utils/test"
)

func newSlackService(t *testing.T) *service.Slack {
	envs := test.NewEnvVars(t, "TEST_SLACK_CHANNEL_ID", "TEST_SLACK_OAUTH_TOKEN", "TEST_SLACK_SIGNING_SECRET")
	return service.NewSlack(envs.Get("TEST_SLACK_OAUTH_TOKEN"), envs.Get("TEST_SLACK_SIGNING_SECRET"), envs.Get("TEST_SLACK_CHANNEL_ID"))
}

func TestSlack(t *testing.T) {
	svc := newSlackService(t)

	_, err := svc.PostAlert(context.Background(), model.Alert{
		Title:  "Test Alert Title",
		Schema: "test.alert.v1",
		Attributes: []model.Attribute{
			{
				Key:   "severity",
				Value: "high",
			},
			{
				Key:   "source",
				Value: "test",
			},
			{
				Key:   "details",
				Value: "Click here",
				Link:  "https://example.com/alert/details",
			},
		},
		Data: map[string]interface{}{
			"foo": "bar",
			"baz": 123,
		},
	})
	gt.NoError(t, err)
}

func TestSlackUpdateAlert(t *testing.T) {
	svc := newSlackService(t)

	alert := model.Alert{
		Title:  "Test Alert Title",
		Schema: "test.alert.v1",
	}

	thread, err := svc.PostAlert(context.Background(), alert)
	gt.NoError(t, err)
	alert.SlackChannel = thread.ChannelID()
	alert.SlackMessageID = thread.ThreadID()

	alert.Title = "Updated Alert Title"
	alert.Attributes = []model.Attribute{
		{
			Key:   "severity",
			Value: "low",
		},
	}

	gt.NoError(t, thread.UpdateAlert(context.Background(), alert))
}

func TestSlackPostThreadMessages(t *testing.T) {
	svc := newSlackService(t)

	alert := model.Alert{
		Title:  "Test Alert Title",
		Schema: "test.alert.v1",
	}

	thread, err := svc.PostAlert(context.Background(), alert)
	gt.NoError(t, err)
	alert.SlackChannel = thread.ChannelID()
	alert.SlackMessageID = thread.ThreadID()

	gt.NoError(t, thread.PostNextAction(context.Background(), prompt.ActionPromptResult{
		Action: "test",
		Args: map[string]any{
			"foo": "bar",
			"baz": "qux",
		},
	}))

	gt.NoError(t, thread.AttachFile(context.Background(),
		"this is test data",
		"test.csv",
		[]byte("hoge,mage,fuga\nred,blue,green"),
	))
}
