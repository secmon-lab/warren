package service_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/service"
	"github.com/secmon-lab/warren/pkg/utils/test"
)

func TestSlack(t *testing.T) {
	envs := test.NewEnvVars(t, "TEST_SLACK_CHANNEL_ID", "TEST_SLACK_OAUTH_TOKEN", "TEST_SLACK_SIGNING_SECRET")

	svc := service.NewSlack(envs.Get("TEST_SLACK_OAUTH_TOKEN"), envs.Get("TEST_SLACK_SIGNING_SECRET"), envs.Get("TEST_SLACK_CHANNEL_ID"))

	_, _, err := svc.PostAlert(context.Background(), model.Alert{
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
	envs := test.NewEnvVars(t, "TEST_SLACK_CHANNEL_ID", "TEST_SLACK_OAUTH_TOKEN", "TEST_SLACK_SIGNING_SECRET")

	svc := service.NewSlack(envs.Get("TEST_SLACK_OAUTH_TOKEN"), envs.Get("TEST_SLACK_SIGNING_SECRET"), envs.Get("TEST_SLACK_CHANNEL_ID"))

	alert := model.Alert{
		Title:  "Test Alert Title",
		Schema: "test.alert.v1",
	}

	channelID, timestamp, err := svc.PostAlert(context.Background(), alert)
	gt.NoError(t, err)
	alert.SlackChannel = channelID
	alert.SlackMessageID = timestamp

	alert.Title = "Updated Alert Title"
	alert.Attributes = []model.Attribute{
		{
			Key:   "severity",
			Value: "low",
		},
	}

	gt.NoError(t, svc.UpdateAlert(context.Background(), alert))
}
