package interfaces

import "errors"

// ErrSessionAlreadyExists is returned by Repository.CreateSession when a
// document already exists at the provided Session.ID. Callers (typically
// SessionResolver) recover by issuing GetSession to fetch the existing row.
var ErrSessionAlreadyExists = errors.New("session already exists")

// ErrLockNotHeld is returned by Repository.RefreshSessionLock and
// ReleaseSessionLock when the supplied holderID does not match the lock
// currently recorded on the Session (either because the holder changed or
// because the lock has been released).
var ErrLockNotHeld = errors.New("session lock not held by caller")
