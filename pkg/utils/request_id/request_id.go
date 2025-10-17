package request_id

import (
	"context"

	"github.com/google/uuid"
)

type contextKey string

const (
	requestIDKey contextKey = "request_id"
)

// With sets request ID in context
func With(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// FromContext extracts request ID from context
func FromContext(ctx context.Context) string {
	if requestID, ok := ctx.Value(requestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// Generate generates a new request ID and sets it in context
func Generate(ctx context.Context) (context.Context, string) {
	requestID := uuid.New().String()
	return With(ctx, requestID), requestID
}
