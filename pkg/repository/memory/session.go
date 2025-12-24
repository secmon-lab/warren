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
	messages map[types.SessionID][]*session.Message
}

func newSessionStore() *sessionStore {
	return &sessionStore{
		sessions: make(map[types.SessionID]*session.Session),
		messages: make(map[types.SessionID][]*session.Message),
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
	delete(r.session.messages, sessionID)
	return nil
}

func (r *Memory) PutSessionMessage(ctx context.Context, message *session.Message) error {
	r.session.mu.Lock()
	defer r.session.mu.Unlock()

	// Deep copy to prevent external modification
	copied := *message
	r.session.messages[message.SessionID] = append(r.session.messages[message.SessionID], &copied)

	return nil
}

func (r *Memory) GetSessionMessages(ctx context.Context, sessionID types.SessionID) ([]*session.Message, error) {
	r.session.mu.RLock()
	defer r.session.mu.RUnlock()

	messages := r.session.messages[sessionID]
	if messages == nil {
		return []*session.Message{}, nil
	}

	// Deep copy to prevent external modification
	result := make([]*session.Message, len(messages))
	for i, msg := range messages {
		copied := *msg
		result[i] = &copied
	}

	return result, nil
}
