package types

import "github.com/google/uuid"

type HistoryID string

func NewHistoryID() HistoryID {
	return HistoryID(uuid.New().String())
}

func (x HistoryID) String() string {
	return string(x)
}

type MessageID string

func NewMessageID() MessageID {
	return MessageID(uuid.New().String())
}

func (x MessageID) String() string {
	return string(x)
}
