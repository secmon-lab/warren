package user

import (
	"context"
)

type contextKey string

const (
	userIDKey  contextKey = "user_id"
	isAgentKey contextKey = "is_agent"
)

// WithUserID sets user ID in context
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// FromContext extracts user ID from context
func FromContext(ctx context.Context) string {
	if userID, ok := ctx.Value(userIDKey).(string); ok {
		return userID
	}
	return ""
}

// WithAgent marks the context as coming from an agent
func WithAgent(ctx context.Context) context.Context {
	return context.WithValue(ctx, isAgentKey, true)
}

// IsAgent checks if the context indicates an agent operation
func IsAgent(ctx context.Context) bool {
	if isAgent, ok := ctx.Value(isAgentKey).(bool); ok {
		return isAgent
	}
	return false
}
