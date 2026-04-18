package types

import "github.com/google/uuid"

// TurnID uniquely identifies a single AI request/response cycle inside a
// Session. One Session may contain many Turns (Slack thread with many
// mentions, CLI interactive mode, etc.). See the chat-session-redesign spec
// for the full lifecycle.
type TurnID string

// NewTurnID generates a fresh random TurnID.
func NewTurnID() TurnID {
	return TurnID(uuid.New().String())
}

func (x TurnID) String() string { return string(x) }
