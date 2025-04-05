package session

import (
	"time"

	"github.com/secmon-lab/warren/pkg/domain/types"
)

type Note struct {
	ID        types.NoteID    `json:"id"`
	SessionID types.SessionID `json:"session_id"`
	CreatedAt time.Time       `json:"created_at"`
	Note      string          `json:"note"`
}

func NewNote(sessionID types.SessionID, note string) *Note {
	return &Note{
		ID:        types.NewNoteID(),
		SessionID: sessionID,
		CreatedAt: time.Now(),
		Note:      note,
	}
}
