package slack_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/policy"
	model "github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/slack"
	"github.com/secmon-lab/warren/pkg/utils/test"

	slack_sdk "github.com/slack-go/slack"
)

func newSlackService(t *testing.T) *slack.Service {
	envs := test.NewEnvVars(t, "TEST_SLACK_CHANNEL_ID", "TEST_SLACK_OAUTH_TOKEN")
	client := slack_sdk.New(envs.Get("TEST_SLACK_OAUTH_TOKEN"))

	svc, err := slack.New(client, envs.Get("TEST_SLACK_CHANNEL_ID"))
	gt.NoError(t, err).Required()

	return svc
}

func TestSlackPostAlert(t *testing.T) {
	svc := newSlackService(t)

	_, err := svc.PostAlert(context.Background(), alert.Alert{
		ID:          "1234567890",
		Title:       "Test Alert Title",
		Schema:      "test.alert.v1",
		Description: "Test Alert Description",
		Attributes: []alert.Attribute{
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
		Assignee: &model.User{
			ID:   "U0123456789",
			Name: "John Doe",
		},
		Status: types.AlertStatusAcknowledged,
		Data: map[string]interface{}{
			"foo": "bar",
			"baz": 123,
		},
		Conclusion: types.AlertConclusionFalsePositive,
		Reason:     "Test Reason",
		Finding: &alert.Finding{
			Severity:       types.AlertSeverityHigh,
			Summary:        "Test Summary",
			Reason:         "Test Reason",
			Recommendation: "Test Recommendation",
		},
	})
	gt.NoError(t, err)
}

func TestSlackUpdateAlert(t *testing.T) {
	svc := newSlackService(t)

	dummy := genDummyAlert()

	thread, err := svc.PostAlert(context.Background(), dummy)
	gt.NoError(t, err).Required()
	dummy.SlackThread = &model.Thread{
		ChannelID: thread.ChannelID(),
		ThreadID:  thread.ThreadID(),
	}

	dummy.Title = "Updated Alert Title"
	dummy.Finding = &alert.Finding{
		Severity:       types.AlertSeverityLow,
		Summary:        "Updated Summary",
		Reason:         "Updated Reason",
		Recommendation: "Updated Recommendation",
	}

	gt.NoError(t, thread.UpdateAlert(context.Background(), dummy))
}

func genDummyAlert() alert.Alert {
	return alert.New(context.Background(), "test.alert.v1", alert.Metadata{
		Title: "Test Alert Title",
		Attrs: []alert.Attribute{
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

func genDummyAlertWithSlackThread() alert.Alert {
	alert := genDummyAlert()
	alert.SlackThread = &model.Thread{
		ChannelID: "C0123456789",
		ThreadID:  fmt.Sprintf("%d", time.Now().Unix()),
	}
	return alert
}

func TestSlackPostConclusion(t *testing.T) {
	svc := newSlackService(t)

	dummy := genDummyAlert()

	thread, err := svc.PostAlert(context.Background(), dummy)
	gt.NoError(t, err)
	dummy.SlackThread = &model.Thread{
		ChannelID: thread.ChannelID(),
		ThreadID:  thread.ThreadID(),
	}

	gt.NoError(t, thread.PostFinding(context.Background(), alert.Finding{
		Severity:       types.AlertSeverityHigh,
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
	alert.SlackThread = &model.Thread{
		ChannelID: thread.ChannelID(),
		ThreadID:  thread.ThreadID(),
	}

	newThread := svc.NewThread(*alert.SlackThread)
	gt.NoError(t, newThread.AttachFile(context.Background(), "test", "test.txt", []byte("test")))
}

func TestIsBotUser(t *testing.T) {
	svc := newSlackService(t)

	botID := svc.BotUserID()
	gt.S(t, botID).Match(`^[A-Z][A-Z0-9]{6,12}$`)
}

func TestPostPolicyDiff(t *testing.T) {
	svc := newSlackService(t)

	diff := policy.NewDiff(context.Background(), "Policy Diff", "This is a test policy diff", map[string]string{
		"test.rego": `package test

allow if {
  input.color == "red"
}
`,
	},
		map[string]string{
			"test.rego": `package test

allow if {
  input.color == "blue"
}
`,
		},
		policy.NewTestDataSet(),
	)

	thread, err := svc.PostMessage(context.Background(), "policy diff test")
	gt.NoError(t, err)
	gt.NoError(t, thread.PostPolicyDiff(context.Background(), diff))
}

func TestPostAlerts(t *testing.T) {
	svc := newSlackService(t)

	alerts := []alert.Alert{
		genDummyAlertWithSlackThread(),
		genDummyAlertWithSlackThread(),
		genDummyAlertWithSlackThread(),
		genDummyAlertWithSlackThread(),
	}
	alerts[1].ParentID = alerts[0].ID
	alerts[1].CreatedAt = alerts[0].CreatedAt.Add(time.Second)
	alerts[1].Status = types.AlertStatusAcknowledged
	alerts[2].ParentID = alerts[0].ID
	alerts[2].CreatedAt = alerts[0].CreatedAt.Add(time.Second * 2)
	alerts[3].Assignee = &model.User{
		ID:   "U0123456789",
		Name: "John Doex",
	}
	alerts[3].Status = types.AlertStatusResolved

	thread, err := svc.PostMessage(context.Background(), "alerts test")
	gt.NoError(t, err)
	gt.NoError(t, thread.PostAlerts(context.Background(), alerts))
}

func TestPostAlertList(t *testing.T) {
	svc := newSlackService(t)

	alertList := alert.NewList(context.Background(), model.Thread{
		ChannelID: "C0123456789",
		ThreadID:  "T0123456789",
	}, &model.User{
		ID:   "U0123456789",
		Name: "John Doe",
	}, []alert.Alert{
		genDummyAlertWithSlackThread(),
		genDummyAlertWithSlackThread(),
		genDummyAlertWithSlackThread(),
		genDummyAlertWithSlackThread(),
	})
	alertList.Title = "Test Alert List"
	alertList.Description = "This is a test alert list"

	thread, err := svc.PostMessage(context.Background(), "alert list test")
	gt.NoError(t, err)
	gt.NoError(t, thread.PostAlertList(context.Background(), &alertList))
}

func TestNewStateFunc(t *testing.T) {
	svc := newSlackService(t)

	cases := []struct {
		name     string
		base     string
		messages []string
		want     int
	}{
		{
			name:     "empty base and messages",
			base:     "",
			messages: []string{},
			want:     0,
		},
		{
			name:     "only base message",
			base:     "base message",
			messages: []string{},
			want:     1,
		},
		{
			name: "only state messages",
			base: "",
			messages: []string{
				"message 1",
				"message 2",
			},
			want: 1,
		},
		{
			name: "base and state messages",
			base: "base message",
			messages: []string{
				"message 1",
				"message 2",
			},
			want: 2,
		},
		{
			name: "state messages with markdown",
			base: "base message",
			messages: []string{
				"*message 1*",
				"_message 2_",
				"`message 3`",
				"```message 4\nmessage 4\n```",
			},
			want: 2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			thread, err := svc.PostMessage(ctx, "State test: "+tc.name)
			gt.NoError(t, err)

			fn := thread.NewStateFunc(ctx, tc.base)
			for _, msg := range tc.messages {
				fn(ctx, msg)
			}
		})
	}
}
