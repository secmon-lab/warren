package session

import "time"

// SessionLock is embedded in a Session.Lock field to express ownership of a
// Slack Session by a specific in-flight request. When Lock is nil the
// Session is unlocked.
//
// SessionLock is the persistent state; the runtime "Lock" handle returned by
// the LockService (see pkg/service/session) is a separate type that holds
// unexported dependencies (repo client, clock) and exposes Refresh/Release.
type SessionLock struct {
	// HolderID is the request_id of the goroutine holding the lock.
	HolderID string `firestore:"holder_id" json:"holder_id"`
	// AcquiredAt is the instant the lock was first granted.
	AcquiredAt time.Time `firestore:"acquired_at" json:"acquired_at"`
	// ExpiresAt is AcquiredAt + TTL (default 3 minutes). The lock is
	// considered stale once ExpiresAt <= now, at which point another
	// holder may take over.
	ExpiresAt time.Time `firestore:"expires_at" json:"expires_at"`
}

// IsExpired reports whether the lock has passed its TTL relative to now.
func (l *SessionLock) IsExpired(now time.Time) bool {
	if l == nil {
		return true
	}
	return !now.Before(l.ExpiresAt)
}
