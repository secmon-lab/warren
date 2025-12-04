package memory

import (
	"context"
	"sync"

	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

type sessionStore struct {
	mu       sync.RWMutex
	sessions map[types.SessionID]*session.Session
}

func newSessionStore() *sessionStore {
	return &sessionStore{
		sessions: make(map[types.SessionID]*session.Session),
	}
}

func (r *Memory) PutSession(ctx context.Context, sess *session.Session) error {
	r.session.mu.Lock()
	defer r.session.mu.Unlock()

	// Deep copy to prevent external modification
	copied := *sess
	r.session.sessions[sess.ID] = &copied

	return nil
}

func (r *Memory) GetSession(ctx context.Context, sessionID types.SessionID) (*session.Session, error) {
	r.session.mu.RLock()
	defer r.session.mu.RUnlock()

	sess, ok := r.session.sessions[sessionID]
	if !ok {
		return nil, nil
	}

	// Deep copy to prevent external modification
	copied := *sess
	return &copied, nil
}

func (r *Memory) GetSessionsByTicket(ctx context.Context, ticketID types.TicketID) ([]*session.Session, error) {
	r.session.mu.RLock()
	defer r.session.mu.RUnlock()

	var sessions []*session.Session
	for _, sess := range r.session.sessions {
		if sess.TicketID == ticketID {
			// Deep copy to prevent external modification
			copied := *sess
			sessions = append(sessions, &copied)
		}
	}

	return sessions, nil
}

func (r *Memory) DeleteSession(ctx context.Context, sessionID types.SessionID) error {
	r.session.mu.Lock()
	defer r.session.mu.Unlock()

	delete(r.session.sessions, sessionID)
	return nil
}
