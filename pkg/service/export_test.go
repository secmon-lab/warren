package service

func NewDummySlackService(userID string) *Slack {
	return &Slack{
		userID: userID,
	}
}
