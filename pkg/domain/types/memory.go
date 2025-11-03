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

type AgentMemoryID string

func (x AgentMemoryID) String() string {
	return string(x)
}

func NewAgentMemoryID() AgentMemoryID {
	id, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}
	return AgentMemoryID(id.String())
}

const (
	EmptyAgentMemoryID AgentMemoryID = ""
)
