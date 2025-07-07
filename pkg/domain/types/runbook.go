package types

import (
	"github.com/google/uuid"
)

type RunbookID string

func (x RunbookID) String() string {
	return string(x)
}

func NewRunbookID() RunbookID {
	return RunbookID(uuid.New().String())
}
