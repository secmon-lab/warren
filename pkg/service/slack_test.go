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

func TestSlackPostAlert(t *testing.T) {
	svc := newSlackService(t)

	_, err := svc.PostAlert(context.Background(), model.Alert{
		Title:       "Test Alert Title",
		Schema:      "test.alert.v1",
		Description: "Test Alert Description",
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
		Assignee: &model.SlackUser{
			ID:   "U0123456789",
			Name: "John Doe",
		},
		Status: model.AlertStatusAcknowledged,
		Data: map[string]interface{}{
			"foo": "bar",
			"baz": 123,
		},
		Finding: &model.AlertFinding{
			Severity:       model.AlertSeverityHigh,
			Summary:        "Test Summary",
			Reason:         "Test Reason",
			Recommendation: "Test Recommendation",
		},
	})
	gt.NoError(t, err)
}

func TestSlackUpdateAlert(t *testing.T) {
	svc := newSlackService(t)

	alert := genDummyAlert()

	thread, err := svc.PostAlert(context.Background(), alert)
	gt.NoError(t, err)
	alert.SlackThread = &model.SlackThread{
		ChannelID: thread.ChannelID(),
		ThreadID:  thread.ThreadID(),
	}

	alert.Title = "Updated Alert Title"
	alert.Finding = &model.AlertFinding{
		Severity:       model.AlertSeverityLow,
		Summary:        "Updated Summary",
		Reason:         "Updated Reason",
		Recommendation: "Updated Recommendation",
	}

	gt.NoError(t, thread.UpdateAlert(context.Background(), alert))
}

func genDummyAlert() model.Alert {
	return model.NewAlert(context.Background(), "test.alert.v1", model.PolicyAlert{
		Title: "Test Alert Title",
		Attrs: []model.Attribute{
			{
				Key:   "color",
				Value: "red",
			},
		},
		Data: map[string]any{
			"foo": "bar",
			"baz": 123,
		},
	})
}

func TestSlackPostThreadMessages(t *testing.T) {
	svc := newSlackService(t)

	alert := genDummyAlert()

	thread, err := svc.PostAlert(context.Background(), alert)
	gt.NoError(t, err)
	alert.SlackThread = &model.SlackThread{
		ChannelID: thread.ChannelID(),
		ThreadID:  thread.ThreadID(),
	}

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

func TestSlackPostConclusion(t *testing.T) {
	svc := newSlackService(t)

	alert := genDummyAlert()

	thread, err := svc.PostAlert(context.Background(), alert)
	gt.NoError(t, err)
	alert.SlackThread = &model.SlackThread{
		ChannelID: thread.ChannelID(),
		ThreadID:  thread.ThreadID(),
	}

	gt.NoError(t, thread.PostFinding(context.Background(), model.AlertFinding{
		Severity:       model.AlertSeverityHigh,
		Summary:        "Test Summary",
		Reason:         "Test Reason",
		Recommendation: "Test Recommendation",
	}))
}

func TestAttachFile(t *testing.T) {
	svc := newSlackService(t)

	alert := genDummyAlert()

	thread, err := svc.PostAlert(context.Background(), alert)
	gt.NoError(t, err)
	alert.SlackThread = &model.SlackThread{
		ChannelID: thread.ChannelID(),
		ThreadID:  thread.ThreadID(),
	}

	newThread := svc.NewThread(alert)
	gt.NoError(t, newThread.AttachFile(context.Background(), "test", "test.txt", []byte("test")))
}
