package usecase

/*
func newBackgroundContext(ctx context.Context) context.Context {
	newCtx := context.Background()

	newCtx = logging.With(newCtx, logging.From(ctx))
	newCtx = thread.WithReplyFunc(newCtx, thread.ReplyFuncFrom(ctx))
	newCtx = lang.With(newCtx, lang.From(ctx))

	return newCtx
}

type ctxKeySync struct{}

var syncKey = ctxKeySync{}

func WithSync(ctx context.Context, sync bool) context.Context {
	return context.WithValue(ctx, syncKey, sync)
}

func IsSync(ctx context.Context) bool {
	return ctx.Value(syncKey).(bool)
}
*/
