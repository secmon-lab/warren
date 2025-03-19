package model

type SlackCallbackID string

func (id SlackCallbackID) String() string {
	return string(id)
}

const (
	SlackCallbackSubmitResolveAlert SlackCallbackID = "submit_resolve_alert"
	SlackCallbackSubmitResolveList  SlackCallbackID = "submit_resolve_list"
	SlackCallbackSubmitIgnoreList   SlackCallbackID = "submit_ignore_list"
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

type SlackMention struct {
	UserID string
	Args   []string
}
