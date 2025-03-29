package slack

import "context"

type ctxSyncKey struct{}

func WithSync(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxSyncKey{}, true)
}

func IsSync(ctx context.Context) bool {
	return ctx.Value(ctxSyncKey{}).(bool)
}
