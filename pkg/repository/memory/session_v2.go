package memory

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

// chat-session-redesign additions on *Memory. These live in a separate file
// so the original session.go keeps a tight surface around the pre-redesign
// behavior and the new methods are easy to review as a unit.

type turnStore struct {
	mu    sync.RWMutex
	turns map[types.TurnID]*session.Turn
}

func newTurnStore() *turnStore {
	return &turnStore{turns: make(map[types.TurnID]*session.Turn)}
}

// CreateSession inserts sess only if no Session with the same ID already
// exists. Returns interfaces.ErrSessionAlreadyExists on collision.
func (r *Memory) CreateSession(ctx context.Context, sess *session.Session) error {
	r.session.mu.Lock()
	defer r.session.mu.Unlock()
	if _, exists := r.session.sessions[sess.ID]; exists {
		return interfaces.ErrSessionAlreadyExists
	}
	copied := *sess
	r.session.sessions[sess.ID] = &copied
	return nil
}

func (r *Memory) UpdateSessionLastActive(ctx context.Context, sessionID types.SessionID, t time.Time) error {
	r.session.mu.Lock()
	defer r.session.mu.Unlock()
	sess, ok := r.session.sessions[sessionID]
	if !ok {
		return goerr.New("session not found", goerr.V("session_id", sessionID))
	}
	sess.LastActiveAt = t
	return nil
}

func (r *Memory) PromoteSessionToTicket(ctx context.Context, sessionID types.SessionID, ticketID types.TicketID) error {
	r.session.mu.Lock()
	defer r.session.mu.Unlock()
	sess, ok := r.session.sessions[sessionID]
	if !ok {
		return goerr.New("session not found", goerr.V("session_id", sessionID))
	}
	sess.TicketID = ticketID
	tid := ticketID
	sess.TicketIDPtr = &tid
	return nil
}

// AcquireSessionLock attempts to take the Session.Lock for the duration ttl.
// Returns (true, nil) on success, (false, nil) if the lock is already held
// and not yet expired.
func (r *Memory) AcquireSessionLock(ctx context.Context, sessionID types.SessionID, holderID string, ttl time.Duration) (bool, error) {
	r.session.mu.Lock()
	defer r.session.mu.Unlock()
	sess, ok := r.session.sessions[sessionID]
	if !ok {
		return false, goerr.New("session not found", goerr.V("session_id", sessionID))
	}
	now := time.Now()
	if sess.Lock != nil && !sess.Lock.IsExpired(now) {
		return false, nil
	}
	sess.Lock = &session.SessionLock{
		HolderID:   holderID,
		AcquiredAt: now,
		ExpiresAt:  now.Add(ttl),
	}
	return true, nil
}

func (r *Memory) RefreshSessionLock(ctx context.Context, sessionID types.SessionID, holderID string, ttl time.Duration) error {
	r.session.mu.Lock()
	defer r.session.mu.Unlock()
	sess, ok := r.session.sessions[sessionID]
	if !ok {
		return goerr.New("session not found", goerr.V("session_id", sessionID))
	}
	if sess.Lock == nil || sess.Lock.HolderID != holderID {
		return interfaces.ErrLockNotHeld
	}
	sess.Lock.ExpiresAt = time.Now().Add(ttl)
	return nil
}

func (r *Memory) ReleaseSessionLock(ctx context.Context, sessionID types.SessionID, holderID string) error {
	r.session.mu.Lock()
	defer r.session.mu.Unlock()
	sess, ok := r.session.sessions[sessionID]
	if !ok {
		return goerr.New("session not found", goerr.V("session_id", sessionID))
	}
	if sess.Lock == nil || sess.Lock.HolderID != holderID {
		return interfaces.ErrLockNotHeld
	}
	sess.Lock = nil
	return nil
}

// --- Turn CRUD ---

func (r *Memory) PutTurn(ctx context.Context, turn *session.Turn) error {
	r.turnStore.mu.Lock()
	defer r.turnStore.mu.Unlock()
	copied := *turn
	r.turnStore.turns[turn.ID] = &copied
	return nil
}

func (r *Memory) GetTurn(ctx context.Context, turnID types.TurnID) (*session.Turn, error) {
	r.turnStore.mu.RLock()
	defer r.turnStore.mu.RUnlock()
	t, ok := r.turnStore.turns[turnID]
	if !ok {
		return nil, nil
	}
	copied := *t
	return &copied, nil
}

