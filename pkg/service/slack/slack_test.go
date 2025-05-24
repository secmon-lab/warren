package slack_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	model "github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
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
		ID: "1234567890",
		Metadata: alert.Metadata{
			Title:       "Test Alert Title",
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
		},
		TicketID: types.NewTicketID(),
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

	gt.NoError(t, thread.UpdateAlert(context.Background(), dummy))
}

func TestSlackUpdateTicket(t *testing.T) {
	svc := newSlackService(t)
	ctx := t.Context()
	dummy := genDummyAlert()

	thread, err := svc.PostAlert(context.Background(), dummy)
	gt.NoError(t, err).Required()
	dummy.SlackThread = &model.Thread{
		ChannelID: thread.ChannelID(),
		ThreadID:  thread.ThreadID(),
	}

	ticketData := ticket.New(context.Background(), []types.AlertID{dummy.ID}, &model.Thread{
		ChannelID: thread.ChannelID(),
		ThreadID:  thread.ThreadID(),
	})
	ticketData.Metadata.Title = "Test Ticket Title"
	ticketData.Metadata.Description = "Test Ticket Description"
	ticketData.Metadata.Summary = "Test Ticket Summary"
	ticketData.Status = types.TicketStatusInvestigating
	ticketData.Reason = "Test Ticket Reason"

	ts, err := thread.PostTicket(ctx, ticketData, alert.Alerts{&dummy})
	gt.NoError(t, err)
	ticketData.SlackMessageID = ts
	ticketData.Reason = "Updated reason"

	_, err = thread.PostTicket(ctx, ticketData, alert.Alerts{&dummy})
	gt.NoError(t, err)
}

func genDummyAlert() alert.Alert {
	return alert.New(context.Background(), "test.alert.v1", map[string]any{
		"foo": "bar",
		"baz": 123,
	}, alert.Metadata{
		Title: "Test Alert Title",
		Attributes: []alert.Attribute{
			{
				Key:   "color",
				Value: "red",
			},
		},
	})
}

func genDummyAlertWithSlackThread() *alert.Alert {
	alert := genDummyAlert()
	alert.SlackThread = &model.Thread{
		ChannelID: "C0123456789",
		ThreadID:  fmt.Sprintf("%d", time.Now().Unix()),
	}
	return &alert
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

func TestPostAlerts(t *testing.T) {
	svc := newSlackService(t)

	alerts := alert.Alerts{
		genDummyAlertWithSlackThread(),
		genDummyAlertWithSlackThread(),
		genDummyAlertWithSlackThread(),
		genDummyAlertWithSlackThread(),
	}
	alerts[1].CreatedAt = alerts[0].CreatedAt.Add(time.Second)
	alerts[2].CreatedAt = alerts[0].CreatedAt.Add(time.Second * 2)

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
	}, alert.Alerts{
		genDummyAlertWithSlackThread(),
		genDummyAlertWithSlackThread(),
		genDummyAlertWithSlackThread(),
		genDummyAlertWithSlackThread(),
	})
	alertList.Title = "Test Alert List"
	alertList.Description = "This is a test alert list"

	thread, err := svc.PostMessage(context.Background(), "alert list test")
	gt.NoError(t, err)
	gt.NoError(t, thread.PostAlertList(context.Background(), alertList))
}

func TestPostTicketList(t *testing.T) {
	svc := newSlackService(t)
	ctx := t.Context()

	// Create test tickets
	tickets := []*ticket.Ticket{
		{
			ID: types.NewTicketID(),
			Metadata: ticket.Metadata{
				Title:       "Test Ticket 1",
				Description: "Description for ticket 1",
			},
			Status: types.TicketStatusInvestigating,
			Assignee: &model.User{
				ID:   "U0123456789",
				Name: "John Doe",
			},
			SlackThread: &model.Thread{
				ChannelID: "C0123456789",
				ThreadID:  "T0123456789",
			},
			CreatedAt: time.Now().Add(-time.Hour),
		},
		{
			ID: types.NewTicketID(),
			Metadata: ticket.Metadata{
				Title:       "Test Ticket 2",
				Description: "Description for ticket 2",
			},
			Status: types.TicketStatusResolved,
			SlackThread: &model.Thread{
				ChannelID: "C0123456789",
				ThreadID:  "T0123456789",
			},
			CreatedAt: time.Now().Add(-time.Hour * 2),
		},
	}

	// Post ticket list
	thread, err := svc.PostMessage(ctx, "Ticket list test")
	gt.NoError(t, err)
	gt.NoError(t, thread.PostTicketList(ctx, tickets))
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
