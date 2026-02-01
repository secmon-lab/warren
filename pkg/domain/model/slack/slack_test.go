package slack_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/slack-go/slack/slackevents"
)

func buildMessageAPIEvent(t *testing.T, subType, user, text, threadTS, ts, channel string, innerMsg json.RawMessage) *slackevents.EventsAPIEvent {
	t.Helper()
	raw := map[string]interface{}{
		"type":     "message",
		"user":     user,
		"text":     text,
		"ts":       ts,
		"channel":  channel,
		"event_ts": ts,
	}
	if subType != "" {
		raw["subtype"] = subType
	}
	if threadTS != "" {
		raw["thread_ts"] = threadTS
	}
	if innerMsg != nil {
		raw["message"] = json.RawMessage(innerMsg)
	}

	data, err := json.Marshal(raw)
	gt.NoError(t, err)

	ev := &slackevents.MessageEvent{}
	gt.NoError(t, json.Unmarshal(data, ev))

	return &slackevents.EventsAPIEvent{
		TeamID: "T-TEAM",
		InnerEvent: slackevents.EventsAPIInnerEvent{
			Data: ev,
		},
	}
}

func TestNewMessage_NormalMessage(t *testing.T) {
	apiEvent := buildMessageAPIEvent(t,
		"",          // subType
		"U-USER001", // user
		"hello",     // text
		"1234.5678", // threadTS
		"1234.9999", // ts
		"C-CHAN001", // channel
		nil,         // no inner message
	)

	msg := slack.NewMessage(context.Background(), apiEvent)
	gt.V(t, msg).NotNil()
	gt.Value(t, msg.User().ID).Equal("U-USER001")
	gt.Value(t, msg.Text()).Equal("hello")
	gt.Value(t, msg.ThreadID()).Equal("1234.5678")
	gt.Value(t, msg.Timestamp()).Equal("1234.9999")
	gt.Value(t, msg.ChannelID()).Equal("C-CHAN001")
	gt.Value(t, msg.TeamID()).Equal("T-TEAM")
	gt.Value(t, msg.InThread()).Equal(true)
}

func TestNewMessage_MessageChanged(t *testing.T) {
	innerMsg, err := json.Marshal(map[string]interface{}{
		"user":      "U-EDITOR",
		"text":      "edited text",
		"ts":        "1234.5678",
		"thread_ts": "1234.0000",
	})
	gt.NoError(t, err)

	apiEvent := buildMessageAPIEvent(t,
		"message_changed", // subType
		"",                // user (empty for message_changed)
		"",                // text (empty for message_changed)
		"",                // threadTS (empty for message_changed)
		"1234.9999",       // event ts
		"C-CHAN001",       // channel
		innerMsg,
	)

	msg := slack.NewMessage(context.Background(), apiEvent)
	gt.V(t, msg).NotNil()
	gt.Value(t, msg.User().ID).Equal("U-EDITOR")
	gt.Value(t, msg.Text()).Equal("edited text")
	gt.Value(t, msg.Timestamp()).Equal("1234.5678")
	gt.Value(t, msg.ThreadID()).Equal("1234.0000")
	gt.Value(t, msg.ChannelID()).Equal("C-CHAN001")
	gt.Value(t, msg.TeamID()).Equal("T-TEAM")
	gt.Value(t, msg.InThread()).Equal(true)
}

func TestNewMessage_MessageChangedNilMessage(t *testing.T) {
	// message_changed with no inner message should return nil
	raw := map[string]interface{}{
		"type":     "message",
		"subtype":  "message_changed",
		"ts":       "1234.9999",
		"channel":  "C-CHAN001",
		"event_ts": "1234.9999",
	}
	data, err := json.Marshal(raw)
	gt.NoError(t, err)

	ev := &slackevents.MessageEvent{}
	gt.NoError(t, json.Unmarshal(data, ev))

	// The custom UnmarshalJSON in slack-go populates Message even for message_changed
	// by unmarshalling top-level fields. Force Message to nil for this test.
	ev.Message = nil

	apiEvent := &slackevents.EventsAPIEvent{
		TeamID: "T-TEAM",
		InnerEvent: slackevents.EventsAPIInnerEvent{
			Data: ev,
		},
	}

	msg := slack.NewMessage(context.Background(), apiEvent)
	gt.V(t, msg == nil).Equal(true)
}

func TestNewMessage_MessageDeleted(t *testing.T) {
	raw := map[string]interface{}{
		"type":       "message",
		"subtype":    "message_deleted",
		"deleted_ts": "1234.5678",
		"ts":         "1234.9999",
		"channel":    "C-CHAN001",
		"event_ts":   "1234.9999",
	}
	data, err := json.Marshal(raw)
	gt.NoError(t, err)

	ev := &slackevents.MessageEvent{}
	gt.NoError(t, json.Unmarshal(data, ev))

	apiEvent := &slackevents.EventsAPIEvent{
		TeamID: "T-TEAM",
		InnerEvent: slackevents.EventsAPIInnerEvent{
			Data: ev,
		},
	}

	msg := slack.NewMessage(context.Background(), apiEvent)
	gt.V(t, msg == nil).Equal(true)
}

func TestNewMessage_NormalMessageNotInThread(t *testing.T) {
	apiEvent := buildMessageAPIEvent(t,
		"",          // subType
		"U-USER001", // user
		"hello",     // text
		"",          // threadTS (not in thread)
		"1234.9999", // ts
		"C-CHAN001", // channel
		nil,
	)

	msg := slack.NewMessage(context.Background(), apiEvent)
	gt.V(t, msg).NotNil()
	gt.Value(t, msg.User().ID).Equal("U-USER001")
	gt.Value(t, msg.InThread()).Equal(false)
}
