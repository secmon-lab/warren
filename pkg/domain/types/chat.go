package types

import "github.com/google/uuid"

type HistoryID string

func NewHistoryID() HistoryID {
	return HistoryID(uuid.New().String())
}

func (x HistoryID) String() string {
	return string(x)
}
