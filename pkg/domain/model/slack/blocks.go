package slack

import "github.com/slack-go/slack"

type ActionID string

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
	CallbackSubmitIgnoreList   CallbackID = "submit_ignore_list"
)

type SlackBlockID string

func (id SlackBlockID) String() string {
	return string(id)
}

const (
	SlackBlockIDConclusion   SlackBlockID = "conclusion"
	SlackBlockIDComment      SlackBlockID = "comment"
	SlackBlockIDIgnorePrompt SlackBlockID = "ignore_prompt"
)

type SlackActionID string

func (id SlackActionID) String() string {
	return string(id)
}

const (
	SlackActionIDAck         SlackActionID = "ack"
	SlackActionIDResolve     SlackActionID = "resolve"
	SlackActionIDInspect     SlackActionID = "inspect"
	SlackActionIDCreatePR    SlackActionID = "create_pr"
	SlackActionIDIgnore      SlackActionID = "ignore"
	SlackActionIDIgnoreList  SlackActionID = "ignore_list"
	SlackActionIDResolveList SlackActionID = "resolve_list"

	SlackActionIDConclusion   SlackActionID = "conclusion"
	SlackActionIDComment      SlackActionID = "comment"
	SlackActionIDIgnorePrompt SlackActionID = "ignore_prompt"
)

type Mention struct {
	UserID string
	Args   []string
}
