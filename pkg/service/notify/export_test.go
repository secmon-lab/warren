package notify

func (x *SlackThread) SetClient(client slackClient) {
	x.client = client
}
