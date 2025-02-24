package service

func NewDummySlackService(userID string) *Slack {
	return &Slack{
		slackMetadata: slackMetadata{
			userID: userID,
		},
	}
}
