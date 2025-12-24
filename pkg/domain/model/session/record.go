package session

import (
	"context"
	"time"

	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
)

// SessionRecord represents a record of messages or plans during a chat session
type SessionRecord struct {
	ID        types.SessionRecordID `firestore:"id" json:"id"`
	SessionID types.SessionID       `firestore:"session_id" json:"session_id"`
	Content   string                `firestore:"content" json:"content"`
	CreatedAt time.Time             `firestore:"created_at" json:"created_at"`
}

// NewSessionRecord creates a new session record
func NewSessionRecord(ctx context.Context, sessionID types.SessionID, content string) *SessionRecord {
	return &SessionRecord{
		ID:        types.NewSessionRecordID(),
		SessionID: sessionID,
		Content:   content,
		CreatedAt: clock.Now(ctx),
	}
}
