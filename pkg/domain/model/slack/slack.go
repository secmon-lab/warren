package slack

import (
	"context"
	"strings"

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
	teamID   string

	user     User
	msg      string
	ts       string
	mentions []Mention
}

func (x *Message) Thread() Thread {
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

func (x *Message) ID() string {
	return x.id
}

func (x *Message) Mention() []Mention {
	return x.mentions
}

func (x *Message) User() *User {
	if x.user.ID == "" {
		return nil
	}
	return &x.user
}

func (x *Message) Text() string {
	return x.msg
}

func (x *Message) Timestamp() string {
	return x.ts
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

// SlackURL returns the Slack message URL
// Format: https://slack.com/archives/{channelID}/p{timestamp}
func (x *Message) SlackURL() string {
	if x.channel == "" || x.ts == "" {
		return ""
	}

	// Convert timestamp format: "1234567890.123456" -> "1234567890123456"
	ts := strings.ReplaceAll(x.ts, ".", "")

	return "https://slack.com/archives/" + x.channel + "/p" + ts
}

func NewMessage(ctx context.Context, ev *slackevents.EventsAPIEvent) *Message {
	switch inEv := ev.InnerEvent.Data.(type) {
	case *slackevents.AppMentionEvent:
		return &Message{
			id:       inEv.TimeStamp,
			channel:  inEv.Channel,
			threadID: inEv.ThreadTimeStamp,
			teamID:   ev.TeamID,
			user: User{
				ID:   inEv.User,
				Name: inEv.User, // TODO: get user name
			},
			msg:      inEv.Text,
			ts:       inEv.TimeStamp,
			mentions: ParseMention(inEv.Text),
		}

	case *slackevents.MessageEvent:
		return &Message{
			id:       inEv.TimeStamp,
			channel:  inEv.Channel,
			threadID: inEv.ThreadTimeStamp,
			teamID:   ev.TeamID,
			user: User{
				ID:   inEv.User,
				Name: inEv.User,
			},
			msg: inEv.Text,
			ts:  inEv.TimeStamp,
		}

	default:
		logging.From(ctx).Warn("unknown event type", "event", ev)
	}

	return nil
}

type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
