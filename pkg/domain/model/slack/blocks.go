package slack

import "github.com/slack-go/slack"

type BlockAction slack.BlockAction

type StateValue map[string]map[string]BlockAction

func BlockActionFromValue(value map[string]map[string]slack.BlockAction) StateValue {
	sv := make(StateValue)
	for k, v := range value {
		sv[k] = make(map[string]BlockAction)
		for k2, v2 := range v {
			sv[k][k2] = BlockAction(v2)
		}
	}
	return sv
}

type CallbackID string

func (id CallbackID) String() string {
	return string(id)
}

const (
	CallbackSubmitResolveAlert CallbackID = "submit_resolve_alert"
	CallbackSubmitResolveList  CallbackID = "submit_resolve_list"
)

type SlackBlockID string

func (id SlackBlockID) String() string {
	return string(id)
}

const (
	SlackBlockIDConclusion SlackBlockID = "conclusion"
	SlackBlockIDComment    SlackBlockID = "comment"
)

type ActionID string

func (id ActionID) String() string {
	return string(id)
}

const (
	// For alert
	ActionIDAckAlert  ActionID = "ack_alert"
	ActionIDBindAlert ActionID = "bind_alert"

	// For list
	ActionIDAckList  ActionID = "ack_list"
	ActionIDBindList ActionID = "bind_list"

	// For ticket
	ActionIDConclusion ActionID = "conclusion"
)

type Mention struct {
	UserID  string
	Message string
}
