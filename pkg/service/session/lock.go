// Package session provides service-layer helpers on top of the
// session.Session model: activity lock service, session resolver, etc.
//
// The chat-session-redesign spec mandates that lock state live in the
// Session document itself (Session.Lock) so the pre-existing Session-level
// transactions double as lock transactions. LockService is a thin facade
// over the Repository's AcquireSessionLock / RefreshSessionLock /
// ReleaseSessionLock methods that returns an ergonomic *Lock handle.
package session

import (
	"context"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/request_id"
)

// DefaultLockTTL is the TTL applied to Slack Session activity locks when a
// caller does not specify one explicitly. Three minutes matches the spec
// requirement of roughly matching a user's patience for a stuck chat while
// still allowing crash recovery to kick in within a short window.
const DefaultLockTTL = 3 * time.Minute

// LockService manufactures Lock handles by acquiring Session activity locks
// via the Repository.
type LockService struct {
	repo interfaces.Repository
	ttl  time.Duration
}

// LockServiceOption configures optional behavior on the LockService.
type LockServiceOption func(*LockService)

// WithLockTTL overrides the default 3-minute TTL used by TryAcquire.
func WithLockTTL(ttl time.Duration) LockServiceOption {
	return func(l *LockService) { l.ttl = ttl }
}

// NewLockService constructs a LockService bound to repo.
func NewLockService(repo interfaces.Repository, opts ...LockServiceOption) *LockService {
	s := &LockService{repo: repo, ttl: DefaultLockTTL}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// TryAcquire attempts to grab the activity lock for sessionID. On success,
// the returned *Lock must eventually be passed to Release (typically via
// defer). On contention (lock already held and not expired), returns
// (nil, false, nil). Returns error only on infrastructure failure.
//
// holderID is taken from ctx (request_id). If no request_id is present,
// TryAcquire returns an error — an anonymous lock would make diagnosis
// impossible.
func (s *LockService) TryAcquire(ctx context.Context, sessionID types.SessionID) (*Lock, bool, error) {
	holder := request_id.FromContext(ctx)
	if holder == "" || holder == "(unknown)" {
		return nil, false, goerr.New("TryAcquire requires a request_id in context")
	}
	acquired, err := s.repo.AcquireSessionLock(ctx, sessionID, holder, s.ttl)
	if err != nil {
		return nil, false, goerr.Wrap(err, "failed to acquire session lock",
			goerr.V("session_id", sessionID))
	}
	if !acquired {
		return nil, false, nil
	}
	return &Lock{
		sessionID: sessionID,
		holderID:  holder,
		acquired:  clock.Now(ctx),
		expires:   clock.Now(ctx).Add(s.ttl),
		ttl:       s.ttl,
		repo:      s.repo,
	}, true, nil
}

// Lock is a runtime handle to an acquired Session activity lock. The
// persistent state lives in Session.Lock (Firestore / memory); the handle
// only carries the identifiers needed to refresh/release and so that test
// code can inspect the holder.
type Lock struct {
	sessionID types.SessionID
	holderID  string
	acquired  time.Time
	expires   time.Time
	ttl       time.Duration
	repo      interfaces.Repository
}

// SessionID returns the Session the lock protects.
func (l *Lock) SessionID() types.SessionID { return l.sessionID }

// HolderID returns the request_id of the current holder.
func (l *Lock) HolderID() string { return l.holderID }

// ExpiresAt returns the current expires_at snapshot (only updated by
// Refresh).
func (l *Lock) ExpiresAt() time.Time { return l.expires }

// Refresh extends the lock's ExpiresAt by its configured TTL.
func (l *Lock) Refresh(ctx context.Context) error {
	if err := l.repo.RefreshSessionLock(ctx, l.sessionID, l.holderID, l.ttl); err != nil {
		return goerr.Wrap(err, "failed to refresh session lock",
			goerr.V("session_id", l.sessionID))
	}
	l.expires = clock.Now(ctx).Add(l.ttl)
	return nil
}

// Release relinquishes the lock so the next TryAcquire succeeds
// immediately (without waiting for TTL expiry).
func (l *Lock) Release(ctx context.Context) error {
	if err := l.repo.ReleaseSessionLock(ctx, l.sessionID, l.holderID); err != nil {
		return goerr.Wrap(err, "failed to release session lock",
			goerr.V("session_id", l.sessionID))
	}
	return nil
}
