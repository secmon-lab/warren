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
	CallbackSubmitSalvage       CallbackID = "submit_salvage"
	CallbackEditTicket          CallbackID = "edit_ticket"
)

type BlockID string

func (id BlockID) String() string {
	return string(id)
}

const (
	BlockIDTicketSelect     BlockID = "ticket_select_block"
	BlockIDTicketID         BlockID = "ticket_id_block"
	BlockIDTicketConclusion BlockID = "ticket_conclusion_block"
	BlockIDTicketComment    BlockID = "ticket_comment_block"
	BlockIDTicketTags       BlockID = "ticket_tags_block"
	BlockIDSalvageThreshold BlockID = "salvage_threshold_block"
	BlockIDSalvageKeyword   BlockID = "salvage_keyword_block"
)

// ActionID in block
type BlockActionID string

func (id BlockActionID) String() string {
	return string(id)
}

const (
	BlockActionIDTicketSelect     BlockActionID = "ticket_select_input"
	BlockActionIDTicketID         BlockActionID = "ticket_id_input"
	BlockActionIDTicketComment    BlockActionID = "ticket_comment_input"
	BlockActionIDTicketConclusion BlockActionID = "ticket_conclusion_input"
	BlockActionIDTicketTags       BlockActionID = "ticket_tags_input"
	BlockActionIDSalvageThreshold BlockActionID = "salvage_threshold_input"
	BlockActionIDSalvageKeyword   BlockActionID = "salvage_keyword_input"
	BlockActionIDSalvageRefresh   BlockActionID = "salvage_refresh_button"
	BlockActionIDTicketTitle      BlockActionID = "ticket_title_input"
	BlockActionIDTicketDesc       BlockActionID = "ticket_description_input"
)

type ActionID string

func (id ActionID) String() string {
	return string(id)
}

const (
	// For alert
	ActionIDAckAlert     ActionID = "ack_alert"
	ActionIDBindAlert    ActionID = "bind_alert"
	ActionIDDeclineAlert ActionID = "decline_alert"
	ActionIDReopenAlert  ActionID = "reopen_alert"

	// For list
	ActionIDAckList  ActionID = "ack_list"
	ActionIDBindList ActionID = "bind_list"

	// For ticket
	ActionIDResolveTicket ActionID = "resolve_ticket"
	ActionIDSalvage       ActionID = "salvage"
	ActionIDEditTicket    ActionID = "edit_ticket"

	// For notice
	ActionIDEscalate ActionID = "escalate"
)

type Mention struct {
	UserID  string
	Message string
}
