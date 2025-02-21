package thread

import (
	"context"

	"github.com/secmon-lab/warren/pkg/interfaces"
)

type ctxKey struct{}

func With(ctx context.Context, thread interfaces.SlackThreadService) context.Context {
	return context.WithValue(ctx, ctxKey{}, thread)
}

func Reply(ctx context.Context, msg string) {
	thread, ok := ctx.Value(ctxKey{}).(interfaces.SlackThreadService)
	if !ok {
		// Do nothing if the thread is not found in context
		return
	}
	thread.Reply(ctx, msg)
}
