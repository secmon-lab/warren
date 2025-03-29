package slack

type Thread struct {
	TeamID    string `json:"team_id"`
	ChannelID string `json:"channel_id"`
	ThreadID  string `json:"thread_id"`
}

type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
