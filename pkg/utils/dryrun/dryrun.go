package dryrun

import (
	"context"
)

type contextKey string

const (
	// contextKeyDryRun is the context key for dry-run flag
	contextKeyDryRun contextKey = "dry_run"
)

// With sets dry-run flag in the context
func With(ctx context.Context, dryRun bool) context.Context {
	return context.WithValue(ctx, contextKeyDryRun, dryRun)
}

// From gets dry-run flag from the context
func From(ctx context.Context) bool {
	if value, ok := ctx.Value(contextKeyDryRun).(bool); ok {
		return value
	}
	return false
}

// IsDryRun checks if the context has dry-run flag set to true
func IsDryRun(ctx context.Context) bool {
	return From(ctx)
}
