package lang

import (
	"context"

	"github.com/secmon-lab/warren/pkg/model"
)

// ContextKey is a type for context keys to avoid key collisions
type contextKey struct{}

var defaultLang = model.English

// From retrieves Lang from context
func From(ctx context.Context) model.Lang {
	if v := ctx.Value(contextKey{}); v != nil {
		if lang, ok := v.(model.Lang); ok {
			return lang
		}
	}
	return defaultLang
}

// With adds Lang to context
func With(ctx context.Context, l model.Lang) context.Context {
	return context.WithValue(ctx, contextKey{}, l)
}
