package notify

import "context"

type ctxKey struct{}

func WithThread(ctx context.Context, thread *SlackThread) context.Context {
	return context.WithValue(ctx, ctxKey{}, thread)
}

func Reply(ctx context.Context, msg string) {
	thread, ok := ctx.Value(ctxKey{}).(*SlackThread)
	if !ok {
		return
	}
	thread.Reply(ctx, msg)
}

func NewMessageContext(ctx context.Context, base string) *MessageContext {
	thread, ok := ctx.Value(ctxKey{}).(*SlackThread)
	if !ok {
		return newDummyMessageContext()
	}
	return thread.NewMessageContext(ctx, base)
}
