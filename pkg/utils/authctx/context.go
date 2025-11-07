package authctx

import "context"

type contextKey struct{}

var subjectsKey = contextKey{}

// WithSubject injects Subject into context
// Multiple subjects can be added (e.g., both IAP and Slack authentication)
// Returns a new context with the subject added, without modifying the original
func WithSubject(ctx context.Context, subject Subject) context.Context {
	subjects, _ := ctx.Value(subjectsKey).([]Subject)
	// Create a new slice with capacity for one more element to ensure immutability
	newSubjects := make([]Subject, len(subjects)+1)
	copy(newSubjects, subjects)
	newSubjects[len(subjects)] = subject
	return context.WithValue(ctx, subjectsKey, newSubjects)
}

// GetSubjects retrieves all Subjects from context
// Returns a copy of the slice containing all authenticated subjects
// If no subjects found, returns empty slice (non-nil)
// The returned slice is a copy to ensure immutability
func GetSubjects(ctx context.Context) []Subject {
	subjects, ok := ctx.Value(subjectsKey).([]Subject)
	if !ok || subjects == nil {
		return make([]Subject, 0)
	}
	// Return a copy to prevent external modification
	result := make([]Subject, len(subjects))
	copy(result, subjects)
	return result
}
