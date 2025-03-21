package notify

import "context"

type ctxKey struct{}

type SendFunc func(ctx context.Context, msg string)

func With(ctx context.Context, send SendFunc) context.Context {
	return context.WithValue(ctx, ctxKey{}, send)
}

func Send(ctx context.Context, msg string) {
	send, ok := ctx.Value(ctxKey{}).(SendFunc)
	if !ok {
		return
	}
	send(ctx, msg)
}
