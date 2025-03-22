package types

import "github.com/google/uuid"

type SessionID string

func (x SessionID) String() string {
	return string(x)
}

func NewSessionID() SessionID {
	return SessionID(uuid.New().String())
}

type ProcID string

func (x ProcID) String() string {
	return string(x)
}

func NewProcID() ProcID {
	return ProcID(uuid.New().String())
}
