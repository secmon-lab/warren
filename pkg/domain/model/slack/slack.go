package slack

import (
	"context"

	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/slack-go/slack/slackevents"
)

type Thread struct {
	TeamID    string `json:"team_id"`
	ChannelID string `json:"channel_id"`
	ThreadID  string `json:"thread_id"`
}

type Message struct {
	id      string
	channel string

	// If empty, the message is the first message in the thread
	threadID string

	teamID string
}

func (x *Message) SlackThread() Thread {
	th := Thread{
		TeamID:    x.teamID,
		ChannelID: x.channel,
		ThreadID:  x.threadID,
	}
	if th.ThreadID == "" {
		th.ThreadID = x.id
	}
	return th
}

func NewMessage(ctx context.Context, ev *slackevents.EventsAPIEvent) *Message {
	switch inEv := ev.InnerEvent.Data.(type) {
	case *slackevents.AppMentionEvent:
		return &Message{
			id:       inEv.TimeStamp,
			channel:  inEv.Channel,
			threadID: inEv.ThreadTimeStamp,
			teamID:   ev.TeamID,
		}

	case *slackevents.MessageEvent:
		return &Message{
			id:       inEv.TimeStamp,
			channel:  inEv.Channel,
			threadID: inEv.ThreadTimeStamp,
			teamID:   ev.TeamID,
		}

	default:
		logging.From(ctx).Warn("unknown event type", "event", ev)
	}

	return nil
}

func (x *Message) ID() string {
	return x.id
}

func (x *Message) ChannelID() string {
	return x.channel
}

func (x *Message) ThreadID() string {
	if x.threadID == "" {
		return x.id
	}
	return x.threadID
}

func (x *Message) TeamID() string {
	return x.teamID
}

func (x *Message) InThread() bool {
	return x.threadID != ""
}

type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
