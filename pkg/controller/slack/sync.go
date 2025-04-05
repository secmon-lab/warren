package slack

import "context"

type ctxSyncKey struct{}

func WithSync(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxSyncKey{}, true)
}

func IsSync(ctx context.Context) bool {
	v := ctx.Value(ctxSyncKey{})
	if v == nil {
		return false
	}
	return v.(bool)
}
