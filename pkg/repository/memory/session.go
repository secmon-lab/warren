package memory

import (
	"context"
	"sync"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

type sessionStore struct {
	mu          sync.RWMutex
	sessions    map[types.SessionID]*session.Session
	ticketIndex map[types.TicketID]types.SessionID // ticketID -> latest running sessionID
}

func newSessionStore() *sessionStore {
	return &sessionStore{
		sessions:    make(map[types.SessionID]*session.Session),
		ticketIndex: make(map[types.TicketID]types.SessionID),
	}
}

func (r *Memory) PutSession(ctx context.Context, sess *session.Session) error {
	r.session.mu.Lock()
	defer r.session.mu.Unlock()

	// Deep copy to prevent external modification
	copied := *sess
	r.session.sessions[sess.ID] = &copied

	// Update ticket index if session is running
	if sess.Status == types.SessionStatusRunning {
		r.session.ticketIndex[sess.TicketID] = sess.ID
	} else {
		// Remove from index if session is not running
		if currentID, ok := r.session.ticketIndex[sess.TicketID]; ok && currentID == sess.ID {
			delete(r.session.ticketIndex, sess.TicketID)
		}
	}

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

func (r *Memory) GetSessionByTicket(ctx context.Context, ticketID types.TicketID) (*session.Session, error) {
	r.session.mu.RLock()
	defer r.session.mu.RUnlock()

	// Get latest running session ID from index
	sessionID, ok := r.session.ticketIndex[ticketID]
	if !ok {
		return nil, nil
	}

	sess, ok := r.session.sessions[sessionID]
	if !ok {
		return nil, nil
	}

	// Double-check that session is still running
	if sess.Status != types.SessionStatusRunning {
		return nil, nil
	}

	// Deep copy to prevent external modification
	copied := *sess
	return &copied, nil
}

func (r *Memory) DeleteSession(ctx context.Context, sessionID types.SessionID) error {
	r.session.mu.Lock()
	defer r.session.mu.Unlock()

	sess, ok := r.session.sessions[sessionID]
	if !ok {
		return goerr.New("session not found",
			goerr.V("session_id", sessionID),
			goerr.T(errs.TagNotFound))
	}

	// Remove from ticket index if it's the current session
	if currentID, ok := r.session.ticketIndex[sess.TicketID]; ok && currentID == sessionID {
		delete(r.session.ticketIndex, sess.TicketID)
	}

	delete(r.session.sessions, sessionID)
	return nil
}
