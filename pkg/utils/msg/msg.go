package msg

import (
	"context"
	"fmt"
)

type NotifyFunc func(ctx context.Context, msg string)
type NewTraceFunc func(ctx context.Context, msg string) func(ctx context.Context, msg string)

type ctxNotifyFuncKey struct{}
type ctxNewTraceFuncKey struct{}
type ctxTraceFuncKey struct{}

func With(ctx context.Context, NotifyFunc NotifyFunc, NewTraceFunc NewTraceFunc) context.Context {
	ctx = context.WithValue(ctx, ctxNotifyFuncKey{}, NotifyFunc)
	ctx = context.WithValue(ctx, ctxNewTraceFuncKey{}, NewTraceFunc)
	return ctx
}

func Notify(ctx context.Context, format string, args ...any) {
	if v := ctx.Value(ctxNotifyFuncKey{}); v != nil {
		if NotifyFunc, ok := v.(NotifyFunc); ok && NotifyFunc != nil {
			NotifyFunc(ctx, fmt.Sprintf(format, args...))
			return
		}
	}
}

func NewTrace(ctx context.Context, format string, args ...any) context.Context {
	if v := ctx.Value(ctxNewTraceFuncKey{}); v != nil {
		if fn, ok := v.(NewTraceFunc); ok && fn != nil {
			TraceMsg := fn(ctx, fmt.Sprintf(format, args...))
			return context.WithValue(ctx, ctxTraceFuncKey{}, TraceMsg)
		}
	}
	return context.WithValue(ctx, ctxTraceFuncKey{}, func(ctx context.Context, msg string) {})
}

func Trace(ctx context.Context, base string, args ...any) context.Context {
	msg := fmt.Sprintf(base, args...)

	// If there is already a Trace function, execute it
	if v := ctx.Value(ctxTraceFuncKey{}); v != nil {
		if TraceMsg, ok := v.(func(ctx context.Context, msg string)); ok && TraceMsg != nil {
			TraceMsg(ctx, msg)
			return ctx
		}
	}

	return NewTrace(ctx, base, args...)
}

func WithContext(original context.Context) context.Context {
	ctx := original
	ctx = context.WithValue(ctx, ctxNotifyFuncKey{}, original.Value(ctxNotifyFuncKey{}))
	ctx = context.WithValue(ctx, ctxNewTraceFuncKey{}, original.Value(ctxNewTraceFuncKey{}))
	ctx = context.WithValue(ctx, ctxTraceFuncKey{}, original.Value(ctxTraceFuncKey{}))
	return ctx
}
