package types

import (
	"github.com/google/uuid"
)

type MemoryID string

func (x MemoryID) String() string {
	return string(x)
}

func NewMemoryID() MemoryID {
	id, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}
	return MemoryID(id.String())
}

const (
	EmptyMemoryID MemoryID = ""
)
