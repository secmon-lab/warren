package session

import (
	"context"
)

// StatusCheckFunc is a function type that checks if the session is aborted
type StatusCheckFunc func(ctx context.Context) error

type contextKey string

const statusCheckKey contextKey = "session_status_check"

// WithStatusCheck sets a status check function in the context
// This function will be called by plan hooks and tool middleware to check if session is aborted
func WithStatusCheck(ctx context.Context, checkFunc StatusCheckFunc) context.Context {
	return context.WithValue(ctx, statusCheckKey, checkFunc)
}

// CheckStatus checks if the session is aborted by calling the check function from context
// Returns nil if no check function is set (graceful degradation)
// Returns error if session is aborted
func CheckStatus(ctx context.Context) error {
	checkFunc, ok := ctx.Value(statusCheckKey).(StatusCheckFunc)
	if !ok || checkFunc == nil {
		// No check function set - this is valid (e.g., CLI mode without session tracking)
		return nil
	}
	return checkFunc(ctx)
}
