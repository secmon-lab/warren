package model

import "context"

type ctxSyncKey struct{}

func WithSync(ctx context.Context, sync bool) context.Context {
	return context.WithValue(ctx, ctxSyncKey{}, sync)
}

func IsSync(ctx context.Context) bool {
	sync, ok := ctx.Value(ctxSyncKey{}).(bool)
	if !ok {
		return false
	}
	return sync
}
