package types

import "github.com/google/uuid"

// SessionID represents a unique chat session identifier
type SessionID string

// NewSessionID generates a new session ID
func NewSessionID() SessionID {
	return SessionID(uuid.New().String())
}

func (x SessionID) String() string {
	return string(x)
}

// SessionStatus represents the status of a chat session
type SessionStatus string

const (
	SessionStatusRunning   SessionStatus = "running"
	SessionStatusAborted   SessionStatus = "aborted"
	SessionStatusCompleted SessionStatus = "completed"
)

func (x SessionStatus) String() string {
	return string(x)
}

// SessionRecordID represents a unique session record identifier
type SessionRecordID string

// NewSessionRecordID generates a new session record ID
func NewSessionRecordID() SessionRecordID {
	return SessionRecordID(uuid.New().String())
}

func (x SessionRecordID) String() string {
	return string(x)
}
