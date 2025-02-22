package thread

import (
	"context"
)

type ctxKey struct{}

type ReplyFunc func(ctx context.Context, msg string)

func WithReplyFunc(ctx context.Context, replyFunc ReplyFunc) context.Context {
	return context.WithValue(ctx, ctxKey{}, replyFunc)
}

func ReplyFuncFrom(ctx context.Context) ReplyFunc {
	replyFunc, ok := ctx.Value(ctxKey{}).(ReplyFunc)
	if !ok {
		return func(ctx context.Context, msg string) {}
	}
	return replyFunc
}

func Reply(ctx context.Context, msg string) {
	replyFunc, ok := ctx.Value(ctxKey{}).(ReplyFunc)
	if !ok {
		// Do nothing if the thread is not found in context
		return
	}
	replyFunc(ctx, msg)
}
