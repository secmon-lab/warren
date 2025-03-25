package msg

import (
	"context"
	"fmt"
)

type ReplyFunc func(ctx context.Context, msg string)
type NewStateFunc func(ctx context.Context, msg string) func(ctx context.Context, msg string)

type ctxReplyFuncKey struct{}
type ctxNewStateFuncKey struct{}
type ctxStateFuncKey struct{}

func With(ctx context.Context, replyFunc ReplyFunc, newStateFunc NewStateFunc) context.Context {
	ctx = context.WithValue(ctx, ctxReplyFuncKey{}, replyFunc)
	ctx = context.WithValue(ctx, ctxNewStateFuncKey{}, newStateFunc)
	return ctx
}

func Reply(ctx context.Context, format string, args ...any) {
	replyFunc := ctx.Value(ctxReplyFuncKey{}).(ReplyFunc)
	if replyFunc == nil {
		return
	}
	replyFunc(ctx, fmt.Sprintf(format, args...))
}

func NewState(ctx context.Context, format string, args ...any) context.Context {
	if v := ctx.Value(ctxNewStateFuncKey{}); v != nil {
		if fn, ok := v.(NewStateFunc); ok && fn != nil {
			stateMsg := fn(ctx, fmt.Sprintf(format, args...))
			return context.WithValue(ctx, ctxStateFuncKey{}, stateMsg)
		}
	}
	return context.WithValue(ctx, ctxStateFuncKey{}, func(ctx context.Context, msg string) {})
}

func State(ctx context.Context, base string, args ...any) context.Context {
	msg := fmt.Sprintf(base, args...)

	// If there is already a state function, execute it
	if v := ctx.Value(ctxStateFuncKey{}); v != nil {
		if stateMsg, ok := v.(func(ctx context.Context, msg string)); ok && stateMsg != nil {
			stateMsg(ctx, msg)
			return ctx
		}
	}

	// If there is a new state function, create a new state function
	if v := ctx.Value(ctxNewStateFuncKey{}); v != nil {
		if newState, ok := v.(NewStateFunc); ok && newState != nil {
			stateMsg := newState(ctx, msg)
			return context.WithValue(ctx, ctxStateFuncKey{}, stateMsg)
		}
	}

	// If there is no state function, nothing to do and return the context as is
	return ctx
}

func WithContext(original context.Context) context.Context {
	ctx := original
	ctx = context.WithValue(ctx, ctxReplyFuncKey{}, original.Value(ctxReplyFuncKey{}))
	ctx = context.WithValue(ctx, ctxNewStateFuncKey{}, original.Value(ctxNewStateFuncKey{}))
	ctx = context.WithValue(ctx, ctxStateFuncKey{}, original.Value(ctxStateFuncKey{}))
	return ctx
}
