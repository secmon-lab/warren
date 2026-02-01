package slack_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	slackctrl "github.com/secmon-lab/warren/pkg/controller/slack"
	goslack "github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

func TestResolveMessageEventUserID_NormalMessage(t *testing.T) {
	ev := &slackevents.MessageEvent{
		User: "U-USER001",
	}
	gt.Value(t, slackctrl.ResolveMessageEventUserID(ev)).Equal("U-USER001")
}

func TestResolveMessageEventUserID_MessageChanged(t *testing.T) {
	ev := &slackevents.MessageEvent{
		User:    "",
		SubType: "message_changed",
		Message: &goslack.Msg{
			User: "U-EDITOR",
		},
	}
	gt.Value(t, slackctrl.ResolveMessageEventUserID(ev)).Equal("U-EDITOR")
}

func TestResolveMessageEventUserID_MessageChangedNilMessage(t *testing.T) {
	ev := &slackevents.MessageEvent{
		User:    "",
		SubType: "message_changed",
		Message: nil,
	}
	gt.Value(t, slackctrl.ResolveMessageEventUserID(ev)).Equal("")
}

func TestResolveMessageEventUserID_MessageDeleted(t *testing.T) {
	ev := &slackevents.MessageEvent{
		User:    "",
		SubType: "message_deleted",
	}
	gt.Value(t, slackctrl.ResolveMessageEventUserID(ev)).Equal("")
}

func TestResolveMessageEventUserID_UserTakesPrecedence(t *testing.T) {
	// If top-level User is set, it should be used even with message_changed
	ev := &slackevents.MessageEvent{
		User:    "U-TOPLEVEL",
		SubType: "message_changed",
		Message: &goslack.Msg{
			User: "U-NESTED",
		},
	}
	gt.Value(t, slackctrl.ResolveMessageEventUserID(ev)).Equal("U-TOPLEVEL")
}
