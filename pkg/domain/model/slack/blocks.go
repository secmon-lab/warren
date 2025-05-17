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
	CallbackSubmitResolveTicket CallbackID = "submit_resolve_ticket"
	CallbackSubmitBindAlert     CallbackID = "submit_bind_alert"
	CallbackSubmitBindList      CallbackID = "submit_bind_list"
)

type BlockID string

func (id BlockID) String() string {
	return string(id)
}

const (
	BlockIDTicketSelect BlockID = "ticket_select_block"
	BlockIDTicketID     BlockID = "ticket_id_block"
)

// ActionID in block
type BlockActionID string

func (id BlockActionID) String() string {
	return string(id)
}

const (
	BlockActionIDTicketSelect BlockActionID = "ticket_select_input"
	BlockActionIDTicketID     BlockActionID = "ticket_id_input"
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
	ActionIDResolveTicket ActionID = "resolve_ticket"
)

type Mention struct {
	UserID  string
	Message string
}
