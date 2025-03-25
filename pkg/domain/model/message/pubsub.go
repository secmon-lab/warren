package message

type PubSub struct {
	Message PubSubMessage `json:"message"`
}

type PubSubMessage struct {
	Data []byte `json:"data"`
}
