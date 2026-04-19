package types

import "github.com/google/uuid"

type MessageID string

func NewMessageID() MessageID {
	return MessageID(uuid.New().String())
}

func (x MessageID) String() string {
	return string(x)
}
