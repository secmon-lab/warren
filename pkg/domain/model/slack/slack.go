package slack

import (
	"context"
	"log/slog"

	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/slack-go/slack"
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

	user           User
	msg            string
	ts             string
	mentions       []Mention
	isTraceMessage bool
}

func (x Message) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("id", x.id),
		slog.String("channel", x.channel),
		slog.String("thread_id", x.threadID),
		slog.String("user", x.user.ID),
		slog.String("msg", x.msg),
		slog.String("ts", x.ts),
		slog.Bool("is_trace_message", x.isTraceMessage),
	)
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

func (x *Message) IsTraceMessage() bool {
	return x.isTraceMessage
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
		// Determine if this is a Trace message by checking for ContextBlock
		isTraceMessage := false
		if len(inEv.Blocks.BlockSet) > 0 {
			for _, block := range inEv.Blocks.BlockSet {
				if block.BlockType() == slack.MBTContext {
					isTraceMessage = true
					break
				}
			}
		}

		return &Message{
			id:       inEv.TimeStamp,
			channel:  inEv.Channel,
			threadID: inEv.ThreadTimeStamp,
			teamID:   ev.TeamID,
			user: User{
				ID:   inEv.User,
				Name: inEv.User,
			},
			msg:            inEv.Text,
			ts:             inEv.TimeStamp,
			isTraceMessage: isTraceMessage,
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
