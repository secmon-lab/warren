package service_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/prompt"
	"github.com/secmon-lab/warren/pkg/service"
	"github.com/secmon-lab/warren/pkg/utils/test"
)

func newSlackService(t *testing.T) *service.Slack {
	envs := test.NewEnvVars(t, "TEST_SLACK_CHANNEL_ID", "TEST_SLACK_OAUTH_TOKEN", "TEST_SLACK_SIGNING_SECRET")
	client, err := service.NewSlack(envs.Get("TEST_SLACK_OAUTH_TOKEN"), envs.Get("TEST_SLACK_SIGNING_SECRET"), envs.Get("TEST_SLACK_CHANNEL_ID"))
	gt.NoError(t, err).Must()
	return client
}

func TestSlackPostAlert(t *testing.T) {
	svc := newSlackService(t)

	_, err := svc.PostAlert(context.Background(), model.Alert{
		ID:          "1234567890",
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
		Conclusion: model.AlertConclusionFalsePositive,
		Comment:    "Test Comment",
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
	gt.NoError(t, err).Must()
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

func genDummyAlertWithSlackThread() model.Alert {
	alert := genDummyAlert()
	alert.SlackThread = &model.SlackThread{
		ChannelID: "C0123456789",
		ThreadID:  fmt.Sprintf("%d", time.Now().Unix()),
	}
	return alert
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

	newThread := svc.NewThread(*alert.SlackThread)
	gt.NoError(t, newThread.AttachFile(context.Background(), "test", "test.txt", []byte("test")))
}

func TestTrimMention(t *testing.T) {
	svc := service.NewDummySlackService("U0123456789")

	gt.Equal(t, svc.TrimMention("<@U0123456789> test hoge"), "test hoge")
	gt.Equal(t, svc.TrimMention("test"), "test")
	gt.Equal(t, svc.TrimMention("test <@U0123456789>"), "")
	gt.Equal(t, svc.TrimMention("test <@U0123456789> <@U0123456789> blue"), "blue")
	gt.Equal(t, svc.TrimMention("<@NOT_EXIST> test"), "<@NOT_EXIST> test")
}

func TestPostAlertGroups(t *testing.T) {
	svc := newSlackService(t)

	groups := []model.AlertGroup{
		{
			AlertGroupMetadata: model.AlertGroupMetadata{
				Title:       "Group 1",
				Description: "Group 1 Description. This is a test group.",
			},
			Alerts: []model.Alert{
				genDummyAlertWithSlackThread(),
				genDummyAlertWithSlackThread(),
			},
		},
		{
			AlertGroupMetadata: model.AlertGroupMetadata{
				Title:       "Group 2",
				Description: "Group 2 Description. This is another test group.",
			},
			Alerts: []model.Alert{
				genDummyAlertWithSlackThread(),
				genDummyAlertWithSlackThread(),
			},
		},
	}

	thread, err := svc.PostMessage(context.Background(), "group test")
	gt.NoError(t, err)

	gt.NoError(t, thread.PostAlertGroups(context.Background(), groups))

	link := fmt.Sprintf("slack://channel?id=%s&message=%s", thread.ChannelID(), thread.ThreadID())
	svc.PostMessage(context.Background(), "test link: "+link)
}