func (r *Memory) GetTurnsBySession(ctx context.Context, sessionID types.SessionID) ([]*session.Turn, error) {
	r.turnStore.mu.RLock()
	defer r.turnStore.mu.RUnlock()
	var out []*session.Turn
	for _, t := range r.turnStore.turns {
		if t.SessionID == sessionID {
			copied := *t
			out = append(out, &copied)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartedAt.Before(out[j].StartedAt) })
	return out, nil
}

func (r *Memory) UpdateTurnStatus(ctx context.Context, turnID types.TurnID, status session.TurnStatus, endedAt *time.Time) error {
	r.turnStore.mu.Lock()
	defer r.turnStore.mu.Unlock()
	t, ok := r.turnStore.turns[turnID]
	if !ok {
		return goerr.New("turn not found", goerr.V("turn_id", turnID))
	}
	t.Status = status
	t.EndedAt = endedAt
	return nil
}

func (r *Memory) UpdateTurnIntent(ctx context.Context, turnID types.TurnID, intent string) error {
	r.turnStore.mu.Lock()
	defer r.turnStore.mu.Unlock()
	t, ok := r.turnStore.turns[turnID]
	if !ok {
		return goerr.New("turn not found", goerr.V("turn_id", turnID))
	}
	t.Intent = intent
	return nil
}

// --- Session message queries ---

func (r *Memory) GetMessagesByTurn(ctx context.Context, turnID types.TurnID) ([]*session.Message, error) {
	r.session.mu.RLock()
	defer r.session.mu.RUnlock()
	var out []*session.Message
	for _, msgs := range r.session.messages {
		for _, m := range msgs {
			if m.TurnID != nil && *m.TurnID == turnID {
				copied := *m
				out = append(out, &copied)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (r *Memory) SearchSessionMessages(ctx context.Context, ticketID types.TicketID, query string, limit int) ([]*session.Message, error) {
	r.session.mu.RLock()
	defer r.session.mu.RUnlock()

	// Build set of SessionIDs belonging to ticketID.
	sessionIDs := map[types.SessionID]bool{}
	for _, s := range r.session.sessions {
		if sessionBelongsTo(s, ticketID) {
			sessionIDs[s.ID] = true
		}
	}

	var out []*session.Message
	needle := strings.ToLower(query)
	for sid, msgs := range r.session.messages {
		if !sessionIDs[sid] {
			continue
		}
		for _, m := range msgs {
			if needle == "" || strings.Contains(strings.ToLower(m.Content), needle) {
				copied := *m
				out = append(out, &copied)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (r *Memory) GetTicketSessionMessages(ctx context.Context, ticketID types.TicketID, source *session.SessionSource, msgType *session.MessageType, limit, offset int) ([]*session.Message, error) {
	r.session.mu.RLock()
	defer r.session.mu.RUnlock()

	// Select matching Sessions first.
	sessionIDs := map[types.SessionID]bool{}
	for _, s := range r.session.sessions {
		if !sessionBelongsTo(s, ticketID) {
			continue
		}
		if source != nil && s.Source != *source {
			continue
		}
		sessionIDs[s.ID] = true
	}

	var out []*session.Message
	for sid, msgs := range r.session.messages {
		if !sessionIDs[sid] {
			continue
		}
		for _, m := range msgs {
			if msgType != nil && m.Type != *msgType {
				continue
			}
			copied := *m
			out = append(out, &copied)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })

	// Apply offset/limit.
	if offset < 0 {
		offset = 0
	}
	if offset >= len(out) {
		return []*session.Message{}, nil
	}
	out = out[offset:]
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// AllSessions returns a snapshot copy of every Session stored in memory.
// Test-only: used by migration job tests that need to enumerate sessions
// without going through a listAll abstraction.
func (r *Memory) AllSessions() []*session.Session {
	r.session.mu.RLock()
	defer r.session.mu.RUnlock()
	out := make([]*session.Session, 0, len(r.session.sessions))
	for _, s := range r.session.sessions {
		copied := *s
		out = append(out, &copied)
	}
	return out
}

func sessionBelongsTo(s *session.Session, ticketID types.TicketID) bool {
	if s.TicketIDPtr != nil && *s.TicketIDPtr == ticketID {
		return true
	}
	return s.TicketID == ticketID
}
